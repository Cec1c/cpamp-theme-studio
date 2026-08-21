//go:build windows

package main

import "unsafe"

func unsafePointer(value *uint16) unsafe.Pointer {
	return unsafe.Pointer(value)
}
