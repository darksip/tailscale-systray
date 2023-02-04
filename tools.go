package main

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
)

var rootUrl = "https://head.juvise.cyberfile.fr"
var adminUrl = rootUrl + "/web"
var appName = "CyberVpn"

// TODO: Attention il faut tenir compte de l'OS pour le chemin de l'executable
// on doit le mettre sous cette forme pour l'execution sous windows sinon il refuse
// l'execution
//
//	https://github.com/golang/go/issues/43724
var cliExecutable = ".\\cybervpn-cli.exe"

func openBrowser(url string) {
	log.Printf("open url : %s", url)
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		//err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
		cmd := exec.Command("cmd", "/c", "start", url)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		cmd.Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Printf("could not open link: %v", err.Error())
	}
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
