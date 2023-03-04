package main

//go:generate goversioninfo

import (
	"context"
	"strings"

	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/atotto/clipboard"

	"tailscale.com/client/tailscale"
	"tailscale.com/ipn/ipnstate"
)

// il faudrait faire une struct pour refleter l etat de la struct dans l interface

var (
	//mu           sync.RWMutex
	myIP         string
	localClient  tailscale.LocalClient
	errorMessage = ""
)

// tailscale local client to use for IPN

func exitIfAlreadyRunnning() {
	addr := "localhost:25169"
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Print("Program is already running.")
		os.Exit(1)
	}
	defer l.Close()
}

func main() {

	exitIfAlreadyRunnning()
	// load environement parameters from %programdata%\.env
	loadEnv()

	latencies = make(map[string][]float64)
	movLatencies = map[string]float64{}
	nping = 0

	iconOn = iconOnIco
	iconOff = iconOffIco
	RunWalk()
	// run getlantern systray
	//RunGl()
}

func Notify(message string, iconame string) {
	NotifyWalk(message, iconame)
}

func addMenuHandlers() {
	// compose complete menu with hidden options
	AddConnectionHandlersToMenu()

	if noExitNode == 0 {
		AddExitNodeHandlersToMenu()
	}

	sm.SetHandler("ADMIN", func() {
		err := openBrowser(adminUrl)
		if err != nil {
			Notify(err.Error(), "error")
		}
	})
	sm.SetHandler("SHOW_ERROR", func() {
		Notify(errorMessage, "error")
	})

	sm.SetHandler("MYIP", func() {
		err := clipboard.WriteAll(myIP)
		if err == nil {
			Notify(fmt.Sprintf("Copy the IP address (%s) to the Clipboard", myIP), "info")
		}
	})
}

func setMenuState(status *ipnstate.Status) (exit bool) {
	switch status.BackendState {
	case "NeedsLogin", "NoState":
		sm.SetHiddenAll([]string{"CONNECT", "DISCONNECT", "EXITNODE_ON", "EXITNODE_OFF", "LOGOUT"}, true)
		sm.SetHiddenAll([]string{"EXITNODES", "EN1", "EN2", "EN3", "EN4", "EN5"}, true)
		sm.SetHidden("LOGIN", false)
		sm.SetIcon("", "off")
		sm.SetIcon("MYIP", "redballoon")
		return true
	case "Stopped":
		sm.SetHiddenAll([]string{"DISCONNECT", "EXITNODE_ON", "EXITNODE_OFF", "LOGIN"}, true)
		sm.SetHiddenAll([]string{"EXITNODES", "EN1", "EN2", "EN3", "EN4", "EN5"}, true)
		sm.SetHiddenAll([]string{"LOGOUT", "CONNECT"}, false)
		sm.SetIcon("", "off")
		sm.SetIcon("MYIP", "greyballoon")
		return true
	case "Running", "Starting":
		sm.SetHiddenAll([]string{"CONNECT", "EXITNODE_ON", "EXITNODE_OFF", "LOGIN"}, true)
		if status.ExitNodeStatus != nil {
			sm.SetHidden("EXITNODES", false)
			sm.SetHidden("EXITNODE_OFF", false)
			//sm.SetDisabled("EXITNODE_OFF", false)
		} else {
			sm.SetHiddenAll([]string{"EXITNODES", "EN1", "EN2", "EN3", "EN4", "EN5"}, true)
			sm.SetHidden("EXITNODE_ON", false)
			sm.SetHidden("EXITNODE_OFF", true)
			//sm.SetDisabled("EXITNODE_OFF", false)
		}
		sm.SetHiddenAll([]string{"LOGOUT", "DISCONNECT"}, false)
		sm.SetIcon("", "on")
		sm.SetIcon("MYIP", "blueballoon")
	}
	return false
}

func onMenuReady() {

	log.Printf("getting localClient...")
	getStatus := localClient.Status
	st, _ := getStatus(context.TODO())

	// add handlers to menu items
	addMenuHandlers()
	// set default icon to gray logo
	sm.SetIcon("", "off")

	if st != nil {
		if st.BackendState == "NeedsLogin" {
			Notify("Cyber Vpn needs you to login...\nPlease wait while trying to reach the server...", "needslogin")
			doLogin()
		}
		if strings.ToLower(st.BackendState) == "stopped" {
			Notify(fmt.Sprintf("Cyber Vpn is disconnected\nRight Ckick on systray icon\n and choose Connect"), "disconnected")
		}
	} else {
		log.Println("The service CyberVpn does not respond")
	}

	// base deamon looping forever
	go func() {
		for {
			time.Sleep(3 * time.Second)

			status, err := getStatus(context.TODO())
			// update sytray menu regarding the Backend State
			if err != nil {
				errorMessage = err.Error()
				log.Printf("%s", errorMessage)
				sm.SetHiddenAll([]string{"CONNECT", "LOGIN", "DISCONNECT", "EXITNODE_ON", "EXINODE_OFF", "LOGOUT"}, true)
				sm.SetHiddenAll([]string{"EXITNODES", "EN1", "EN2", "EN3", "EN4", "EN5"}, true)

				sm.SetHidden("SHOW_ERROR", false)
				sm.SetIcon("", "off")
				continue
			} else {
				log.Printf("State: %s", status.BackendState)
				errorMessage = ""

				sm.SetHidden("SHOW_ERROR", true)
				sm.SetLabel("STATUS", status.BackendState)
				sm.SetIcon("CYBERVPN", "off16")
				if setMenuState(status) {
					if noExitNode > 0 {
						sm.SetHiddenAll([]string{"EXITNODE_ON", "EXITNODE_OFF", "EXITNODES", "EN1", "EN2", "EN3", "EN4", "EN5"}, true)
					}
					// if the stats is not Running don't do exit Nodes Check
					continue
				}
			}

			if noExitNode > 0 {
				sm.SetHiddenAll([]string{"EXITNODE_ON", "EXITNODE_OFF", "EXITNODES", "EN1", "EN2", "EN3", "EN4", "EN5"}, true)
			}
			//mu.Lock()

			if len(status.TailscaleIPs) != 0 {
				myIP = status.TailscaleIPs[1].String()
				log.Printf("my ip: %s", myIP)
				sm.SetLabel("MYIP", myIP)
			}
			if wantsToDisableExitNodes {
				log.Println("wants exit nodes to be disabled...")
				setExitNodeOff()
				//mu.Unlock()
				// do not check the best exit node if disabled wanted
				continue
			}
			if noExitNode == 0 {
				refreshExitNodes()
				bestIp := checkLatency()
				showOrderedExitNode(bestIp)
				if status.ExitNodeStatus != nil {
					if len(status.ExitNodeStatus.TailscaleIPs) > 1 {
						activeExitNode = status.ExitNodeStatus.TailscaleIPs[1].Addr().String()
						checkActiveNodeAndSetExitNode()
					}
				} else {
					setExitNode()
				}
			}

			//mu.Unlock()
			// gestion des Peers dans une fenetre separée pour ne faire
			// l'interrogation qu'à l'ouverture de la fenêtre
		}
	}()
}
