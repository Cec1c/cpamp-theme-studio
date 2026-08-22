package main

import (
	"fmt"
	"strings"
	"testing"
)

func preserveRestartHooks(t *testing.T) {
	t.Helper()
	originalOS := restartOS
	originalPID := restartPID
	originalExecutable := restartExecutable
	originalReadFile := restartReadFile
	originalExit := restartExit
	originalCommand := restartCommand
	originalSchedule := restartSchedule
	t.Cleanup(func() {
		restartOS = originalOS
		restartPID = originalPID
		restartExecutable = originalExecutable
		restartReadFile = originalReadFile
		restartExit = originalExit
		restartCommand = originalCommand
		restartSchedule = originalSchedule
		resetRestartRuntimeForTest()
	})
}

func TestPrepareRestartPlanUsesValidatedCurrentSystemdService(t *testing.T) {
	preserveRestartHooks(t)
	restartOS = "linux"
	restartPID = func() int { return 4242 }
	restartExecutable = func() (string, error) { return "/opt/cpa/cli-proxy-api", nil }
	restartReadFile = func(string) ([]byte, error) { return nil, fmt.Errorf("not available") }
	var restarted []string
	restartCommand = func(name string, args ...string) ([]byte, error) {
		if name != "systemctl" {
			t.Fatalf("command = %q", name)
		}
		if len(args) > 0 && args[0] == "show" {
			return []byte("4242\n"), nil
		}
		restarted = append(restarted, strings.Join(args, " "))
		return nil, nil
	}

	plan, errPlan := prepareRestartPlan(injectorConfig{RestartMode: restartModeAuto})
	if errPlan != nil {
		t.Fatal(errPlan)
	}
	if plan.mode != restartModeSystemd {
		t.Fatalf("mode = %q", plan.mode)
	}
	if errRun := plan.run(); errRun != nil {
		t.Fatal(errRun)
	}
	if len(restarted) != 1 || restarted[0] != "--no-block restart cli-proxy-api.service" {
		t.Fatalf("restart commands = %#v", restarted)
	}
}

func TestPrepareRestartPlanRejectsUnrelatedSystemdService(t *testing.T) {
	preserveRestartHooks(t)
	restartOS = "linux"
	restartPID = func() int { return 4242 }
	restartExecutable = func() (string, error) { return "/opt/cpa/cli-proxy-api", nil }
	restartReadFile = func(string) ([]byte, error) { return nil, fmt.Errorf("not available") }
	restartCommand = func(string, ...string) ([]byte, error) { return []byte("9999\n"), nil }

	if _, errPlan := prepareRestartPlan(injectorConfig{RestartMode: restartModeSystemd, RestartService: "another.service"}); errPlan == nil {
		t.Fatal("expected MainPID mismatch to reject the systemd service")
	}
}

func TestPrepareRestartPlanSelfExitIsExplicit(t *testing.T) {
	preserveRestartHooks(t)
	var exitCode int
	restartExit = func(code int) { exitCode = code }

	plan, errPlan := prepareRestartPlan(injectorConfig{RestartMode: restartModeSelfExit})
	if errPlan != nil {
		t.Fatal(errPlan)
	}
	if errRun := plan.run(); errRun == nil {
		t.Fatal("a stubbed process exit must report that it returned")
	}
	if exitCode != restartExitCode {
		t.Fatalf("exit code = %d, want %d", exitCode, restartExitCode)
	}
}

func TestConfigureRestartOnlySchedulesNewReconfigureRequest(t *testing.T) {
	preserveRestartHooks(t)
	resetRestartRuntimeForTest()
	var scheduled []func()
	restartSchedule = func(action func()) { scheduled = append(scheduled, action) }
	restartExit = func(int) {}
	cfg := injectorConfig{RestartMode: restartModeSelfExit, RestartRequest: "baseline"}

	configureRestart(cfg, false)
	if len(scheduled) != 0 {
		t.Fatalf("register scheduled %d restart(s)", len(scheduled))
	}
	cfg.RestartRequest = "request-2"
	configureRestart(cfg, true)
	if len(scheduled) != 1 {
		t.Fatalf("new reconfigure request scheduled %d restart(s), want 1", len(scheduled))
	}
	status := snapshotRestartStatus()
	if status.State != "scheduled" || status.Request != "request-2" || !status.Available {
		t.Fatalf("status = %#v", status)
	}
	configureRestart(cfg, true)
	if len(scheduled) != 1 {
		t.Fatalf("duplicate request scheduled %d restart(s), want 1", len(scheduled))
	}
}

func TestConfigureRestartReportsDisabledModeWithoutScheduling(t *testing.T) {
	preserveRestartHooks(t)
	resetRestartRuntimeForTest()
	restartSchedule = func(func()) { t.Fatal("disabled mode must not schedule a restart") }

	configureRestart(injectorConfig{RestartMode: restartModeDisabled, RestartRequest: "request-1"}, true)
	status := snapshotRestartStatus()
	if status.State != "failed" || status.Available || status.Message == "" {
		t.Fatalf("status = %#v", status)
	}
}
