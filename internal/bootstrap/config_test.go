package bootstrap

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestPatchConfigPreservesUnrelatedValuesAndAddsBootstrapContract(t *testing.T) {
	working := t.TempDir()
	configPath := filepath.Join(working, "config.yaml")
	panelPath := filepath.Join(working, "shared", "management.html")
	raw := []byte(`# keep this operator comment
port: 18318
proxy-url: http://127.0.0.1:7890
plugins:
  enabled: false
  dir: relative-plugins
  store-sources:
    - https://existing.example/registry.json
  configs:
    cpamp-theme-studio:
      enabled: false
      custom_operator_value: keep-me
      store:
        version: 0.1.5
`)
	patched, info, errPatch := PatchConfig(raw, ConfigPatch{
		PluginsDir: filepath.Join(working, "plugins"), PanelPath: panelPath, ConfigPath: configPath,
		RestartMode: "broker", RestartService: "cli-proxy-api.service",
	})
	if errPatch != nil {
		t.Fatal(errPatch)
	}
	text := string(patched)
	for _, expected := range []string{
		"# keep this operator comment", "proxy-url: http://127.0.0.1:7890", "custom_operator_value: keep-me",
		StoreSource, "restart_mode: broker", "restart_service: cli-proxy-api.service",
		"panel_path: " + panelPath, "host_config_path: " + configPath,
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("patched config is missing %q:\n%s", expected, text)
		}
	}
	if info.Port != 18318 || info.DesiredVersion != "0.1.5" || !info.PluginConfigPresent {
		t.Fatalf("patched info = %#v", info)
	}
	if info.PluginsDir != filepath.Join(working, "plugins") {
		t.Fatalf("plugins dir = %q", info.PluginsDir)
	}
}

func TestRestorePluginConfigChangesOnlyThemeStudioSubtree(t *testing.T) {
	accepted := []byte(`plugins:
  configs:
    cpamp-theme-studio:
      enabled: true
      store:
        version: 0.1.5
`)
	acceptedInfo, errAccepted := InspectConfig(accepted, t.TempDir())
	if errAccepted != nil {
		t.Fatal(errAccepted)
	}
	current := []byte(`unrelated: keep
plugins:
  configs:
    another-plugin:
      enabled: true
    cpamp-theme-studio:
      enabled: true
      store:
        version: 0.2.0
`)
	restored, errRestore := RestorePluginConfig(current, acceptedInfo.PluginConfigYAML, true)
	if errRestore != nil {
		t.Fatal(errRestore)
	}
	text := string(restored)
	for _, expected := range []string{"unrelated: keep", "another-plugin:", "version: 0.1.5"} {
		if !strings.Contains(text, expected) {
			t.Errorf("restored config is missing %q:\n%s", expected, text)
		}
	}
	if strings.Contains(text, "version: 0.2.0") {
		t.Fatalf("failed version remains:\n%s", text)
	}
}

func TestReassertBootstrapConfigPreservesMarketplaceFields(t *testing.T) {
	working := t.TempDir()
	configPath := filepath.Join(working, "config.yaml")
	panelPath := filepath.Join(working, "shared", "management.html")
	raw := []byte(`plugins:
  enabled: true
  dir: /old/plugins
  store-sources:
    - https://existing.example/registry.json
  configs:
    cpamp-theme-studio:
      enabled: false
      auto_inject: false
      priority: 10
      custom_operator_value: keep-me
      store:
        source_id: cpamp-theme-studio
        version: 0.2.0
        artifact: pinned.zip
      panel_path: /old/panel.html
      restart_mode: auto
      restart_service: old.service
`)
	reasserted, info, errReassert := ReassertBootstrapConfig(raw, ConfigPatch{
		PluginsDir: filepath.Join(working, "plugins"), PanelPath: panelPath, ConfigPath: configPath,
		RestartMode: "broker", RestartService: "cli-proxy-api.service",
	})
	if errReassert != nil {
		t.Fatal(errReassert)
	}
	text := string(reasserted)
	for _, expected := range []string{
		"dir: " + filepath.Join(working, "plugins"), StoreSource,
		"enabled: false", "auto_inject: false", "custom_operator_value: keep-me", "version: 0.2.0", "artifact: pinned.zip",
		"panel_path: " + panelPath, "host_config_path: " + configPath,
		"restart_mode: broker", "restart_service: cli-proxy-api.service",
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("reasserted config is missing %q:\n%s", expected, text)
		}
	}
	if strings.Contains(text, "panel_path: /old/panel.html") || strings.Contains(text, "restart_mode: auto") {
		t.Fatalf("stale bootstrap settings remain:\n%s", text)
	}
	if info.DesiredVersion != "0.2.0" || !info.PluginConfigPresent {
		t.Fatalf("reasserted info = %#v", info)
	}
	repeated, _, errRepeated := ReassertBootstrapConfig(reasserted, ConfigPatch{
		PluginsDir: filepath.Join(working, "plugins"), PanelPath: panelPath, ConfigPath: configPath,
		RestartMode: "broker", RestartService: "cli-proxy-api.service",
	})
	if errRepeated != nil {
		t.Fatal(errRepeated)
	}
	if !bytes.Equal(reasserted, repeated) {
		t.Fatal("reassertion is not idempotent and would retrigger the broker timer")
	}
}
