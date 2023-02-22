package main

import (
	"strings"

	"github.com/darksip/tailscale-systray/sysmenu"
	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"
)

var (
	menuExitNode *systray.MenuItem
)
var sm sysmenu.SysMenu

func Notify(message string) {
	if strings.Contains(strings.ToLower(message), "tailscale") {
		message = strings.ReplaceAll(strings.ToLower(message), "tailscale", "cybervpn")
	}
	beeep.Notify(
		"Cyber Vpn",
		string(message),
		"./icon/on.png",
	)
}

func setExitNodeOff() {
	for {
		if _, ok := <-menuExitNode.ClickedCh; !ok {
			break
		}
		if len(activeExitNode) > 0 {
			removeExitNode()
			menuExitNode.SetTitle("Set Exit Node On")
		} else {
			setExitNode()
		}
	}

}
