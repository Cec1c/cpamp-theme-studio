//go:build !windows

package bootstrap

import "os"

func effectiveUID() int { return os.Geteuid() }
