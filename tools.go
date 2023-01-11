package main

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
)

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Printf("could not open link: %v", err)
	}
}

func executable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

func execCommand(command string, verb string) ([]byte, error) {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command(command, verb).CombinedOutput()
	case "windows":
		return exec.Command(command, verb).CombinedOutput()
	case "linux":
		return exec.Command("pkexec", command, verb).CombinedOutput()
	default:
		return exec.Command(command, verb).CombinedOutput()
	}
}
