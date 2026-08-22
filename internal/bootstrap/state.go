package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	PluginID                = "cpamp-theme-studio"
	DefaultStateDir         = "/var/lib/cpamp-theme-studio"
	DefaultStatePath        = "/etc/cpamp-theme-studio/bootstrap.json"
	DefaultInstalledBinary  = "/usr/local/libexec/cpamp-theme-studio/cpamp-theme-bootstrap"
	DefaultSystemdDir       = "/etc/systemd/system"
	DefaultRestartRequest   = "/run/cpamp-theme-studio/restart.request"
	DefaultPanelPath        = "/var/lib/cpamp-theme-studio/panel/management.html"
	DefaultPathUnit         = "cpamp-theme-studio-restart.path"
	DefaultTimerUnit        = "cpamp-theme-studio-restart.timer"
	DefaultServiceUnit      = "cpamp-theme-studio-restart.service"
	DefaultTmpfilesPath     = "/etc/tmpfiles.d/cpamp-theme-studio.conf"
	DefaultRuntimeDirectory = "/run/cpamp-theme-studio"
)

type State struct {
	SchemaVersion      int           `json:"schema_version"`
	BootstrapVersion   string        `json:"bootstrap_version"`
	CPAService         string        `json:"cpa_service"`
	ManagerService     string        `json:"manager_service,omitempty"`
	ConfigPath         string        `json:"config_path"`
	PanelPath          string        `json:"panel_path"`
	PanelURL           string        `json:"panel_url,omitempty"`
	HealthURL          string        `json:"health_url"`
	PluginsDir         string        `json:"plugins_dir"`
	PluginPlatformDir  string        `json:"plugin_platform_dir"`
	RestartRequestPath string        `json:"restart_request_path"`
	StateDir           string        `json:"state_dir"`
	StatePath          string        `json:"state_path"`
	InstalledBinary    string        `json:"installed_binary"`
	PathUnit           string        `json:"path_unit"`
	TimerUnit          string        `json:"timer_unit"`
	ServiceUnit        string        `json:"service_unit"`
	LastBackupID       string        `json:"last_backup_id,omitempty"`
	Accepted           AcceptedState `json:"accepted"`
}

type AcceptedState struct {
	Files               []PluginFile `json:"files"`
	DesiredVersion      string       `json:"desired_version,omitempty"`
	PluginConfigYAML    string       `json:"plugin_config_yaml,omitempty"`
	PluginConfigPresent bool         `json:"plugin_config_present"`
	AcceptedAt          string       `json:"accepted_at"`
}

type PluginFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type BackupManifest struct {
	SchemaVersion  int           `json:"schema_version"`
	ID             string        `json:"id"`
	CreatedAt      string        `json:"created_at"`
	CPAService     string        `json:"cpa_service"`
	ManagerService string        `json:"manager_service,omitempty"`
	Entries        []BackupEntry `json:"entries"`
}

type BackupEntry struct {
	Path       string      `json:"path"`
	Existed    bool        `json:"existed"`
	BackupName string      `json:"backup_name,omitempty"`
	Mode       os.FileMode `json:"mode,omitempty"`
	UID        int         `json:"uid,omitempty"`
	GID        int         `json:"gid,omitempty"`
	SHA256     string      `json:"sha256,omitempty"`
}

func LoadState(path string) (State, error) {
	raw, errRead := os.ReadFile(path)
	if errRead != nil {
		return State{}, fmt.Errorf("read bootstrap state: %w", errRead)
	}
	var state State
	if errDecode := json.Unmarshal(raw, &state); errDecode != nil {
		return State{}, fmt.Errorf("decode bootstrap state: %w", errDecode)
	}
	if state.SchemaVersion != 1 || state.CPAService == "" || state.ConfigPath == "" || state.PluginPlatformDir == "" || state.StatePath == "" {
		return State{}, fmt.Errorf("bootstrap state is incomplete or unsupported")
	}
	return state, nil
}

func SaveState(path string, state State) error {
	state.SchemaVersion = 1
	raw, errMarshal := json.MarshalIndent(state, "", "  ")
	if errMarshal != nil {
		return fmt.Errorf("encode bootstrap state: %w", errMarshal)
	}
	raw = append(raw, '\n')
	return writeFileAtomic(path, raw, 0o600, -1, -1)
}

func InventoryPlugins(dir string) ([]PluginFile, error) {
	entries, errRead := os.ReadDir(dir)
	if errRead != nil {
		if errors.Is(errRead, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read plugin directory: %w", errRead)
	}
	files := make([]PluginFile, 0)
	for _, entry := range entries {
		name := entry.Name()
		if name != PluginID+".so" && !(strings.HasPrefix(name, PluginID+"-v") && strings.HasSuffix(name, ".so")) {
			continue
		}
		path := filepath.Join(dir, name)
		info, errInfo := os.Lstat(path)
		if errInfo != nil {
			return nil, fmt.Errorf("inspect plugin %s: %w", path, errInfo)
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("plugin candidate is not a regular non-symlink file: %s", path)
		}
		digest, errDigest := fileSHA256(path)
		if errDigest != nil {
			return nil, errDigest
		}
		files = append(files, PluginFile{Path: filepath.Clean(path), SHA256: digest, Size: info.Size()})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func SameInventory(left, right []PluginFile) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func InventoryHasVersion(files []PluginFile, version string) bool {
	wanted := PluginID + "-v" + normalizeVersion(version) + ".so"
	for _, file := range files {
		if filepath.Base(file.Path) == wanted {
			return true
		}
	}
	return false
}

func RemoveUnacceptedPlugins(current, accepted []PluginFile) error {
	keep := make(map[string]string, len(accepted))
	for _, file := range accepted {
		keep[filepath.Clean(file.Path)] = file.SHA256
	}
	for _, file := range current {
		path := filepath.Clean(file.Path)
		if digest, found := keep[path]; found && digest == file.SHA256 {
			continue
		}
		if errRemove := os.Remove(path); errRemove != nil && !errors.Is(errRemove, os.ErrNotExist) {
			return fmt.Errorf("remove unaccepted plugin %s: %w", path, errRemove)
		}
	}
	return nil
}

func NewAcceptedState(files []PluginFile, info ConfigInfo, now time.Time) AcceptedState {
	return AcceptedState{
		Files:               append([]PluginFile(nil), files...),
		DesiredVersion:      info.DesiredVersion,
		PluginConfigYAML:    info.PluginConfigYAML,
		PluginConfigPresent: info.PluginConfigPresent,
		AcceptedAt:          now.UTC().Format(time.RFC3339Nano),
	}
}

func fileSHA256(path string) (string, error) {
	handle, errOpen := os.Open(path)
	if errOpen != nil {
		return "", fmt.Errorf("open %s for SHA-256: %w", path, errOpen)
	}
	defer handle.Close()
	digest := sha256.New()
	if _, errCopy := io.Copy(digest, handle); errCopy != nil {
		return "", fmt.Errorf("hash %s: %w", path, errCopy)
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}
