//go:build !windows

package bootstrap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func writeFileAtomic(path string, data []byte, mode os.FileMode, uid int, gid int) error {
	path = filepath.Clean(path)
	parent := filepath.Dir(path)
	if errMkdir := os.MkdirAll(parent, 0o755); errMkdir != nil {
		return fmt.Errorf("create parent directory for %s: %w", path, errMkdir)
	}
	if info, errLstat := os.Lstat(path); errLstat == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("refusing to replace non-regular or symlink path %s", path)
		}
	} else if !errors.Is(errLstat, os.ErrNotExist) {
		return fmt.Errorf("inspect %s: %w", path, errLstat)
	}
	temp, errCreate := os.CreateTemp(parent, ".cpamp-theme-studio-*")
	if errCreate != nil {
		return fmt.Errorf("create temporary file for %s: %w", path, errCreate)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, errWrite := temp.Write(data); errWrite != nil {
		temp.Close()
		return fmt.Errorf("write temporary file for %s: %w", path, errWrite)
	}
	if errSync := temp.Sync(); errSync != nil {
		temp.Close()
		return fmt.Errorf("sync temporary file for %s: %w", path, errSync)
	}
	if errChmod := temp.Chmod(mode.Perm()); errChmod != nil {
		temp.Close()
		return fmt.Errorf("set mode for %s: %w", path, errChmod)
	}
	if uid >= 0 || gid >= 0 {
		if errChown := temp.Chown(uid, gid); errChown != nil {
			temp.Close()
			return fmt.Errorf("set ownership for %s: %w", path, errChown)
		}
	}
	if errClose := temp.Close(); errClose != nil {
		return fmt.Errorf("close temporary file for %s: %w", path, errClose)
	}
	if errRename := os.Rename(tempPath, path); errRename != nil {
		return fmt.Errorf("replace %s: %w", path, errRename)
	}
	return nil
}

func fileOwner(info os.FileInfo) (int, int) {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return int(stat.Uid), int(stat.Gid)
	}
	return -1, -1
}
