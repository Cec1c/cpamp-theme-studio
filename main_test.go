package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseInjectorConfig(t *testing.T) {
	cfg, errConfig := parseInjectorConfig([]byte("auto_inject: true\npanel_path: 'D:\\panel\\management.html'\nhost_config_path: 'D:\\cpa\\config.yaml'\nwatch_seconds: 999\n"))
	if errConfig != nil {
		t.Fatal(errConfig)
	}
	if !cfg.AutoInject || cfg.PanelPath != `D:\panel\management.html` {
		t.Fatalf("config = %#v", cfg)
	}
	if cfg.HostConfigPath != `D:\cpa\config.yaml` {
		t.Fatalf("host config path = %q", cfg.HostConfigPath)
	}
	if cfg.WatchPeriod != 300*time.Second {
		t.Fatalf("watch period = %s, want 300s cap", cfg.WatchPeriod)
	}
}

func TestResolveHostConfigPathFromArguments(t *testing.T) {
	want := filepath.Clean(filepath.Join(t.TempDir(), "config.yaml"))
	got := resolveHostConfigPath("", "", []string{"cli-proxy-api", "--config", want})
	if got != want {
		t.Fatalf("resolveHostConfigPath() = %q, want %q", got, want)
	}
}

func TestStopInjectorCleansConfiguredPanelWithoutTrackedState(t *testing.T) {
	stopInjector(true)
	t.Cleanup(func() { stopInjector(true) })

	panel := filepath.Join(t.TempDir(), "management.html")
	original := []byte("<html><head></head><body></body></html>")
	if errWrite := os.WriteFile(panel, original, 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	if _, errPatch := patchPanelFile(panel); errPatch != nil {
		t.Fatal(errPatch)
	}

	injectorRuntime.Lock()
	injectorRuntime.panelPath = panel
	injectorRuntime.patched = make(map[string]struct{})
	injectorRuntime.Unlock()
	stopInjector(true)

	got, errRead := os.ReadFile(panel)
	if errRead != nil {
		t.Fatal(errRead)
	}
	if string(got) != string(original) {
		t.Fatalf("panel was not restored\ngot:  %s\nwant: %s", got, original)
	}
}

func TestPluginEnabledInHostConfig(t *testing.T) {
	base := `
host: 127.0.0.1
plugins:
  enabled: true
  configs:
    another-plugin:
      enabled: false
    cpamp-theme-studio:
      enabled: %s # watched even though the host skips reconfigure on disable
      panel_path: "D:/panel/management.html"
`
	for _, testCase := range []struct {
		name    string
		value   string
		enabled bool
	}{
		{name: "enabled", value: "true", enabled: true},
		{name: "disabled", value: "false", enabled: false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			enabled, definitive := pluginEnabledInHostConfig([]byte(strings.Replace(base, "%s", testCase.value, 1)))
			if !definitive || enabled != testCase.enabled {
				t.Fatalf("pluginEnabledInHostConfig() = (%v, %v), want (%v, true)", enabled, definitive, testCase.enabled)
			}
		})
	}
}

func TestPluginEnabledInHostConfigTreatsRemovedPluginAsDisabled(t *testing.T) {
	raw := []byte("plugins:\n  enabled: true\n  configs:\n    another-plugin:\n      enabled: true\n")
	enabled, definitive := pluginEnabledInHostConfig(raw)
	if !definitive || enabled {
		t.Fatalf("pluginEnabledInHostConfig() = (%v, %v), want (false, true)", enabled, definitive)
	}
}

func TestPluginEnabledInHostConfigTreatsGlobalDisableAsDisabled(t *testing.T) {
	raw := []byte("plugins:\n  enabled: false\n  configs:\n    cpamp-theme-studio:\n      enabled: true\n")
	enabled, definitive := pluginEnabledInHostConfig(raw)
	if !definitive || enabled {
		t.Fatalf("pluginEnabledInHostConfig() = (%v, %v), want (false, true)", enabled, definitive)
	}
}

func TestPluginEnabledInHostConfigIgnoresInvalidBoolean(t *testing.T) {
	raw := []byte("plugins:\n  enabled: true\n  configs:\n    cpamp-theme-studio:\n      enabled: perhaps\n")
	_, definitive := pluginEnabledInHostConfig(raw)
	if definitive {
		t.Fatal("invalid boolean should not trigger destructive cleanup")
	}
}

func TestParseInjectorConfigDefaults(t *testing.T) {
	cfg, errConfig := parseInjectorConfig(nil)
	if errConfig != nil {
		t.Fatal(errConfig)
	}
	if !cfg.AutoInject || cfg.WatchPeriod != 3*time.Second {
		t.Fatalf("defaults = %#v", cfg)
	}
}

func TestParseInjectorConfigRejectsWrongTypes(t *testing.T) {
	if _, errConfig := parseInjectorConfig([]byte("auto_inject: yes-please\n")); errConfig == nil {
		t.Fatal("expected invalid boolean to fail")
	}
}

func TestParseInjectorConfigClampsMinimumWatchPeriod(t *testing.T) {
	cfg, errConfig := parseInjectorConfig([]byte("watch_seconds: -5\n"))
	if errConfig != nil {
		t.Fatal(errConfig)
	}
	if cfg.WatchPeriod != time.Second {
		t.Fatalf("watch period = %s, want 1s floor", cfg.WatchPeriod)
	}
}

func TestManagementLoaderResponse(t *testing.T) {
	request := managementRequest{Method: "GET", Query: map[string][]string{"asset": {"loader"}}}
	rawRequest, errMarshal := json.Marshal(request)
	if errMarshal != nil {
		t.Fatal(errMarshal)
	}
	raw, errHandle := handleManagement(rawRequest)
	if errHandle != nil {
		t.Fatal(errHandle)
	}
	var env envelope
	if errUnmarshal := json.Unmarshal(raw, &env); errUnmarshal != nil {
		t.Fatal(errUnmarshal)
	}
	var response managementResponse
	if errUnmarshal := json.Unmarshal(env.Result, &response); errUnmarshal != nil {
		t.Fatal(errUnmarshal)
	}
	if response.StatusCode != 200 || !strings.Contains(response.Headers.Get("Content-Type"), "javascript") {
		t.Fatalf("response = %#v", response)
	}
	if !strings.Contains(string(response.Body), "__CPAMP_THEME_STUDIO__") {
		t.Fatal("loader response is missing the runtime marker")
	}
	if response.Headers.Get("ETag") == "" || !strings.Contains(response.Headers.Get("Cache-Control"), "immutable") {
		t.Fatalf("loader cache headers = %#v", response.Headers)
	}
}

func TestManagementResourceRejectsMutationAndUnknownAsset(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		request managementRequest
		status  int
	}{
		{name: "post", request: managementRequest{Method: "POST"}, status: 405},
		{name: "unknown asset", request: managementRequest{Method: "GET", Query: map[string][]string{"asset": {"missing"}}}, status: 404},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			rawRequest, _ := json.Marshal(testCase.request)
			raw, errHandle := handleManagement(rawRequest)
			if errHandle != nil {
				t.Fatal(errHandle)
			}
			var env envelope
			if errUnmarshal := json.Unmarshal(raw, &env); errUnmarshal != nil {
				t.Fatal(errUnmarshal)
			}
			var response managementResponse
			if errUnmarshal := json.Unmarshal(env.Result, &response); errUnmarshal != nil {
				t.Fatal(errUnmarshal)
			}
			if response.StatusCode != testCase.status {
				t.Fatalf("status = %d, want %d", response.StatusCode, testCase.status)
			}
		})
	}
}

func TestPluginRegistrationUsesStandaloneRepository(t *testing.T) {
	registration := pluginRegistration()
	if registration.Metadata.GitHubRepository != pluginRepository {
		t.Fatalf("repository = %q", registration.Metadata.GitHubRepository)
	}
	if registration.SchemaVersion != schemaVersion || !registration.Capabilities.ManagementAPI {
		t.Fatalf("registration = %#v", registration)
	}
}

func TestRegistrationOnlyPublishesStudioMenu(t *testing.T) {
	raw, errHandle := handleMethod(methodManagementRegister, nil)
	if errHandle != nil {
		t.Fatal(errHandle)
	}
	if strings.Count(string(raw), `"Path"`) != 1 || !strings.Contains(string(raw), `"/studio"`) {
		t.Fatalf("registration = %s", raw)
	}
}

func TestLoaderRuntimeContract(t *testing.T) {
	loader := string(loaderJavaScript)
	for _, required := range []string{
		"__CPAMP_THEME_STUDIO__",
		"data-cpamp-theme-studio-loader",
		"attachShadow({ mode: 'open' })",
		"'zh-CN'",
		"'zh-TW'",
		"en:",
		"ru:",
		"prefers-reduced-motion",
	} {
		if !strings.Contains(loader, required) {
			t.Fatalf("loader is missing runtime contract %q", required)
		}
	}
	if strings.Contains(loader, "var hostConnected = hostWindow !== window || !window.parent") {
		t.Fatal("standalone plugin resources must not be reported as an injected host panel")
	}
}
