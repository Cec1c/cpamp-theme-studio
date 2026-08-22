package bootstrap

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var unitNamePattern = regexp.MustCompile(`^[A-Za-z0-9_.@:-]+\.service$`)

type Runner interface {
	Run(name string, args ...string) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

type ServiceInfo struct {
	Unit             string
	MainPID          int
	WorkingDirectory string
	Executable       string
	User             string
	Group            string
	Environment      map[string]string
}

type Systemd struct {
	Runner Runner
}

func (systemd Systemd) Service(unit string) (ServiceInfo, error) {
	if !unitNamePattern.MatchString(strings.TrimSpace(unit)) {
		return ServiceInfo{}, fmt.Errorf("invalid systemd service name %q", unit)
	}
	runner := systemd.Runner
	if runner == nil {
		runner = ExecRunner{}
	}
	raw, errShow := runner.Run("systemctl", "show", "--no-pager", "--property=MainPID,WorkingDirectory,User,Group,Environment", unit)
	if errShow != nil {
		return ServiceInfo{}, fmt.Errorf("inspect systemd service %s: %w: %s", unit, errShow, strings.TrimSpace(string(raw)))
	}
	values := parseProperties(string(raw))
	pid, errPID := strconv.Atoi(strings.TrimSpace(values["MainPID"]))
	if errPID != nil || pid <= 0 {
		return ServiceInfo{}, fmt.Errorf("systemd service %s has no running MainPID", unit)
	}
	executable, errExecutable := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if errExecutable != nil {
		return ServiceInfo{}, fmt.Errorf("resolve executable for %s PID %d: %w", unit, pid, errExecutable)
	}
	workingDir := strings.TrimSpace(values["WorkingDirectory"])
	if workingDir == "" {
		if resolved, errWorking := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid)); errWorking == nil {
			workingDir = resolved
		}
	}
	if strings.TrimSpace(workingDir) == "" || !filepath.IsAbs(workingDir) {
		return ServiceInfo{}, fmt.Errorf("systemd service %s has no absolute working directory", unit)
	}
	return ServiceInfo{
		Unit: unit, MainPID: pid, WorkingDirectory: filepath.Clean(workingDir), Executable: filepath.Clean(executable),
		User: strings.TrimSpace(values["User"]), Group: strings.TrimSpace(values["Group"]),
		Environment: processEnvironment(pid),
	}, nil
}

func (systemd Systemd) FindServices() (ServiceInfo, *ServiceInfo, error) {
	runner := systemd.Runner
	if runner == nil {
		runner = ExecRunner{}
	}
	raw, errList := runner.Run("systemctl", "list-units", "--type=service", "--state=running", "--no-legend", "--no-pager", "--plain")
	if errList != nil {
		return ServiceInfo{}, nil, fmt.Errorf("list running systemd services: %w: %s", errList, strings.TrimSpace(string(raw)))
	}
	var cpaCandidates []ServiceInfo
	var managerCandidates []ServiceInfo
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || !unitNamePattern.MatchString(fields[0]) {
			continue
		}
		service, errService := systemd.Service(fields[0])
		if errService != nil {
			continue
		}
		base := strings.ToLower(filepath.Base(service.Executable))
		cmdline := strings.ToLower(strings.Join(processArguments(service.MainPID), " "))
		signature := base + " " + cmdline
		if strings.HasPrefix(base, "cli-proxy-api") {
			cpaCandidates = append(cpaCandidates, service)
			continue
		}
		if strings.Contains(signature, "cpa-manager") || strings.Contains(signature, "manager-server") || strings.Contains(signature, "cpamp-manager") {
			managerCandidates = append(managerCandidates, service)
		}
	}
	if len(cpaCandidates) != 1 {
		return ServiceInfo{}, nil, fmt.Errorf("found %d running CPA systemd services; pass --service explicitly", len(cpaCandidates))
	}
	if len(managerCandidates) > 1 {
		return ServiceInfo{}, nil, fmt.Errorf("found %d possible Manager Server services; pass --manager-service explicitly", len(managerCandidates))
	}
	if len(managerCandidates) == 1 {
		manager := managerCandidates[0]
		return cpaCandidates[0], &manager, nil
	}
	return cpaCandidates[0], nil, nil
}

func (systemd Systemd) Restart(unit string) error {
	if !unitNamePattern.MatchString(strings.TrimSpace(unit)) {
		return fmt.Errorf("invalid systemd service name %q", unit)
	}
	raw, errRun := systemd.runner().Run("systemctl", "restart", unit)
	if errRun != nil {
		return fmt.Errorf("restart %s: %w: %s", unit, errRun, strings.TrimSpace(string(raw)))
	}
	return nil
}

func (systemd Systemd) IsActive(unit string) bool {
	_, errRun := systemd.runner().Run("systemctl", "is-active", "--quiet", unit)
	return errRun == nil
}

func (systemd Systemd) DaemonReload() error {
	raw, errRun := systemd.runner().Run("systemctl", "daemon-reload")
	if errRun != nil {
		return fmt.Errorf("reload systemd: %w: %s", errRun, strings.TrimSpace(string(raw)))
	}
	return nil
}

func (systemd Systemd) EnablePath(unit string) error {
	raw, errRun := systemd.runner().Run("systemctl", "enable", "--now", unit)
	if errRun != nil {
		return fmt.Errorf("enable %s: %w: %s", unit, errRun, strings.TrimSpace(string(raw)))
	}
	return nil
}

func (systemd Systemd) DisablePath(unit string) error {
	raw, errRun := systemd.runner().Run("systemctl", "disable", "--now", unit)
	if errRun != nil && !strings.Contains(strings.ToLower(string(raw)), "not loaded") && !strings.Contains(strings.ToLower(string(raw)), "does not exist") {
		return fmt.Errorf("disable %s: %w: %s", unit, errRun, strings.TrimSpace(string(raw)))
	}
	return nil
}

func (systemd Systemd) runner() Runner {
	if systemd.Runner == nil {
		return ExecRunner{}
	}
	return systemd.Runner
}

func parseProperties(raw string) map[string]string {
	values := make(map[string]string)
	for _, line := range strings.Split(raw, "\n") {
		key, value, found := strings.Cut(line, "=")
		if found {
			values[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return values
}

func processArguments(pid int) []string {
	raw, errRead := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if errRead != nil {
		return nil
	}
	parts := strings.Split(string(raw), "\x00")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}

func processEnvironment(pid int) map[string]string {
	result := make(map[string]string)
	raw, errRead := os.ReadFile(fmt.Sprintf("/proc/%d/environ", pid))
	if errRead != nil {
		return result
	}
	for _, item := range strings.Split(string(raw), "\x00") {
		key, value, found := strings.Cut(item, "=")
		if found && key != "" {
			result[key] = value
		}
	}
	return result
}

func ConfigPathFromProcess(service ServiceInfo) string {
	args := processArguments(service.MainPID)
	for index, arg := range args {
		for _, name := range []string{"--config", "-config"} {
			if arg == name && index+1 < len(args) {
				return absoluteFrom(service.WorkingDirectory, args[index+1])
			}
			if strings.HasPrefix(arg, name+"=") {
				return absoluteFrom(service.WorkingDirectory, strings.TrimPrefix(arg, name+"="))
			}
		}
	}
	for _, candidate := range []string{
		filepath.Join(service.WorkingDirectory, "config.yaml"),
		filepath.Join(filepath.Dir(service.Executable), "config.yaml"),
	} {
		if regularFile(candidate) {
			return filepath.Clean(candidate)
		}
	}
	return ""
}

func PanelPathFromManager(service ServiceInfo) string {
	if panel := strings.TrimSpace(service.Environment["PANEL_PATH"]); panel != "" {
		return absoluteFrom(service.WorkingDirectory, panel)
	}
	return ""
}

func absoluteFrom(base, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	return filepath.Clean(filepath.Join(base, value))
}

func regularFile(path string) bool {
	info, errInfo := os.Lstat(path)
	return errInfo == nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0
}

func systemdQuote(value string) string {
	value = strings.ReplaceAll(value, "%", "%%")
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

func PathUnitContents(state State) []byte {
	return []byte(fmt.Sprintf(`[Unit]
Description=Restart CPA after CPAMP Theme Studio install or confirmed request
After=%s

[Path]
PathChanged=%s
PathExists=%s
Unit=%s

[Install]
WantedBy=multi-user.target
`, state.CPAService, state.PluginPlatformDir, state.RestartRequestPath, state.ServiceUnit))
}

func ServiceUnitContents(state State) []byte {
	return []byte(fmt.Sprintf(`[Unit]
Description=Validate and restart CPA for CPAMP Theme Studio
After=network-online.target

[Service]
Type=oneshot
ExecStart=%s watch --state %s
TimeoutStartSec=180
NoNewPrivileges=true
ProtectHome=true
ProtectSystem=strict
ReadWritePaths=%s %s %s %s %s %s
`, systemdQuote(state.InstalledBinary), systemdQuote(state.StatePath), systemdQuote(state.StateDir), systemdQuote(filepath.Dir(state.StatePath)), systemdQuote(filepath.Dir(state.ConfigPath)), systemdQuote(state.PluginPlatformDir), systemdQuote(filepath.Dir(state.PanelPath)), systemdQuote(filepath.Dir(state.RestartRequestPath))))
}

func TimerUnitContents(state State) []byte {
	return []byte(fmt.Sprintf(`[Unit]
Description=Safety-net check for CPAMP Theme Studio plugin changes

[Timer]
OnBootSec=30s
OnUnitInactiveSec=15s
AccuracySec=2s
Unit=%s

[Install]
WantedBy=timers.target
`, state.ServiceUnit))
}

func ManagerDropInContents(panelPath string) []byte {
	return []byte("[Service]\nEnvironment=" + systemdQuote("PANEL_PATH="+panelPath) + "\n")
}
