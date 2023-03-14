package main

import (
	"runtime"

	"golang.org/x/sys/windows"
)

// Windows Product Types
const (
	VER_NT_WORKSTATION = 1
	VER_NT_SERVER      = 3
)

func IsWindowsServer() bool {
	osName := runtime.GOOS
	// Determine if running on a Windows Server
	if osName != "windows" {
		return false
	}
	isWindowsServer := false
	if osName == "windows" {

		info := windows.RtlGetVersion()
		//var info windows.RTL_OSVERSIONINFOEX
		if info != nil {
			if info.ProductType == VER_NT_SERVER {
				return true
			}
		}
	}

	return isWindowsServer
}
