package bootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolvePanelRequiresPublicURLForManagerServer(t *testing.T) {
	directory := t.TempDir()
	panelPath := filepath.Join(directory, "management.html")
	if errWrite := os.WriteFile(panelPath, []byte("<html><head></head><body></body></html>"), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	cpa := ServiceInfo{WorkingDirectory: directory}
	manager := &ServiceInfo{WorkingDirectory: directory, Environment: map[string]string{"PANEL_PATH": panelPath}}
	_, _, _, errResolve := (Engine{}).resolvePanel(context.Background(), Options{}, cpa, manager, filepath.Join(directory, "config.yaml"))
	if errResolve == nil || !strings.Contains(errResolve.Error(), "require --panel-url") {
		t.Fatalf("resolvePanel() error = %v", errResolve)
	}
	_, _, _, errExplicit := (Engine{}).resolvePanel(
		context.Background(), Options{PanelPath: panelPath}, cpa, manager, filepath.Join(directory, "config.yaml"),
	)
	if errExplicit == nil || !strings.Contains(errExplicit.Error(), "require --panel-url") {
		t.Fatalf("resolvePanel() with explicit path error = %v", errExplicit)
	}
}

func TestResolvePanelConfirmsManagerPanelMatchesPublicURL(t *testing.T) {
	directory := t.TempDir()
	panelPath := filepath.Join(directory, "management.html")
	panel := []byte("<html><head></head><body>active</body></html>")
	if errWrite := os.WriteFile(panelPath, panel, 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write(panel)
	}))
	defer server.Close()
	cpa := ServiceInfo{WorkingDirectory: directory}
	manager := &ServiceInfo{WorkingDirectory: directory, Environment: map[string]string{"PANEL_PATH": panelPath}}
	resolved, seed, externalize, errResolve := (Engine{}).resolvePanel(
		context.Background(), Options{PanelURL: server.URL}, cpa, manager, filepath.Join(directory, "config.yaml"),
	)
	if errResolve != nil {
		t.Fatal(errResolve)
	}
	if resolved != panelPath || len(seed) != 0 || externalize {
		t.Fatalf("resolved=%q seed=%d externalize=%t", resolved, len(seed), externalize)
	}
}

func TestResolvePanelRejectsManagerPanelThatDiffersFromPublicURL(t *testing.T) {
	directory := t.TempDir()
	panelPath := filepath.Join(directory, "management.html")
	if errWrite := os.WriteFile(panelPath, []byte("<html><head></head><body>wrong</body></html>"), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write([]byte("<html><head></head><body>public</body></html>"))
	}))
	defer server.Close()
	cpa := ServiceInfo{WorkingDirectory: directory}
	manager := &ServiceInfo{WorkingDirectory: directory, Environment: map[string]string{"PANEL_PATH": panelPath}}
	_, _, _, errResolve := (Engine{}).resolvePanel(
		context.Background(), Options{PanelURL: server.URL}, cpa, manager, filepath.Join(directory, "config.yaml"),
	)
	if errResolve == nil || !strings.Contains(errResolve.Error(), "does not match --panel-url") {
		t.Fatalf("resolvePanel() error = %v", errResolve)
	}
}

func TestVerifierRejectsManagerDeploymentWithoutPublicPanelURL(t *testing.T) {
	state := State{ManagerService: "cpa-manager-plus.service"}
	errVerify := (Verifier{}).verifyHTTP(context.Background(), state, "0.2.1", true)
	if errVerify == nil || !strings.Contains(errVerify.Error(), "no public panel URL") {
		t.Fatalf("verifyHTTP() error = %v", errVerify)
	}
}

func TestWatchStableTimerCheckDoesNotRestart(t *testing.T) {
	directory := t.TempDir()
	configPath := filepath.Join(directory, "config.yaml")
	panelPath := filepath.Join(directory, "management.html")
	pluginDir := filepath.Join(directory, "plugins")
	if errMkdir := os.MkdirAll(pluginDir, 0o755); errMkdir != nil {
		t.Fatal(errMkdir)
	}
	if errWrite := os.WriteFile(filepath.Join(pluginDir, PluginID+"-v0.2.0.so"), []byte("fixture"), 0o755); errWrite != nil {
		t.Fatal(errWrite)
	}
	configRaw, configInfo, errConfig := ReassertBootstrapConfig([]byte(`plugins:
  enabled: true
  configs:
    cpamp-theme-studio:
      enabled: true
      store:
        version: 0.2.0
`), ConfigPatch{
		PluginsDir: pluginDir, PanelPath: panelPath, ConfigPath: configPath,
		RestartMode: "broker", RestartService: "cli-proxy-api.service",
	})
	if errConfig != nil {
		t.Fatal(errConfig)
	}
	if errWrite := os.WriteFile(configPath, configRaw, 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	files, errFiles := InventoryPlugins(pluginDir)
	if errFiles != nil {
		t.Fatal(errFiles)
	}
	statePath := filepath.Join(directory, "bootstrap.json")
	state := State{
		SchemaVersion: 1, CPAService: "cli-proxy-api.service", ConfigPath: configPath,
		PanelPath: panelPath, PluginsDir: pluginDir, PluginPlatformDir: pluginDir,
		RestartRequestPath: filepath.Join(directory, "restart.request"), StatePath: statePath,
		Accepted: NewAcceptedState(files, configInfo, (Engine{}).now()),
	}
	if errSave := SaveState(statePath, state); errSave != nil {
		t.Fatal(errSave)
	}
	if errWatch := (Engine{}).Watch(context.Background(), statePath); errWatch != nil {
		t.Fatalf("stable timer check reached restart path: %v", errWatch)
	}
}
