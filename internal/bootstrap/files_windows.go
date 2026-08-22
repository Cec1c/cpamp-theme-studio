//go:build windows

package bootstrap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func writeFileAtomic(path string, data []byte, mode os.FileMode, _ int, _ int) error {
	path = filepath.Clean(path)
	parent := filepath.Dir(path)
	if errMkdir := os.MkdirAll(parent, 0o755); errMkdir != nil {
		return errMkdir
	}
	if info, errLstat := os.Lstat(path); errLstat == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("refusing to replace non-regular or symlink path %s", path)
		}
	} else if !errors.Is(errLstat, os.ErrNotExist) {
		return errLstat
	}
	temp, errCreate := os.CreateTemp(parent, ".cpamp-theme-studio-*")
	if errCreate != nil {
		return errCreate
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, errWrite := temp.Write(data); errWrite != nil {
		temp.Close()
		return errWrite
	}
	if errClose := temp.Close(); errClose != nil {
		return errClose
	}
	if errChmod := os.Chmod(tempPath, mode.Perm()); errChmod != nil {
		return errChmod
	}
	return os.Rename(tempPath, path)
}

func fileOwner(os.FileInfo) (int, int) { return -1, -1 }
