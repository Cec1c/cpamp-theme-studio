package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Options struct {
	BootstrapVersion string
	Service          string
	ManagerService   string
	ConfigPath       string
	PanelPath        string
	PanelURL         string
	PluginsDir       string
	HealthURL        string
	StatePath        string
	StateDir         string
	InstalledBinary  string
	SystemdDir       string
	TmpfilesPath     string
	RuntimeDirectory string
	Executable       string
	RequireRoot      bool
}

type Plan struct {
	Options           Options
	CPA               ServiceInfo
	Manager           *ServiceInfo
	ConfigRaw         []byte
	ConfigPath        string
	ConfigInfo        ConfigInfo
	PatchedConfig     []byte
	PanelPath         string
	PanelSeed         []byte
	ExternalizePanel  bool
	ManagerDropInPath string
	PluginsDir        string
	PluginPlatformDir string
	HealthURL         string
	ServiceUser       string
	ServiceGroup      string
	ServiceUID        int
	ServiceGID        int
}

type Engine struct {
	Systemd Systemd
	Client  *http.Client
	Now     func() time.Time
}

func (engine Engine) Resolve(ctx context.Context, rawOptions Options) (Plan, error) {
	options := normalizeOptions(rawOptions)
	if errPaths := validateOptionPaths(options); errPaths != nil {
		return Plan{}, errPaths
	}
	if runtime.GOOS != "linux" && options.RequireRoot {
		return Plan{}, fmt.Errorf("bootstrap apply is supported only on Linux")
	}
	if options.RequireRoot && effectiveUID() != 0 {
		return Plan{}, fmt.Errorf("bootstrap apply requires root; rerun with sudo")
	}
	systemd := engine.Systemd
	var cpa ServiceInfo
	var manager *ServiceInfo
	if options.Service == "" {
		resolvedCPA, resolvedManager, errFind := systemd.FindServices()
		if errFind != nil {
			return Plan{}, errFind
		}
		cpa, manager = resolvedCPA, resolvedManager
	} else {
		resolvedCPA, errCPA := systemd.Service(options.Service)
		if errCPA != nil {
			return Plan{}, errCPA
		}
		cpa = resolvedCPA
		if options.ManagerService != "" {
			resolvedManager, errManager := systemd.Service(options.ManagerService)
			if errManager != nil {
				return Plan{}, errManager
			}
			manager = &resolvedManager
		}
	}
	if options.ManagerService != "" && (manager == nil || manager.Unit != options.ManagerService) {
		resolvedManager, errManager := systemd.Service(options.ManagerService)
		if errManager != nil {
			return Plan{}, errManager
		}
		manager = &resolvedManager
	}

	configPath := strings.TrimSpace(options.ConfigPath)
	if configPath == "" {
		configPath = ConfigPathFromProcess(cpa)
	}
	configPath = absoluteFrom(cpa.WorkingDirectory, configPath)
	if configPath == "" || !regularFile(configPath) {
		return Plan{}, fmt.Errorf("could not resolve a regular non-symlink CPA config; pass --config")
	}
	if canonical, errCanonical := filepath.EvalSymlinks(configPath); errCanonical == nil {
		configPath = filepath.Clean(canonical)
	}
	configRaw, errConfig := os.ReadFile(configPath)
	if errConfig != nil {
		return Plan{}, fmt.Errorf("read CPA config: %w", errConfig)
	}
	configInfo, errInspect := InspectConfig(configRaw, cpa.WorkingDirectory)
	if errInspect != nil {
		return Plan{}, errInspect
	}
	pluginsDir := strings.TrimSpace(options.PluginsDir)
	if pluginsDir == "" {
		pluginsDir = configInfo.PluginsDir
	}
	pluginsDir = absoluteFrom(cpa.WorkingDirectory, pluginsDir)
	if pluginsDir == "" || !filepath.IsAbs(pluginsDir) || isFilesystemRoot(pluginsDir) {
		return Plan{}, fmt.Errorf("could not resolve an absolute plugin directory")
	}
	if info, errInfo := os.Lstat(pluginsDir); errInfo == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return Plan{}, fmt.Errorf("plugin directory is not a regular non-symlink directory: %s", pluginsDir)
		}
		if canonical, errCanonical := filepath.EvalSymlinks(pluginsDir); errCanonical == nil {
			pluginsDir = filepath.Clean(canonical)
		}
	} else if !errors.Is(errInfo, os.ErrNotExist) {
		return Plan{}, fmt.Errorf("inspect plugin directory: %w", errInfo)
	}
	panelPath, panelSeed, externalize, errPanel := engine.resolvePanel(ctx, options, cpa, manager, configPath)
	if errPanel != nil {
		return Plan{}, errPanel
	}
	patched, patchedInfo, errPatch := PatchConfig(configRaw, ConfigPatch{
		PluginsDir: pluginsDir, PanelPath: panelPath, ConfigPath: configPath,
		RestartMode: "broker", RestartService: cpa.Unit,
	})
	if errPatch != nil {
		return Plan{}, errPatch
	}
	serviceUser, serviceGroup, uid, gid, errIdentity := resolveServiceIdentity(cpa)
	if errIdentity != nil {
		return Plan{}, errIdentity
	}
	healthURL := strings.TrimRight(strings.TrimSpace(options.HealthURL), "/")
	if healthURL == "" {
		healthURL = fmt.Sprintf("http://127.0.0.1:%d", patchedInfo.Port)
	}
	if errURL := validateHTTPURL(healthURL, "health URL"); errURL != nil {
		return Plan{}, errURL
	}
	if strings.TrimSpace(options.PanelURL) != "" {
		if errURL := validateHTTPURL(options.PanelURL, "panel URL"); errURL != nil {
			return Plan{}, errURL
		}
	}
	managerDropIn := ""
	if manager != nil && (externalize || filepath.Clean(PanelPathFromManager(*manager)) != filepath.Clean(panelPath)) {
		managerDropIn = filepath.Join(options.SystemdDir, manager.Unit+".d", "90-cpamp-theme-studio-panel.conf")
	}
	return Plan{
		Options: options, CPA: cpa, Manager: manager, ConfigRaw: configRaw, ConfigPath: configPath, ConfigInfo: patchedInfo,
		PatchedConfig: patched, PanelPath: panelPath, PanelSeed: panelSeed, ExternalizePanel: externalize,
		ManagerDropInPath: managerDropIn, PluginsDir: pluginsDir,
		PluginPlatformDir: filepath.Join(pluginsDir, "linux", runtime.GOARCH), HealthURL: healthURL,
		ServiceUser: serviceUser, ServiceGroup: serviceGroup, ServiceUID: uid, ServiceGID: gid,
	}, nil
}

func (engine Engine) Apply(ctx context.Context, plan Plan) (State, error) {
	options := plan.Options
	if options.RequireRoot && effectiveUID() != 0 {
		return State{}, fmt.Errorf("bootstrap apply requires root")
	}
	if options.Executable == "" || !regularFile(options.Executable) {
		return State{}, fmt.Errorf("bootstrap executable is not a regular file: %s", options.Executable)
	}
	pathUnitPath := filepath.Join(options.SystemdDir, DefaultPathUnit)
	timerUnitPath := filepath.Join(options.SystemdDir, DefaultTimerUnit)
	serviceUnitPath := filepath.Join(options.SystemdDir, DefaultServiceUnit)
	backupPaths := []string{plan.Options.StatePath, plan.Options.InstalledBinary, plan.Options.TmpfilesPath, pathUnitPath, timerUnitPath, serviceUnitPath, plan.ConfigPath, plan.PanelPath}
	if plan.ManagerDropInPath != "" {
		backupPaths = append(backupPaths, plan.ManagerDropInPath)
	}
	backup, errBackup := CreateBackup(options.StateDir, plan.CPA.Unit, managerUnit(plan.Manager), backupPaths, engine.now())
	if errBackup != nil {
		return State{}, errBackup
	}
	rollback := func(cause error) (State, error) {
		var rollbackErrors []string
		if errDisable := engine.Systemd.DisablePath(DefaultPathUnit); errDisable != nil {
			rollbackErrors = append(rollbackErrors, errDisable.Error())
		}
		if errDisable := engine.Systemd.DisablePath(DefaultTimerUnit); errDisable != nil {
			rollbackErrors = append(rollbackErrors, errDisable.Error())
		}
		errRestore := RestoreBackup(options.StateDir, backup)
		if errRestore != nil {
			rollbackErrors = append(rollbackErrors, errRestore.Error())
		}
		if errReload := engine.Systemd.DaemonReload(); errReload != nil {
			rollbackErrors = append(rollbackErrors, errReload.Error())
		}
		if plan.Manager != nil {
			if errManager := engine.Systemd.Restart(plan.Manager.Unit); errManager != nil {
				rollbackErrors = append(rollbackErrors, errManager.Error())
			}
		}
		if errCPA := engine.Systemd.Restart(plan.CPA.Unit); errCPA != nil {
			rollbackErrors = append(rollbackErrors, errCPA.Error())
		}
		if len(rollbackErrors) > 0 {
			return State{}, fmt.Errorf("bootstrap failed: %v; automatic rollback was incomplete: %s", cause, strings.Join(rollbackErrors, "; "))
		}
		return State{}, fmt.Errorf("bootstrap failed and was rolled back: %w", cause)
	}

	if errMkdir := os.MkdirAll(plan.PluginPlatformDir, 0o755); errMkdir != nil {
		return rollback(fmt.Errorf("create plugin platform directory: %w", errMkdir))
	}
	if len(plan.PanelSeed) > 0 {
		if errPanel := writeFileAtomic(plan.PanelPath, plan.PanelSeed, 0o644, plan.ServiceUID, plan.ServiceGID); errPanel != nil {
			return rollback(errPanel)
		}
	}
	configInfo, errInfo := os.Stat(plan.ConfigPath)
	if errInfo != nil {
		return rollback(errInfo)
	}
	configUID, configGID := fileOwner(configInfo)
	if errWrite := writeFileAtomic(plan.ConfigPath, plan.PatchedConfig, configInfo.Mode().Perm(), configUID, configGID); errWrite != nil {
		return rollback(errWrite)
	}
	executableRaw, errExecutable := os.ReadFile(options.Executable)
	if errExecutable != nil {
		return rollback(errExecutable)
	}
	if errInstall := writeFileAtomic(options.InstalledBinary, executableRaw, 0o755, 0, 0); errInstall != nil {
		return rollback(errInstall)
	}
	if errRuntime := ensureRuntimeDirectory(options.RuntimeDirectory, plan.ServiceUID, plan.ServiceGID); errRuntime != nil {
		return rollback(errRuntime)
	}
	tmpfiles := []byte(fmt.Sprintf("d %s 0750 %s %s -\n", options.RuntimeDirectory, plan.ServiceUser, plan.ServiceGroup))
	if errTmpfiles := writeFileAtomic(options.TmpfilesPath, tmpfiles, 0o644, 0, 0); errTmpfiles != nil {
		return rollback(errTmpfiles)
	}
	currentFiles, errInventory := InventoryPlugins(plan.PluginPlatformDir)
	if errInventory != nil {
		return rollback(errInventory)
	}
	state := State{
		SchemaVersion: 1, BootstrapVersion: options.BootstrapVersion, CPAService: plan.CPA.Unit,
		ConfigPath: plan.ConfigPath, PanelPath: plan.PanelPath, PanelURL: options.PanelURL,
		HealthURL: plan.HealthURL, PluginsDir: plan.PluginsDir, PluginPlatformDir: plan.PluginPlatformDir,
		RestartRequestPath: filepath.Join(options.RuntimeDirectory, "restart.request"), StateDir: options.StateDir,
		StatePath: options.StatePath, InstalledBinary: options.InstalledBinary,
		PathUnit: DefaultPathUnit, TimerUnit: DefaultTimerUnit, ServiceUnit: DefaultServiceUnit, LastBackupID: backup.ID,
		Accepted: NewAcceptedState(currentFiles, plan.ConfigInfo, engine.now()),
	}
	if plan.Manager != nil {
		state.ManagerService = plan.Manager.Unit
	}
	if errState := SaveState(options.StatePath, state); errState != nil {
		return rollback(errState)
	}
	if plan.ManagerDropInPath != "" {
		if errDropIn := writeFileAtomic(plan.ManagerDropInPath, ManagerDropInContents(plan.PanelPath), 0o644, 0, 0); errDropIn != nil {
			return rollback(errDropIn)
		}
	}
	if errUnit := writeFileAtomic(pathUnitPath, PathUnitContents(state), 0o644, 0, 0); errUnit != nil {
		return rollback(errUnit)
	}
	if errUnit := writeFileAtomic(serviceUnitPath, ServiceUnitContents(state), 0o644, 0, 0); errUnit != nil {
		return rollback(errUnit)
	}
	if errUnit := writeFileAtomic(timerUnitPath, TimerUnitContents(state), 0o644, 0, 0); errUnit != nil {
		return rollback(errUnit)
	}
	if errReload := engine.Systemd.DaemonReload(); errReload != nil {
		return rollback(errReload)
	}
	if plan.Manager != nil && plan.ManagerDropInPath != "" {
		if errManager := engine.Systemd.Restart(plan.Manager.Unit); errManager != nil {
			return rollback(errManager)
		}
	}
	oldPID := plan.CPA.MainPID
	if errRestart := engine.Systemd.Restart(plan.CPA.Unit); errRestart != nil {
		return rollback(errRestart)
	}
	if errEnable := engine.Systemd.EnablePath(DefaultPathUnit); errEnable != nil {
		return rollback(errEnable)
	}
	if errEnable := engine.Systemd.EnablePath(DefaultTimerUnit); errEnable != nil {
		return rollback(errEnable)
	}
	expectPlugin := len(currentFiles) > 0 && plan.ConfigInfo.DesiredVersion != ""
	verifier := Verifier{Systemd: engine.Systemd, Client: engine.Client}
	if errVerify := verifier.Verify(ctx, state, oldPID, plan.ConfigInfo.DesiredVersion, expectPlugin); errVerify != nil {
		return rollback(errVerify)
	}
	refreshedConfig, errReadConfig := os.ReadFile(state.ConfigPath)
	if errReadConfig != nil {
		return rollback(errReadConfig)
	}
	refreshedInfo, errInspect := InspectConfig(refreshedConfig, filepath.Dir(state.ConfigPath))
	if errInspect != nil {
		return rollback(errInspect)
	}
	refreshedFiles, errFiles := InventoryPlugins(state.PluginPlatformDir)
	if errFiles != nil {
		return rollback(errFiles)
	}
	state.Accepted = NewAcceptedState(refreshedFiles, refreshedInfo, engine.now())
	if errState := SaveState(options.StatePath, state); errState != nil {
		return rollback(errState)
	}
	return state, nil
}

func (engine Engine) Rollback(_ context.Context, options Options, id string) error {
	options = normalizeOptions(options)
	if options.RequireRoot && effectiveUID() != 0 {
		return fmt.Errorf("bootstrap rollback requires root")
	}
	manifest, errLoad := LoadBackup(options.StateDir, id)
	if errLoad != nil {
		return errLoad
	}
	_ = engine.Systemd.DisablePath(DefaultPathUnit)
	_ = engine.Systemd.DisablePath(DefaultTimerUnit)
	if errRestore := RestoreBackup(options.StateDir, manifest); errRestore != nil {
		return errRestore
	}
	if errReload := engine.Systemd.DaemonReload(); errReload != nil {
		return errReload
	}
	if manifest.ManagerService != "" {
		if errManager := engine.Systemd.Restart(manifest.ManagerService); errManager != nil {
			return errManager
		}
	}
	if errCPA := engine.Systemd.Restart(manifest.CPAService); errCPA != nil {
		return errCPA
	}
	return nil
}

func (engine Engine) Watch(ctx context.Context, statePath string) error {
	state, errState := LoadState(statePath)
	if errState != nil {
		return errState
	}
	manual, errRequest := consumeRestartRequest(state.RestartRequestPath)
	if errRequest != nil {
		return errRequest
	}
	current, errCurrent := InventoryPlugins(state.PluginPlatformDir)
	if errCurrent != nil {
		return errCurrent
	}
	configRaw, errConfigRead := os.ReadFile(state.ConfigPath)
	if errConfigRead != nil {
		return fmt.Errorf("read CPA config before marketplace verification: %w", errConfigRead)
	}
	reassertedConfig, configInfo, errReassert := ReassertBootstrapConfig(configRaw, ConfigPatch{
		PluginsDir: state.PluginsDir, PanelPath: state.PanelPath, ConfigPath: state.ConfigPath,
		RestartMode: "broker", RestartService: state.CPAService,
	})
	if errReassert != nil {
		return fmt.Errorf("reassert bootstrap-owned plugin settings: %w", errReassert)
	}
	configChanged := !bytes.Equal(configRaw, reassertedConfig)
	inventoryStable := SameInventory(current, state.Accepted.Files)
	if !manual && inventoryStable && !configChanged {
		return nil
	}
	if configChanged {
		configInfoStat, errConfigStat := os.Stat(state.ConfigPath)
		if errConfigStat != nil {
			return fmt.Errorf("inspect CPA config before bootstrap reassertion: %w", errConfigStat)
		}
		uid, gid := fileOwner(configInfoStat)
		if errConfigWrite := writeFileAtomic(state.ConfigPath, reassertedConfig, configInfoStat.Mode().Perm(), uid, gid); errConfigWrite != nil {
			return fmt.Errorf("reassert bootstrap-owned plugin settings: %w", errConfigWrite)
		}
	}
	settleManual := manual || (configChanged && inventoryStable && configInfo.DesiredVersion == "")
	info, stableFiles, errStable := waitForSettledInstall(ctx, state, settleManual)
	if errStable != nil {
		return errStable
	}
	service, errService := engine.Systemd.Service(state.CPAService)
	if errService != nil {
		return errService
	}
	if errRestart := engine.Systemd.Restart(state.CPAService); errRestart != nil {
		return engine.runtimeRollback(ctx, state, stableFiles, fmt.Errorf("restart CPA: %w", errRestart))
	}
	verifier := Verifier{Systemd: engine.Systemd, Client: engine.Client}
	if errVerify := verifier.Verify(ctx, state, service.MainPID, info.DesiredVersion, len(stableFiles) > 0); errVerify != nil {
		return engine.runtimeRollback(ctx, state, stableFiles, errVerify)
	}
	state.Accepted = NewAcceptedState(stableFiles, info, engine.now())
	if errSave := SaveState(state.StatePath, state); errSave != nil {
		return fmt.Errorf("restart succeeded but saving accepted state failed: %w", errSave)
	}
	return nil
}

func (engine Engine) runtimeRollback(ctx context.Context, state State, current []PluginFile, cause error) error {
	if errRemove := RemoveUnacceptedPlugins(current, state.Accepted.Files); errRemove != nil {
		return fmt.Errorf("verification failed: %v; remove failed plugin: %w", cause, errRemove)
	}
	raw, errRead := os.ReadFile(state.ConfigPath)
	if errRead != nil {
		return fmt.Errorf("verification failed: %v; read config for rollback: %w", cause, errRead)
	}
	restored, errRestoreConfig := RestorePluginConfig(raw, state.Accepted.PluginConfigYAML, state.Accepted.PluginConfigPresent)
	if errRestoreConfig != nil {
		return fmt.Errorf("verification failed: %v; restore plugin config: %w", cause, errRestoreConfig)
	}
	info, errInfo := os.Stat(state.ConfigPath)
	if errInfo != nil {
		return fmt.Errorf("verification failed: %v; inspect config: %w", cause, errInfo)
	}
	uid, gid := fileOwner(info)
	if errWrite := writeFileAtomic(state.ConfigPath, restored, info.Mode().Perm(), uid, gid); errWrite != nil {
		return fmt.Errorf("verification failed: %v; write restored config: %w", cause, errWrite)
	}
	before, _ := engine.Systemd.Service(state.CPAService)
	if errRestart := engine.Systemd.Restart(state.CPAService); errRestart != nil {
		return fmt.Errorf("verification failed: %v; rollback restart failed: %w", cause, errRestart)
	}
	verifier := Verifier{Systemd: engine.Systemd, Client: engine.Client}
	if errVerify := verifier.Verify(ctx, state, before.MainPID, state.Accepted.DesiredVersion, len(state.Accepted.Files) > 0); errVerify != nil {
		return fmt.Errorf("verification failed: %v; rollback verification failed: %w", cause, errVerify)
	}
	return fmt.Errorf("new plugin deployment failed verification and was rolled back: %w", cause)
}

func (engine Engine) resolvePanel(ctx context.Context, options Options, cpa ServiceInfo, manager *ServiceInfo, configPath string) (string, []byte, bool, error) {
	panelURL := strings.TrimSpace(options.PanelURL)
	if manager != nil && panelURL == "" {
		return "", nil, false, fmt.Errorf("Manager Server deployments require --panel-url so bootstrap can verify the public panel")
	}
	requested := strings.TrimSpace(options.PanelPath)
	if requested != "" {
		requested = absoluteFrom(cpa.WorkingDirectory, requested)
		seed, errSeed := engine.panelSeed(ctx, requested, panelURL)
		if errSeed == nil && manager != nil && len(seed) == 0 {
			errSeed = engine.confirmPanelMatchesURL(ctx, requested, panelURL)
		}
		return requested, seed, manager != nil && PanelPathFromManager(*manager) != requested, errSeed
	}
	if manager != nil {
		if configured := PanelPathFromManager(*manager); configured != "" {
			if !regularFile(configured) {
				return "", nil, false, fmt.Errorf("Manager Server PANEL_PATH is not a regular file: %s", configured)
			}
			if errConfirm := engine.confirmPanelMatchesURL(ctx, configured, panelURL); errConfirm != nil {
				return "", nil, false, errConfirm
			}
			return configured, nil, false, nil
		}
		seed, errSeed := engine.panelSeed(ctx, DefaultPanelPath, panelURL)
		if errSeed == nil && len(seed) == 0 {
			errSeed = engine.confirmPanelMatchesURL(ctx, DefaultPanelPath, panelURL)
		}
		return DefaultPanelPath, seed, true, errSeed
	}
	candidates := []string{
		filepath.Join(cpa.WorkingDirectory, "static", "management.html"),
		filepath.Join(filepath.Dir(cpa.Executable), "static", "management.html"),
		filepath.Join(filepath.Dir(configPath), "static", "management.html"),
	}
	unique := make([]string, 0)
	seen := make(map[string]struct{})
	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		if _, found := seen[candidate]; found || !regularFile(candidate) {
			continue
		}
		seen[candidate] = struct{}{}
		unique = append(unique, candidate)
	}
	if len(unique) != 1 {
		return "", nil, false, fmt.Errorf("found %d possible panel files and no Manager Server PANEL_PATH; pass --panel-path explicitly", len(unique))
	}
	if strings.TrimSpace(options.PanelURL) != "" {
		publicRaw, _, status, errFetch := fetchLimited(ctx, engine.client(), options.PanelURL)
		if errFetch != nil || status != http.StatusOK {
			return "", nil, false, fmt.Errorf("fetch public panel for path confirmation: HTTP %d: %w", status, errFetch)
		}
		localRaw, errLocal := os.ReadFile(unique[0])
		if errLocal != nil {
			return "", nil, false, errLocal
		}
		if !bytes.Equal(publicRaw, localRaw) {
			return "", nil, false, fmt.Errorf("the only local panel does not match --panel-url; refusing to patch the wrong file")
		}
	}
	return unique[0], nil, false, nil
}

func (engine Engine) panelSeed(ctx context.Context, panelPath, panelURL string) ([]byte, error) {
	if !filepath.IsAbs(panelPath) || isFilesystemRoot(panelPath) || !strings.EqualFold(filepath.Ext(panelPath), ".html") || strings.ContainsAny(panelPath, "\r\n") {
		return nil, fmt.Errorf("panel path must be an absolute .html file path: %s", panelPath)
	}
	if regularFile(panelPath) {
		return nil, nil
	}
	if strings.TrimSpace(panelURL) == "" {
		return nil, fmt.Errorf("panel file does not exist and --panel-url was not supplied: %s", panelPath)
	}
	raw, _, status, errFetch := fetchLimited(ctx, engine.client(), panelURL)
	if errFetch != nil {
		return nil, fmt.Errorf("download active panel: %w", errFetch)
	}
	if status != http.StatusOK || !bytes.Contains(bytes.ToLower(raw), []byte("<html")) {
		return nil, fmt.Errorf("active panel download returned HTTP %d or non-HTML content", status)
	}
	return raw, nil
}

func (engine Engine) confirmPanelMatchesURL(ctx context.Context, panelPath, panelURL string) error {
	publicRaw, _, status, errFetch := fetchLimited(ctx, engine.client(), panelURL)
	if errFetch != nil {
		return fmt.Errorf("fetch public panel for path confirmation: %w", errFetch)
	}
	if status != http.StatusOK {
		return fmt.Errorf("fetch public panel for path confirmation: HTTP %d", status)
	}
	localRaw, errLocal := os.ReadFile(panelPath)
	if errLocal != nil {
		return fmt.Errorf("read panel for public path confirmation: %w", errLocal)
	}
	if !bytes.Equal(publicRaw, localRaw) {
		return fmt.Errorf("configured panel does not match --panel-url; refusing to patch the wrong file")
	}
	return nil
}

func waitForSettledInstall(ctx context.Context, state State, manual bool) (ConfigInfo, []PluginFile, error) {
	deadline := time.Now().Add(30 * time.Second)
	var previous []PluginFile
	stableCount := 0
	for time.Now().Before(deadline) {
		raw, errRead := os.ReadFile(state.ConfigPath)
		if errRead != nil {
			return ConfigInfo{}, nil, errRead
		}
		info, errInspect := InspectConfig(raw, filepath.Dir(state.ConfigPath))
		if errInspect != nil {
			return ConfigInfo{}, nil, errInspect
		}
		files, errFiles := InventoryPlugins(state.PluginPlatformDir)
		if errFiles != nil {
			return ConfigInfo{}, nil, errFiles
		}
		if SameInventory(previous, files) {
			stableCount++
		} else {
			stableCount = 0
			previous = files
		}
		ready := manual || (info.DesiredVersion != "" && InventoryHasVersion(files, info.DesiredVersion))
		if ready && stableCount >= 2 {
			return info, files, nil
		}
		timer := time.NewTimer(500 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ConfigInfo{}, nil, ctx.Err()
		case <-timer.C:
		}
	}
	return ConfigInfo{}, nil, fmt.Errorf("plugin install did not settle with a matching store.version within 30 seconds")
}

func consumeRestartRequest(path string) (bool, error) {
	info, errInfo := os.Lstat(path)
	if errors.Is(errInfo, os.ErrNotExist) {
		return false, nil
	}
	if errInfo != nil {
		return false, fmt.Errorf("inspect restart request: %w", errInfo)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > 4096 {
		return false, fmt.Errorf("restart request is not a small regular non-symlink file")
	}
	if errRemove := os.Remove(path); errRemove != nil {
		return false, fmt.Errorf("consume restart request: %w", errRemove)
	}
	return true, nil
}

func normalizeOptions(options Options) Options {
	if options.StatePath == "" {
		options.StatePath = DefaultStatePath
	}
	if options.StateDir == "" {
		options.StateDir = DefaultStateDir
	}
	if options.InstalledBinary == "" {
		options.InstalledBinary = DefaultInstalledBinary
	}
	if options.SystemdDir == "" {
		options.SystemdDir = DefaultSystemdDir
	}
	if options.TmpfilesPath == "" {
		options.TmpfilesPath = DefaultTmpfilesPath
	}
	if options.RuntimeDirectory == "" {
		options.RuntimeDirectory = DefaultRuntimeDirectory
	}
	return options
}

func validateOptionPaths(options Options) error {
	for label, path := range map[string]string{
		"state path": options.StatePath, "state directory": options.StateDir,
		"installed binary": options.InstalledBinary, "systemd directory": options.SystemdDir,
		"tmpfiles path": options.TmpfilesPath, "runtime directory": options.RuntimeDirectory,
	} {
		if !filepath.IsAbs(path) || isFilesystemRoot(path) || strings.ContainsAny(path, "\r\n") {
			return fmt.Errorf("%s must be a non-root absolute path: %s", label, path)
		}
	}
	return nil
}

func validateHTTPURL(value, label string) error {
	parsed, errParse := url.Parse(strings.TrimSpace(value))
	if errParse != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
		return fmt.Errorf("%s must be an http(s) URL without embedded credentials", label)
	}
	return nil
}

func isFilesystemRoot(path string) bool {
	cleaned := filepath.Clean(path)
	return cleaned == filepath.VolumeName(cleaned)+string(filepath.Separator)
}

func resolveServiceIdentity(service ServiceInfo) (string, string, int, int, error) {
	name := strings.TrimSpace(service.User)
	if name == "" {
		name = "root"
	}
	group := strings.TrimSpace(service.Group)
	account, errLookup := user.Lookup(name)
	if errLookup != nil {
		return "", "", 0, 0, fmt.Errorf("resolve CPA service user %s: %w", name, errLookup)
	}
	uid, errUID := strconv.Atoi(account.Uid)
	if errUID != nil {
		return "", "", 0, 0, errUID
	}
	if group == "" {
		groupAccount, errGroupID := user.LookupGroupId(account.Gid)
		if errGroupID == nil {
			gid, errGID := strconv.Atoi(groupAccount.Gid)
			if errGID != nil {
				return "", "", 0, 0, errGID
			}
			return name, groupAccount.Name, uid, gid, nil
		}
		group = account.Gid
	}
	groupAccount, errGroup := user.LookupGroup(group)
	if errGroup != nil {
		if numeric, errNumeric := strconv.Atoi(group); errNumeric == nil {
			return name, group, uid, numeric, nil
		}
		return "", "", 0, 0, fmt.Errorf("resolve CPA service group %s: %w", group, errGroup)
	}
	gid, errGID := strconv.Atoi(groupAccount.Gid)
	if errGID != nil {
		return "", "", 0, 0, errGID
	}
	return name, groupAccount.Name, uid, gid, nil
}

func ensureRuntimeDirectory(path string, uid, gid int) error {
	if errMkdir := os.MkdirAll(path, 0o750); errMkdir != nil {
		return errMkdir
	}
	info, errInfo := os.Lstat(path)
	if errInfo != nil {
		return errInfo
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("runtime path is not a regular directory: %s", path)
	}
	if errMode := os.Chmod(path, 0o750); errMode != nil {
		return errMode
	}
	return os.Chown(path, uid, gid)
}

func managerUnit(manager *ServiceInfo) string {
	if manager == nil {
		return ""
	}
	return manager.Unit
}

func (engine Engine) now() time.Time {
	if engine.Now != nil {
		return engine.Now()
	}
	return time.Now()
}

func (engine Engine) client() *http.Client {
	if engine.Client != nil {
		return engine.Client
	}
	return &http.Client{Timeout: 20 * time.Second}
}
