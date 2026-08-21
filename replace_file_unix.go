//go:build !windows

package main

import "os"

func replacePathAtomic(source, target string) error {
	return os.Rename(source, target)
}
