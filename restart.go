package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	restartModeAuto     = "auto"
	restartModeDisabled = "disabled"
	restartModeSystemd  = "systemd"
	restartModeSelfExit = "self-exit"

	maxRestartRequestLength = 128
	restartExitCode         = 75
)

type restartStatus struct {
	ProcessInstance string `json:"process_instance"`
	Request         string `json:"request,omitempty"`
	State           string `json:"state"`
	Mode            string `json:"mode"`
	Available       bool   `json:"available"`
	Message         string `json:"message,omitempty"`
	UpdatedAt       string `json:"updated_at"`
}

type restartPlan struct {
	mode string
	run  func() error
}

var (
	restartServicePattern = regexp.MustCompile(`^[A-Za-z0-9_.@-]+\.service$`)
	restartOS             = runtime.GOOS
	restartPID            = os.Getpid
	restartExecutable     = os.Executable
	restartReadFile       = os.ReadFile
	restartExit           = os.Exit
	restartCommand        = func(name string, args ...string) ([]byte, error) {
		return exec.Command(name, args...).CombinedOutput()
	}
	restartSchedule = func(action func()) {
		time.AfterFunc(1500*time.Millisecond, action)
	}
)

var restartRuntime = struct {
	sync.Mutex
	lastRequest string
	pending     bool
	status      restartStatus
}{status: newRestartStatus()}

func validRestartMode(mode string) bool {
	switch mode {
	case restartModeAuto, restartModeDisabled, restartModeSystemd, restartModeSelfExit:
		return true
	default:
		return false
	}
}

func newRestartStatus() restartStatus {
	return restartStatus{
		ProcessInstance: randomProcessInstance(),
		State:           "initializing",
		Mode:            restartModeAuto,
		UpdatedAt:       time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func randomProcessInstance() string {
	raw := make([]byte, 12)
	if _, errRead := rand.Read(raw); errRead == nil {
		return hex.EncodeToString(raw)
	}
	return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
}

func configureRestart(cfg injectorConfig, isReconfigure bool) {
	mode := strings.TrimSpace(cfg.RestartMode)
	if mode == "" {
		mode = restartModeAuto
	}

	restartRuntime.Lock()
	previousRequest := restartRuntime.lastRequest
	restartRuntime.lastRequest = cfg.RestartRequest
	shouldRestart := isReconfigure && cfg.RestartRequest != "" && cfg.RestartRequest != previousRequest
	pending := restartRuntime.pending
	restartRuntime.Unlock()

	if !shouldRestart {
		_, errPlan := prepareRestartPlan(cfg)
		updateRestartStatus(func(status *restartStatus) {
			status.Request = cfg.RestartRequest
			status.Mode = mode
			status.Available = errPlan == nil
			if errPlan == nil {
				status.State = "ready"
				status.Message = ""
			} else {
				status.State = "unavailable"
				status.Message = publicRestartError(errPlan)
			}
		})
		return
	}

	if pending {
		updateRestartStatus(func(status *restartStatus) {
			status.Request = cfg.RestartRequest
			status.Mode = mode
			status.Available = false
			status.State = "failed"
			status.Message = "A CPA restart is already in progress."
		})
		return
	}

	plan, errPlan := prepareRestartPlan(cfg)
	if errPlan != nil {
		updateRestartStatus(func(status *restartStatus) {
			status.Request = cfg.RestartRequest
			status.Mode = mode
			status.Available = false
			status.State = "failed"
			status.Message = publicRestartError(errPlan)
		})
		return
	}

	restartRuntime.Lock()
	restartRuntime.pending = true
	restartRuntime.status.Request = cfg.RestartRequest
	restartRuntime.status.Mode = plan.mode
	restartRuntime.status.Available = true
	restartRuntime.status.State = "scheduled"
	restartRuntime.status.Message = "CPA restart is scheduled."
	restartRuntime.status.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	restartRuntime.Unlock()

	restartSchedule(func() {
		updateRestartStatus(func(status *restartStatus) {
			status.State = "executing"
			status.Message = "CPA restart is executing."
		})
		if errRun := plan.run(); errRun != nil {
			restartRuntime.Lock()
			restartRuntime.pending = false
			restartRuntime.status.Available = false
			restartRuntime.status.State = "failed"
			restartRuntime.status.Message = "The restart command failed. Check CPA service permissions and logs."
			restartRuntime.status.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
			restartRuntime.Unlock()
			return
		}
		updateRestartStatus(func(status *restartStatus) {
			status.State = "accepted"
			status.Message = "The restart command was accepted."
		})
	})
}

func prepareRestartPlan(cfg injectorConfig) (restartPlan, error) {
	mode := strings.TrimSpace(cfg.RestartMode)
	if mode == "" {
		mode = restartModeAuto
	}
	switch mode {
	case restartModeDisabled:
		return restartPlan{}, fmt.Errorf("automatic restart is disabled")
	case restartModeSelfExit:
		return restartPlan{mode: restartModeSelfExit, run: func() error {
			restartExit(restartExitCode)
			return fmt.Errorf("process exit returned unexpectedly")
		}}, nil
	case restartModeAuto, restartModeSystemd:
		if restartOS != "linux" {
			return restartPlan{}, fmt.Errorf("systemd restart is unavailable on %s", restartOS)
		}
		service, errService := resolveSystemdService(cfg.RestartService)
		if errService != nil {
			return restartPlan{}, errService
		}
		return restartPlan{mode: restartModeSystemd, run: func() error {
			output, errCommand := restartCommand("systemctl", "--no-block", "restart", service)
			if errCommand != nil {
				return fmt.Errorf("restart current systemd service: %w: %s", errCommand, strings.TrimSpace(string(output)))
			}
			return nil
		}}, nil
	default:
		return restartPlan{}, fmt.Errorf("unsupported restart mode %q", mode)
	}
}

func resolveSystemdService(configured string) (string, error) {
	candidates := []string{strings.TrimSpace(configured), strings.TrimSpace(os.Getenv("CPAMP_THEME_STUDIO_SYSTEMD_SERVICE"))}
	if raw, errRead := restartReadFile("/proc/self/cgroup"); errRead == nil {
		for _, segment := range strings.FieldsFunc(string(raw), func(r rune) bool { return r == '/' || r == '\n' || r == ':' }) {
			segment = strings.TrimSpace(segment)
			if strings.HasSuffix(segment, ".service") {
				candidates = append(candidates, segment)
			}
		}
	}
	if executable, errExecutable := restartExecutable(); errExecutable == nil && strings.HasPrefix(strings.ToLower(filepath.Base(executable)), "cli-proxy-api") {
		candidates = append(candidates, "cli-proxy-api.service")
	}

	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || !restartServicePattern.MatchString(candidate) {
			continue
		}
		if _, found := seen[candidate]; found {
			continue
		}
		seen[candidate] = struct{}{}
		output, errShow := restartCommand("systemctl", "show", "--property=MainPID", "--value", candidate)
		if errShow != nil {
			continue
		}
		mainPID, errPID := strconv.Atoi(strings.TrimSpace(string(output)))
		if errPID == nil && mainPID == restartPID() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no systemd service owned by the current CPA process was found")
}

func publicRestartError(err error) string {
	if err == nil {
		return ""
	}
	if strings.Contains(err.Error(), "disabled") {
		return "Automatic restart is disabled in the plugin configuration."
	}
	return "Automatic restart is unavailable for this CPA deployment. Configure restart_mode or restart_service."
}

func updateRestartStatus(update func(*restartStatus)) {
	restartRuntime.Lock()
	defer restartRuntime.Unlock()
	update(&restartRuntime.status)
	restartRuntime.status.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
}

func snapshotRestartStatus() restartStatus {
	restartRuntime.Lock()
	defer restartRuntime.Unlock()
	return restartRuntime.status
}

func resetRestartRuntimeForTest() {
	restartRuntime.Lock()
	restartRuntime.lastRequest = ""
	restartRuntime.pending = false
	restartRuntime.status = newRestartStatus()
	restartRuntime.Unlock()
}
