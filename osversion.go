package main

import (
	"runtime"
)

// Windows Product Types
const (
	VER_NT_WORKSTATION = 1
	VER_NT_SERVER      = 3
)

func IsMacOs() bool {
	osName := runtime.GOOS
	return osName == "darwin"
}

func IsWindowsServer() bool {
	return false
}
