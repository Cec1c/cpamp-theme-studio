package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Cec1c/cpamp-theme-studio/internal/bootstrap"
)

var bootstrapVersion = "0.0.0-dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "watch" {
		watch(os.Args[2:])
		return
	}
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("cpamp-theme-bootstrap", flag.ContinueOnError)
	apply := flags.Bool("apply", false, "apply the inspected bootstrap plan")
	rollback := flags.String("rollback", "", "restore a bootstrap backup by id")
	service := flags.String("service", "", "CPA systemd service (auto-detected when unique)")
	managerService := flags.String("manager-service", "", "CPAMP Manager Server systemd service")
	configPath := flags.String("config", "", "absolute CPA config.yaml path")
	panelPath := flags.String("panel-path", "", "absolute active management.html path")
	panelURL := flags.String("panel-url", "", "public management.html URL used for confirmation/externalization")
	pluginsDir := flags.String("plugins-dir", "", "absolute CPA plugins.dir override")
	healthURL := flags.String("health-url", "", "loopback CPA base URL (default derives from config port)")
	statePath := flags.String("state", bootstrap.DefaultStatePath, "bootstrap state file")
	stateDir := flags.String("state-dir", bootstrap.DefaultStateDir, "bootstrap data and backup directory")
	showVersion := flags.Bool("version", false, "print bootstrap version")
	if errParse := flags.Parse(args); errParse != nil {
		return errParse
	}
	if *showVersion {
		fmt.Println(bootstrapVersion)
		return nil
	}
	executable, errExecutable := os.Executable()
	if errExecutable != nil {
		return fmt.Errorf("resolve bootstrap executable: %w", errExecutable)
	}
	options := bootstrap.Options{
		BootstrapVersion: bootstrapVersion,
		Service:          strings.TrimSpace(*service), ManagerService: strings.TrimSpace(*managerService),
		ConfigPath: strings.TrimSpace(*configPath), PanelPath: strings.TrimSpace(*panelPath), PanelURL: strings.TrimSpace(*panelURL),
		PluginsDir: strings.TrimSpace(*pluginsDir), HealthURL: strings.TrimSpace(*healthURL),
		StatePath: filepath.Clean(*statePath), StateDir: filepath.Clean(*stateDir), Executable: filepath.Clean(executable),
		RequireRoot: *apply || strings.TrimSpace(*rollback) != "",
	}
	engine := bootstrap.Engine{Systemd: bootstrap.Systemd{Runner: bootstrap.ExecRunner{}}}
	ctx := context.Background()
	if backupID := strings.TrimSpace(*rollback); backupID != "" {
		if *apply {
			return fmt.Errorf("--apply and --rollback are mutually exclusive")
		}
		if errRollback := engine.Rollback(ctx, options, backupID); errRollback != nil {
			return errRollback
		}
		fmt.Printf("Rollback %s completed; the CPA service recorded in that backup was restarted.\n", backupID)
		return nil
	}
	plan, errPlan := engine.Resolve(ctx, options)
	if errPlan != nil {
		return errPlan
	}
	printPlan(plan, *apply)
	if !*apply {
		fmt.Println("Dry run only. Re-run the same command with --apply after reviewing this plan.")
		return nil
	}
	state, errApply := engine.Apply(ctx, plan)
	if errApply != nil {
		return errApply
	}
	fmt.Printf("Bootstrap applied successfully. Backup: %s\n", state.LastBackupID)
	fmt.Println("Future CPAMP marketplace installs/upgrades and confirmed card requests will restart and verify CPA automatically.")
	return nil
}

func watch(args []string) {
	flags := flag.NewFlagSet("watch", flag.ContinueOnError)
	statePath := flags.String("state", bootstrap.DefaultStatePath, "bootstrap state file")
	if errParse := flags.Parse(args); errParse != nil {
		os.Exit(2)
	}
	engine := bootstrap.Engine{Systemd: bootstrap.Systemd{Runner: bootstrap.ExecRunner{}}}
	if errWatch := engine.Watch(context.Background(), filepath.Clean(*statePath)); errWatch != nil {
		fmt.Fprintln(os.Stderr, "cpamp-theme-studio restart broker:", errWatch)
		os.Exit(1)
	}
}

func printPlan(plan bootstrap.Plan, applying bool) {
	mode := "DRY-RUN"
	if applying {
		mode = "APPLY"
	}
	manager := "none (CPA lightweight panel)"
	if plan.Manager != nil {
		manager = plan.Manager.Unit
	}
	fmt.Printf("CPAMP Theme Studio bootstrap plan [%s]\n", mode)
	fmt.Printf("  CPA service:       %s (PID %d)\n", plan.CPA.Unit, plan.CPA.MainPID)
	fmt.Printf("  Manager service:   %s\n", manager)
	fmt.Printf("  CPA config:        %s\n", plan.ConfigPath)
	fmt.Printf("  Active panel:      %s\n", plan.PanelPath)
	fmt.Printf("  Public panel URL:  %s\n", emptyAsNone(plan.Options.PanelURL))
	fmt.Printf("  Plugin directory:  %s\n", plan.PluginPlatformDir)
	fmt.Printf("  Restart mechanism: root-owned systemd path broker for %s only\n", plan.CPA.Unit)
	fmt.Printf("  Externalize panel: %t\n", plan.ExternalizePanel)
}

func emptyAsNone(value string) string {
	if strings.TrimSpace(value) == "" {
		return "not configured"
	}
	return value
}
