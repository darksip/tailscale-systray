package main

import (
	"strings"

	"github.com/darksip/tailscale-systray/sysmenu"
	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"
)

var (
	//menuExitNode *systray.MenuItem
	menuItems    map[string]*systray.MenuItem
	itemsHandler map[string]*sysmenu.EvtHnd
)

func RunGl() {
	initGLIcons()
	systray.Run(onReady, nil)
}

func onReady() {
	SetupMenuGL()
	onMenuReady()
}

func SetupMenuGL() {

	menuItems = map[string]*systray.MenuItem{}
	hide = func(id string, v bool) {
		if mi, ok := menuItems[id]; ok == true {
			if v {
				mi.Hide()
			} else {
				mi.Show()
			}
		}
	}
	enable = func(id string, v bool) {
		if mi, ok := menuItems[id]; ok == true {
			if v {
				mi.Enable()
			} else {
				mi.Disable()
			}
		}
	}
	add = func(e sysmenu.Melt) {
		if e.Separator {
			systray.AddSeparator()
			return
		}
		mi := systray.AddMenuItem(e.Label, "")
		if e.Disabled {
			mi.Disable()
		}
		if e.Hidden {
			mi.Hide()
		}
		menuItems[e.Id] = mi
	}
	sethnd = func(id string, e sysmenu.EvtHnd) {
		if mi, ok := menuItems[id]; ok == true {
			if _, ok := itemsHandler[id]; !ok {
				go waitForClick(mi, e)
			}
		}
	}
	setlbl = func(id string, l string) {
		if mi, ok := menuItems[id]; ok == true {
			mi.SetTitle(l)
		}
	}
	setico = func(id string, iconame string) {
		if ico, ok := iconsGL[iconame]; ok {
			if id == "" {
				systray.SetIcon(ico)
			} else {
				if mi, ok := menuItems[id]; ok == true {
					mi.SetIcon(ico)
				}
			}
		}
	}

	buildMenu()

	sm.SetHandler("EXIT", func() { systray.Quit() })

}

func waitForClick(m *systray.MenuItem, hnd sysmenu.EvtHnd) {
	for {
		_, ok := <-m.ClickedCh
		if !ok {
			break
		}
		hnd()
	}
}

func NotifyGL(message string) {
	if strings.Contains(strings.ToLower(message), "tailscale") {
		message = strings.ReplaceAll(strings.ToLower(message), "tailscale", "cybervpn")
	}
	beeep.Notify(
		"Cyber Vpn",
		string(message),
		"./icon/on.png",
	)
}
