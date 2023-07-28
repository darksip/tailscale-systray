package main

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
)

// mettre le root Url dans un .env

// TODO: Attention il faut tenir compte de l'OS pour le chemin de l'executable
// on doit le mettre sous cette forme pour l'execution sous windows sinon il refuse
// l'execution
//
//	https://github.com/golang/go/issues/43724

func openBrowser(url string) error {
	log.Printf("open url : %s", url)
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		var cmd *exec.Cmd
		if browserMethod != "CMD" {
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		} else {
			cmd = exec.Command("cmd", "/c", "start", url)
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		}
		err = cmd.Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Printf("could not open link: %v", err.Error())
	}
	return err
}

// dont forget : go build -ldflags="-H windowsgui"
func execCommand(command string, verb ...string) ([]byte, error) {
	log.Printf("exec command for %s : %s", runtime.GOOS, command)
	log.Printf("args : %s", strings.Join(verb, " "))
	//path, err := exec.LookPath(command)

	switch runtime.GOOS {
	case "darwin":
		return exec.Command(command, verb...).CombinedOutput()
	case "windows":
		cmd := exec.Command(command, verb...)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		return cmd.CombinedOutput()
	case "linux":
		allverbs := append([]string{command}, verb...)
		return exec.Command("pkexec", allverbs...).CombinedOutput()
	default:
		return exec.Command(command, verb...).CombinedOutput()
	}
}

func execNewGroupCommand(command string, verb ...string) ([]byte, error) {
	cmd := exec.Command(command, verb...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
	return cmd.CombinedOutput()
}
