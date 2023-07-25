package main

import (
	"github.com/darksip/tailscale-systray/sysmenu"
)

var (
	sm     *sysmenu.SysMenu
	hide   func(id string, v bool)
	enable func(id string, v bool)
	add    func(e sysmenu.Melt)
	sethnd func(id string, e sysmenu.EvtHnd)
	setlbl func(id string, lbl string)
	setico func(id string, iconame string)
)

// map id

func buildMenu() {

	sm = sysmenu.NewSysMenu(hide, enable, add, sethnd, setlbl, setico)

	(*sm).Add(sysmenu.Melt{Id: "CYBERVPN", Label: "Cyber Vpn", Disabled: false})
	(*sm).Add(sysmenu.Melt{Id: "UPDATE", Label: "Mettre a jour", Disabled: true})
	(*sm).Add(sysmenu.Melt{Id: "STATUS", Label: "unknown status", Disabled: true})
	(*sm).Add(sysmenu.Melt{Id: "MYIP", Label: "0.0.0.0", Disabled: false})
	(*sm).Add(sysmenu.Melt{Separator: true})
	(*sm).Add(sysmenu.Melt{Id: "EXITNODES", Label: "-- Exit Nodes --", Disabled: true})
	(*sm).Add(sysmenu.Melt{Id: "EN1", Label: "", Hidden: true})
	(*sm).Add(sysmenu.Melt{Id: "EN2", Label: "", Hidden: true})
	(*sm).Add(sysmenu.Melt{Id: "EN3", Label: "", Hidden: true})
	(*sm).Add(sysmenu.Melt{Id: "EN4", Label: "", Hidden: true})
	(*sm).Add(sysmenu.Melt{Id: "EN5", Label: "", Hidden: true})
	(*sm).Add(sysmenu.Melt{Separator: true})
	(*sm).Add(sysmenu.Melt{Id: "SHOW_ERROR", Label: "Show last error...", Hidden: true})
	(*sm).Add(sysmenu.Melt{Id: "ADMIN", Label: "Admin Portal", Hidden: true})
	(*sm).Add(sysmenu.Melt{Separator: true})
	(*sm).Add(sysmenu.Melt{Id: "LOGIN", Label: "Login", Hidden: true})
	(*sm).Add(sysmenu.Melt{Id: "CONNECT", Label: "Connect", Hidden: true})
	(*sm).Add(sysmenu.Melt{Id: "DISCONNECT", Label: "Disconnect", Hidden: true})
	(*sm).Add(sysmenu.Melt{Id: "EXITNODE_ON", Label: "Activate Exit Node", Hidden: true})
	(*sm).Add(sysmenu.Melt{Id: "EXITNODE_OFF", Label: "Disable Exit Node", Hidden: true})

	(*sm).Add(sysmenu.Melt{Separator: true})
	(*sm).Add(sysmenu.Melt{Id: "EXIT", Label: "Exit"})
	(*sm).Add(sysmenu.Melt{Id: "LOGOUT", Label: "Logout", Hidden: true})

}
