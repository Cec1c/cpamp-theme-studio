package bootstrap

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func CreateBackup(root, cpaService, managerService string, paths []string, now time.Time) (BackupManifest, error) {
	id := now.UTC().Format("20060102T150405.000000000Z")
	directory := filepath.Join(filepath.Clean(root), "backups", id)
	if errMkdir := os.MkdirAll(directory, 0o700); errMkdir != nil {
		return BackupManifest{}, fmt.Errorf("create backup directory: %w", errMkdir)
	}
	manifest := BackupManifest{SchemaVersion: 1, ID: id, CreatedAt: now.UTC().Format(time.RFC3339Nano), CPAService: cpaService, ManagerService: managerService}
	seen := make(map[string]struct{})
	for _, rawPath := range paths {
		path := filepath.Clean(strings.TrimSpace(rawPath))
		if path == "." || !filepath.IsAbs(path) {
			return BackupManifest{}, fmt.Errorf("backup path must be absolute: %q", rawPath)
		}
		if _, found := seen[path]; found {
			continue
		}
		seen[path] = struct{}{}
		entry := BackupEntry{Path: path}
		info, errInfo := os.Lstat(path)
		if errors.Is(errInfo, os.ErrNotExist) {
			manifest.Entries = append(manifest.Entries, entry)
			continue
		}
		if errInfo != nil {
			return BackupManifest{}, fmt.Errorf("inspect backup target %s: %w", path, errInfo)
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return BackupManifest{}, fmt.Errorf("backup target is not a regular non-symlink file: %s", path)
		}
		raw, errRead := os.ReadFile(path)
		if errRead != nil {
			return BackupManifest{}, fmt.Errorf("read backup target %s: %w", path, errRead)
		}
		entry.Existed = true
		entry.BackupName = fmt.Sprintf("%03d.bin", len(manifest.Entries))
		entry.Mode = info.Mode().Perm()
		entry.UID, entry.GID = fileOwner(info)
		entry.SHA256, _ = fileSHA256(path)
		if errWrite := writeFileAtomic(filepath.Join(directory, entry.BackupName), raw, 0o600, -1, -1); errWrite != nil {
			return BackupManifest{}, errWrite
		}
		manifest.Entries = append(manifest.Entries, entry)
	}
	if errSave := saveBackupManifest(directory, manifest); errSave != nil {
		return BackupManifest{}, errSave
	}
	return manifest, nil
}

func LoadBackup(root, id string) (BackupManifest, error) {
	if strings.TrimSpace(id) == "" || filepath.Base(id) != id {
		return BackupManifest{}, fmt.Errorf("invalid backup id %q", id)
	}
	directory := filepath.Join(filepath.Clean(root), "backups", id)
	raw, errRead := os.ReadFile(filepath.Join(directory, "manifest.json"))
	if errRead != nil {
		return BackupManifest{}, fmt.Errorf("read backup manifest: %w", errRead)
	}
	var manifest BackupManifest
	if errDecode := json.Unmarshal(raw, &manifest); errDecode != nil {
		return BackupManifest{}, fmt.Errorf("decode backup manifest: %w", errDecode)
	}
	if manifest.SchemaVersion != 1 || manifest.ID != id {
		return BackupManifest{}, fmt.Errorf("backup manifest does not match requested id")
	}
	return manifest, nil
}

func RestoreBackup(root string, manifest BackupManifest) error {
	directory := filepath.Join(filepath.Clean(root), "backups", manifest.ID)
	for _, entry := range manifest.Entries {
		path := filepath.Clean(entry.Path)
		if !filepath.IsAbs(path) {
			return fmt.Errorf("backup contains a non-absolute target path")
		}
		if !entry.Existed {
			if info, errInfo := os.Lstat(path); errInfo == nil {
				if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
					return fmt.Errorf("refusing to remove non-regular rollback target %s", path)
				}
				if errRemove := os.Remove(path); errRemove != nil {
					return fmt.Errorf("remove newly created file %s: %w", path, errRemove)
				}
			} else if !errors.Is(errInfo, os.ErrNotExist) {
				return fmt.Errorf("inspect rollback target %s: %w", path, errInfo)
			}
			continue
		}
		backupPath := filepath.Join(directory, entry.BackupName)
		digest, errDigest := fileSHA256(backupPath)
		if errDigest != nil {
			return errDigest
		}
		if digest != entry.SHA256 {
			return fmt.Errorf("backup SHA-256 mismatch for %s", path)
		}
		raw, errRead := os.ReadFile(backupPath)
		if errRead != nil {
			return errRead
		}
		if errWrite := writeFileAtomic(path, raw, entry.Mode, entry.UID, entry.GID); errWrite != nil {
			return errWrite
		}
	}
	return nil
}

func saveBackupManifest(directory string, manifest BackupManifest) error {
	raw, errMarshal := json.MarshalIndent(manifest, "", "  ")
	if errMarshal != nil {
		return fmt.Errorf("encode backup manifest: %w", errMarshal)
	}
	return writeFileAtomic(filepath.Join(directory, "manifest.json"), append(raw, '\n'), 0o600, -1, -1)
}
