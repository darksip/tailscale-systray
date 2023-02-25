package main

import (
	"log"
	"strings"

	"github.com/darksip/tailscale-systray/sysmenu"
	"github.com/lxn/walk"
)

type Tray struct {
	*walk.NotifyIcon
	// Current known tunnels by name
	exitNodes map[string]*walk.Action
	actions   map[string]*walk.Action

	clicked func()
}

var tray *Tray

func NewTray(mtw *walk.MainWindow) (*Tray, error) {
	var err error

	tray := &Tray{
		exitNodes: make(map[string]*walk.Action),
		actions:   make(map[string]*walk.Action),
	}

	tray.NotifyIcon, err = walk.NewNotifyIcon(mtw)
	if err != nil {
		return nil, err
	}

	return tray, tray.setup()
}

func (tray *Tray) setup() error {

	add = func(e sysmenu.Melt) {
		var action *walk.Action
		if e.Separator {
			action = walk.NewSeparatorAction()
		} else {
			action = walk.NewAction()
			action.SetText(e.Label)
			action.SetEnabled(!e.Disabled)
			action.SetVisible(!e.Hidden)
			action.SetDefault(e.Defawlt)
			tray.actions[e.Id] = action
		}
		tray.ContextMenu().Actions().Add(action)
	}
	enable = func(id string, v bool) {
		if ta, ok := tray.actions[id]; ok == true {
			ta.SetEnabled(v)
		}
	}
	hide = func(id string, v bool) {
		if ta, ok := tray.actions[id]; ok == true {
			ta.SetVisible(!v)
		}
	}
	sethnd = func(id string, e sysmenu.EvtHnd) {
		if ta, ok := tray.actions[id]; ok == true {
			ta.Triggered().Attach(walk.EventHandler(e))
		}
	}
	setlbl = func(id, lbl string) {
		if ta, ok := tray.actions[id]; ok == true {
			ta.SetText(lbl)
		}
	}
	setico = func(id string, iconame string) {
		if id == "" {
			if icon, ok := iconsWalk[iconame]; ok {
				tray.SetIcon(icon)
			}
		}
		if ta, ok := tray.actions[id]; ok == true {
			if icon, ok := iconsWalk[iconame]; ok {
				ta.SetImage(icon)
			} else {
				log.Printf("pb icone : %s  ok=%t", iconame, ok)
			}

		}
	}
	//icon, err := walk.Resources.Icon("icon/bluebaloon.ico")
	icon, _ := iconsWalk["on"]
	tray.SetIcon(icon)
	tray.SetVisible(true)

	buildMenu()

	sm.SetHandler("EXIT", func() {
		walk.App().Exit(0)
	})
	return nil
}

func NotifyWalk(message string, iconame string) {
	if strings.Contains(strings.ToLower(message), "tailscale") {
		message = strings.ReplaceAll(strings.ToLower(message), "tailscale", "cybervpn")
	}
	if icon, ok := iconsWalk[iconame]; ok {
		tray.ShowCustom(
			"Cyber Vpn",
			message,
			icon)

	}

}

func RunWalk() {

	initIconsWalk()

	mw, err := walk.NewMainWindow()
	if err != nil {
		log.Fatal(err)
	}
	//	icon, err := walk.Resources.Icon("icon/on.ico")
	tray, err = NewTray(mw)

	onMenuReady()

	mw.Run()
}
