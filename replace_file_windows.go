//go:build windows

package main

import (
	"fmt"
	"syscall"
)

const (
	moveFileReplaceExisting = 0x1
	moveFileWriteThrough    = 0x8
)

var moveFileExW = syscall.NewLazyDLL("kernel32.dll").NewProc("MoveFileExW")

func replacePathAtomic(source, target string) error {
	sourceUTF16, errSource := syscall.UTF16PtrFromString(source)
	if errSource != nil {
		return errSource
	}
	targetUTF16, errTarget := syscall.UTF16PtrFromString(target)
	if errTarget != nil {
		return errTarget
	}
	result, _, callErr := moveFileExW.Call(
		uintptr(unsafePointer(sourceUTF16)),
		uintptr(unsafePointer(targetUTF16)),
		uintptr(moveFileReplaceExisting|moveFileWriteThrough),
	)
	if result == 0 {
		return fmt.Errorf("replace panel file: %w", callErr)
	}
	return nil
}
