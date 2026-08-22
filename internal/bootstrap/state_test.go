package bootstrap

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInventoryPluginsIsScopedAndVersionAware(t *testing.T) {
	directory := t.TempDir()
	for name, body := range map[string]string{
		"cpamp-theme-studio-v0.1.5.so": "old",
		"cpamp-theme-studio-v0.2.0.so": "new",
		"another-plugin.so":            "ignored",
	} {
		if errWrite := os.WriteFile(filepath.Join(directory, name), []byte(body), 0o755); errWrite != nil {
			t.Fatal(errWrite)
		}
	}
	files, errInventory := InventoryPlugins(directory)
	if errInventory != nil {
		t.Fatal(errInventory)
	}
	if len(files) != 2 || !InventoryHasVersion(files, "v0.2.0") {
		t.Fatalf("inventory = %#v", files)
	}
	accepted := []PluginFile{files[0]}
	if errRemove := RemoveUnacceptedPlugins(files, accepted); errRemove != nil {
		t.Fatal(errRemove)
	}
	if _, errStat := os.Stat(files[1].Path); !os.IsNotExist(errStat) {
		t.Fatalf("unaccepted plugin was not removed: %v", errStat)
	}
	if _, errStat := os.Stat(filepath.Join(directory, "another-plugin.so")); errStat != nil {
		t.Fatalf("unrelated plugin was changed: %v", errStat)
	}
}

func TestBackupRoundTripRestoresAndRemovesOnlyRecordedFiles(t *testing.T) {
	root := t.TempDir()
	existing := filepath.Join(root, "config.yaml")
	created := filepath.Join(root, "new.conf")
	if errWrite := os.WriteFile(existing, []byte("before\n"), 0o640); errWrite != nil {
		t.Fatal(errWrite)
	}
	manifest, errBackup := CreateBackup(root, "cli-proxy-api.service", "", []string{existing, created}, time.Unix(123, 0))
	if errBackup != nil {
		t.Fatal(errBackup)
	}
	if errWrite := os.WriteFile(existing, []byte("after\n"), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	if errWrite := os.WriteFile(created, []byte("created\n"), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	if errRestore := RestoreBackup(root, manifest); errRestore != nil {
		t.Fatal(errRestore)
	}
	raw, _ := os.ReadFile(existing)
	if string(raw) != "before\n" {
		t.Fatalf("restored content = %q", raw)
	}
	if _, errStat := os.Stat(created); !os.IsNotExist(errStat) {
		t.Fatalf("new rollback target remains: %v", errStat)
	}
}
