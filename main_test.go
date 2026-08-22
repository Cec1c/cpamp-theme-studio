package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseInjectorConfig(t *testing.T) {
	cfg, errConfig := parseInjectorConfig([]byte("auto_inject: true\npanel_path: 'D:\\panel\\management.html'\nhost_config_path: 'D:\\cpa\\config.yaml'\nwatch_seconds: 999\nrestart_mode: systemd\nrestart_service: cli-proxy-api.service\nrestart_request: request-1\n"))
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
	if cfg.RestartMode != restartModeSystemd || cfg.RestartService != "cli-proxy-api.service" || cfg.RestartRequest != "request-1" {
		t.Fatalf("restart config = %#v", cfg)
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
	if !cfg.AutoInject || cfg.WatchPeriod != 3*time.Second || cfg.RestartMode != restartModeAuto {
		t.Fatalf("defaults = %#v", cfg)
	}
}

func TestParseInjectorConfigRejectsWrongTypes(t *testing.T) {
	if _, errConfig := parseInjectorConfig([]byte("auto_inject: yes-please\n")); errConfig == nil {
		t.Fatal("expected invalid boolean to fail")
	}
	if _, errConfig := parseInjectorConfig([]byte("restart_mode: unsafe\n")); errConfig == nil {
		t.Fatal("expected invalid restart mode to fail")
	}
	if _, errConfig := parseInjectorConfig([]byte("restart_request: '" + strings.Repeat("x", maxRestartRequestLength+1) + "'\n")); errConfig == nil {
		t.Fatal("expected oversized restart request to fail")
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

func TestManagementServesBundledJetBrainsMono(t *testing.T) {
	for _, asset := range []string{"font-regular", "font-semibold"} {
		t.Run(asset, func(t *testing.T) {
			request := managementRequest{Method: "GET", Query: map[string][]string{"asset": {asset}}}
			rawRequest, _ := json.Marshal(request)
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
			if response.StatusCode != 200 || response.Headers.Get("Content-Type") != "font/woff2" {
				t.Fatalf("response = %#v", response)
			}
			if len(response.Body) < 80_000 || !strings.HasPrefix(string(response.Body), "wOF2") {
				t.Fatalf("%s is not the expected bundled WOFF2 font (%d bytes)", asset, len(response.Body))
			}
			if response.Headers.Get("ETag") == "" || !strings.Contains(response.Headers.Get("Cache-Control"), "immutable") {
				t.Fatalf("font cache headers = %#v", response.Headers)
			}
		})
	}
}

func TestManagementServesReadOnlyRestartStatus(t *testing.T) {
	resetRestartRuntimeForTest()
	t.Cleanup(resetRestartRuntimeForTest)
	request := managementRequest{Method: "GET", Query: map[string][]string{"asset": {"restart-status"}}}
	rawRequest, _ := json.Marshal(request)
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
	if response.StatusCode != http.StatusOK || response.Headers.Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("response = %#v", response)
	}
	if response.Headers.Get("Cache-Control") != "no-store" {
		t.Fatalf("cache control = %q", response.Headers.Get("Cache-Control"))
	}
	var status restartStatus
	if errUnmarshal := json.Unmarshal(response.Body, &status); errUnmarshal != nil {
		t.Fatal(errUnmarshal)
	}
	if status.ProcessInstance == "" {
		t.Fatal("restart status is missing its process instance")
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

func TestRegistrationKeepsStudioResourceOutOfSidebar(t *testing.T) {
	raw, errHandle := handleMethod(methodManagementRegister, nil)
	if errHandle != nil {
		t.Fatal(errHandle)
	}
	var env envelope
	if errUnmarshal := json.Unmarshal(raw, &env); errUnmarshal != nil {
		t.Fatal(errUnmarshal)
	}
	var registration managementRegistration
	if errUnmarshal := json.Unmarshal(env.Result, &registration); errUnmarshal != nil {
		t.Fatal(errUnmarshal)
	}
	if len(registration.Resources) != 1 || registration.Resources[0].Path != "/studio" {
		t.Fatalf("registration = %#v", registration)
	}
	if registration.Resources[0].Menu != "" {
		t.Fatalf("studio resource must stay hidden from the CPAMP sidebar: %#v", registration.Resources[0])
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
		"JetBrains Mono",
		"button('font','host'",
		"button('font','jetbrains-mono'",
		"button('font','system'",
		"min-height:0;overflow-x:hidden;overflow-y:auto",
		"scrollbar-color:color-mix(in srgb,var(--primary-color,#3b82f6) 32%",
		".ts-body::-webkit-scrollbar{width:10px;height:10px}",
		".ts-body::-webkit-scrollbar-thumb{border:2px solid transparent;border-radius:999px",
		".ts-body::-webkit-scrollbar-track{background:transparent}",
		"max-height:100dvh",
		"event.composedPath",
		"MutationObserver",
		"getDebugState",
		"bindHostThemeControl",
		"handleHostControlClick",
		"stopImmediatePropagation",
		"data-cpamp-theme-studio-trigger",
		"doc.querySelectorAll('button')",
		"data-cpamp-theme-studio-restart",
		"plugin-field-restart_request",
		"restart-status",
		"role', 'alertdialog",
		"restartConfirmTitle",
		"restartCancel",
		"waitForPanelLoader(12000)",
		"waitForRestartRequest(requestID, processInstance)",
		"cts-restart-feedback",
	} {
		if !strings.Contains(loader, required) {
			t.Fatalf("loader is missing runtime contract %q", required)
		}
	}
	if strings.Contains(loader, "var hostConnected = hostWindow !== window || !window.parent") {
		t.Fatal("standalone plugin resources must not be reported as an injected host panel")
	}
	if strings.Contains(loader, "if (opened) return") {
		t.Fatal("loader must not rely on stale open-state guards")
	}
	if strings.Contains(loader, "ts-launcher") {
		t.Fatal("loader must reuse CPAMP's native top-bar theme control instead of publishing a floating launcher")
	}
	if strings.Contains(loader, "managementKey") || strings.Contains(loader, "Authorization") {
		t.Fatal("loader must use CPAMP's authenticated config-save flow instead of reading or capturing the management key")
	}
	if strings.Contains(loader, "win.confirm(") {
		t.Fatal("loader must use its accessible restart confirmation dialog instead of a browser-native confirm")
	}
	if strings.Contains(loader, "win.alert(") {
		t.Fatal("loader must report restart errors without a blocking browser-native alert")
	}
}
