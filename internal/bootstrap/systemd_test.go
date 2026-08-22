package bootstrap

import (
	"strings"
	"testing"
)

func TestSystemdUnitsBindOnlySelectedCPAAndBrokerPaths(t *testing.T) {
	state := State{
		CPAService: "cli-proxy-api.service", ConfigPath: "/opt/cpa/config.yaml",
		PanelPath:         "/var/lib/cpamp-theme-studio/panel/management.html",
		PluginPlatformDir: "/opt/cpa/plugins/linux/amd64", RestartRequestPath: DefaultRestartRequest,
		StateDir: DefaultStateDir, StatePath: DefaultStatePath, InstalledBinary: DefaultInstalledBinary,
		PathUnit: DefaultPathUnit, ServiceUnit: DefaultServiceUnit,
	}
	pathUnit := string(PathUnitContents(state))
	serviceUnit := string(ServiceUnitContents(state))
	timerUnit := string(TimerUnitContents(state))
	for _, expected := range []string{
		"After=cli-proxy-api.service", "PathChanged=/opt/cpa/plugins/linux/amd64",
		"PathExists=/run/cpamp-theme-studio/restart.request", "Unit=cpamp-theme-studio-restart.service",
	} {
		if !strings.Contains(pathUnit, expected) {
			t.Errorf("path unit is missing %q:\n%s", expected, pathUnit)
		}
	}
	for _, expected := range []string{"NoNewPrivileges=true", "ProtectSystem=strict", " watch --state ", "/opt/cpa/plugins/linux/amd64"} {
		if !strings.Contains(serviceUnit, expected) {
			t.Errorf("service unit is missing %q:\n%s", expected, serviceUnit)
		}
	}
	if strings.Contains(serviceUnit, "bash -c") || strings.Contains(serviceUnit, "sudo") {
		t.Fatalf("broker unit exposes a shell or sudo path:\n%s", serviceUnit)
	}
	if strings.Contains(serviceUnit, "PrivateTmp=true") {
		t.Fatalf("PrivateTmp would hide valid CPA paths under /tmp:\n%s", serviceUnit)
	}
	for _, expected := range []string{"OnUnitInactiveSec=15s", "Unit=cpamp-theme-studio-restart.service", "WantedBy=timers.target"} {
		if !strings.Contains(timerUnit, expected) {
			t.Errorf("timer unit is missing %q:\n%s", expected, timerUnit)
		}
	}
}

func TestVerifyPanelMarkersRejectsDuplicatesAndWrongVersion(t *testing.T) {
	valid := []byte(loaderStartMarker + `<script src="?v=0.2.0"></script>` + loaderEndMarker)
	if errValid := verifyPanelMarkers(valid, "0.2.0"); errValid != nil {
		t.Fatal(errValid)
	}
	if errDuplicate := verifyPanelMarkers(append(valid, valid...), "0.2.0"); errDuplicate == nil {
		t.Fatal("duplicate markers were accepted")
	}
	if errVersion := verifyPanelMarkers(valid, "0.2.1"); errVersion == nil {
		t.Fatal("wrong loader version was accepted")
	}
}
