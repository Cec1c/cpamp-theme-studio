package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type injectorConfig struct {
	PanelPath      string
	HostConfigPath string
	AutoInject     bool
	WatchPeriod    time.Duration
}

type injectorStatus struct {
	Configured     bool
	CandidateCount int
	PatchedCount   int
	LastError      string
	LastCheck      time.Time
}

var injectorRuntime struct {
	sync.Mutex
	stop      chan struct{}
	done      chan struct{}
	patched   map[string]struct{}
	panelPath string
	status    injectorStatus
}

func parseInjectorConfig(raw []byte) (injectorConfig, error) {
	cfg := injectorConfig{AutoInject: true, WatchPeriod: 3 * time.Second}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return cfg, nil
	}
	values := make(map[string]any)
	if errUnmarshal := yaml.Unmarshal(raw, &values); errUnmarshal != nil {
		return injectorConfig{}, fmt.Errorf("decode plugin config: %w", errUnmarshal)
	}
	if value, found := configValue(values, "auto_inject", "auto-inject"); found {
		parsed, ok := value.(bool)
		if !ok {
			return injectorConfig{}, fmt.Errorf("auto_inject must be a boolean")
		}
		cfg.AutoInject = parsed
	}
	if value, found := configValue(values, "panel_path", "panel-path"); found {
		parsed, ok := value.(string)
		if !ok {
			return injectorConfig{}, fmt.Errorf("panel_path must be a string")
		}
		cfg.PanelPath = strings.TrimSpace(parsed)
	}
	if value, found := configValue(values, "host_config_path", "host-config-path"); found {
		parsed, ok := value.(string)
		if !ok {
			return injectorConfig{}, fmt.Errorf("host_config_path must be a string")
		}
		cfg.HostConfigPath = strings.TrimSpace(parsed)
	}
	if value, found := configValue(values, "watch_seconds", "watch-seconds"); found {
		seconds, ok := integerValue(value)
		if !ok {
			return injectorConfig{}, fmt.Errorf("watch_seconds must be an integer")
		}
		if seconds < 1 {
			seconds = 1
		}
		if seconds > 300 {
			seconds = 300
		}
		cfg.WatchPeriod = time.Duration(seconds) * time.Second
	}
	return cfg, nil
}

func configValue(values map[string]any, names ...string) (any, bool) {
	for _, name := range names {
		if value, found := values[name]; found {
			return value, true
		}
	}
	return nil, false
}

func integerValue(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case uint64:
		if typed > uint64(^uint(0)>>1) {
			return 0, false
		}
		return int(typed), true
	case string:
		parsed, errParse := strconv.Atoi(strings.TrimSpace(typed))
		return parsed, errParse == nil
	default:
		return 0, false
	}
}

func configureInjector(cfg injectorConfig) {
	stopInjector(true)
	cfg.HostConfigPath = resolveHostConfigPath(cfg.HostConfigPath, cfg.PanelPath, os.Args)
	injectorRuntime.Lock()
	injectorRuntime.panelPath = cfg.PanelPath
	injectorRuntime.status = injectorStatus{Configured: cfg.AutoInject}
	if !cfg.AutoInject {
		injectorRuntime.Unlock()
		return
	}
	stop := make(chan struct{})
	done := make(chan struct{})
	injectorRuntime.stop = stop
	injectorRuntime.done = done
	injectorRuntime.patched = make(map[string]struct{})
	injectorRuntime.Unlock()

	patchCandidates(cfg.PanelPath)
	go func() {
		defer close(done)
		ticker := time.NewTicker(cfg.WatchPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				if hostConfigDisablesPlugin(cfg.HostConfigPath) {
					unpatchTrackedPanels()
					return
				}
				patchCandidates(cfg.PanelPath)
			}
		}
	}()
}

func resolveHostConfigPath(configuredPath string, panelPath string, args []string) string {
	if path := absoluteCleanPath(configuredPath); path != "" {
		return path
	}
	for index := 1; index < len(args); index++ {
		arg := strings.TrimSpace(args[index])
		if arg == "--config" || arg == "-config" {
			if index+1 < len(args) {
				return absoluteCleanPath(args[index+1])
			}
			continue
		}
		for _, prefix := range []string{"--config=", "-config="} {
			if strings.HasPrefix(arg, prefix) {
				return absoluteCleanPath(strings.TrimPrefix(arg, prefix))
			}
		}
	}

	candidates := make([]string, 0, 8)
	if workingDirectory, errWorkingDirectory := os.Getwd(); errWorkingDirectory == nil {
		candidates = append(candidates, filepath.Join(workingDirectory, "config.yaml"), filepath.Join(workingDirectory, "config.yml"))
	}
	if executable, errExecutable := os.Executable(); errExecutable == nil {
		directory := filepath.Dir(executable)
		candidates = append(candidates, filepath.Join(directory, "config.yaml"), filepath.Join(directory, "config.yml"))
	}
	if panelPath = absoluteCleanPath(panelPath); panelPath != "" {
		directory := panelPath
		if info, errStat := os.Stat(panelPath); errStat == nil && !info.IsDir() {
			directory = filepath.Dir(panelPath)
		} else if strings.EqualFold(filepath.Ext(panelPath), ".html") {
			directory = filepath.Dir(panelPath)
		}
		for depth := 0; depth < 4; depth++ {
			candidates = append(candidates, filepath.Join(directory, "config.yaml"), filepath.Join(directory, "config.yml"))
			parent := filepath.Dir(directory)
			if parent == directory {
				break
			}
			directory = parent
		}
	}
	for _, candidate := range candidates {
		if info, errStat := os.Stat(candidate); errStat == nil && !info.IsDir() {
			return filepath.Clean(candidate)
		}
	}
	return ""
}

func absoluteCleanPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if absolute, errAbsolute := filepath.Abs(path); errAbsolute == nil {
		return filepath.Clean(absolute)
	}
	return filepath.Clean(path)
}

func hostConfigDisablesPlugin(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	raw, errRead := os.ReadFile(path)
	if errRead != nil {
		return false
	}
	enabled, definitive := pluginEnabledInHostConfig(raw)
	return definitive && !enabled
}

func pluginEnabledInHostConfig(raw []byte) (bool, bool) {
	root := make(map[string]any)
	if errUnmarshal := yaml.Unmarshal(raw, &root); errUnmarshal != nil {
		return false, false
	}
	plugins, ok := stringMap(root["plugins"])
	if !ok {
		return false, true
	}
	globalEnabled, ok := plugins["enabled"].(bool)
	if !ok {
		return false, false
	}
	if !globalEnabled {
		return false, true
	}
	configs, ok := stringMap(plugins["configs"])
	if !ok {
		return false, true
	}
	pluginConfig, ok := stringMap(configs[pluginID])
	if !ok {
		return false, true
	}
	rawEnabled, found := pluginConfig["enabled"]
	if !found {
		return false, true
	}
	pluginEnabled, ok := rawEnabled.(bool)
	if !ok {
		return false, false
	}
	return pluginEnabled, true
}

func stringMap(value any) (map[string]any, bool) {
	typed, ok := value.(map[string]any)
	return typed, ok
}

func unpatchTrackedPanels() {
	injectorRuntime.Lock()
	paths := make([]string, 0, len(injectorRuntime.patched))
	for path := range injectorRuntime.patched {
		paths = append(paths, path)
	}
	panelPath := injectorRuntime.panelPath
	injectorRuntime.patched = make(map[string]struct{})
	injectorRuntime.status.Configured = false
	injectorRuntime.status.PatchedCount = 0
	injectorRuntime.status.LastCheck = time.Now()
	injectorRuntime.Unlock()
	paths = cleanupPanelCandidates(paths, panelPath)
	for _, path := range paths {
		_, _ = unpatchPanelFile(path)
	}
}

func stopInjector(remove bool) {
	injectorRuntime.Lock()
	stop := injectorRuntime.stop
	done := injectorRuntime.done
	paths := make([]string, 0, len(injectorRuntime.patched))
	for path := range injectorRuntime.patched {
		paths = append(paths, path)
	}
	panelPath := injectorRuntime.panelPath
	injectorRuntime.stop = nil
	injectorRuntime.done = nil
	injectorRuntime.patched = nil
	injectorRuntime.panelPath = ""
	injectorRuntime.Unlock()
	if stop != nil {
		close(stop)
		if done != nil {
			select {
			case <-done:
			case <-time.After(2 * time.Second):
			}
		}
	}
	if remove {
		paths = cleanupPanelCandidates(paths, panelPath)
		for _, path := range paths {
			_, _ = unpatchPanelFile(path)
		}
	}
}

func cleanupPanelCandidates(tracked []string, configuredPath string) []string {
	paths := append([]string(nil), tracked...)
	paths = append(paths, resolvePanelCandidates(configuredPath)...)
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		path = filepath.Clean(strings.TrimSpace(path))
		if path == "." || path == "" {
			continue
		}
		key := strings.ToLower(path)
		if _, found := seen[key]; found {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, path)
	}
	return result
}

func patchCandidates(configuredPath string) {
	candidates := resolvePanelCandidates(configuredPath)
	nextPatched := make(map[string]struct{})
	lastError := ""
	for _, candidate := range candidates {
		if _, errStat := os.Stat(candidate); errStat != nil {
			continue
		}
		if _, errPatch := patchPanelFile(candidate); errPatch != nil {
			lastError = errPatch.Error()
			continue
		}
		nextPatched[candidate] = struct{}{}
	}
	injectorRuntime.Lock()
	if injectorRuntime.status.Configured {
		injectorRuntime.patched = nextPatched
		injectorRuntime.status.CandidateCount = len(candidates)
		injectorRuntime.status.PatchedCount = len(nextPatched)
		injectorRuntime.status.LastError = lastError
		injectorRuntime.status.LastCheck = time.Now()
	}
	injectorRuntime.Unlock()
}

func snapshotInjectorStatus() injectorStatus {
	injectorRuntime.Lock()
	defer injectorRuntime.Unlock()
	return injectorRuntime.status
}
