package main

//go:generate goversioninfo

import (
	"context"

	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/atotto/clipboard"

	"tailscale.com/client/tailscale"
)

// il faudrait faire une struct pour refleter l etat de la struct dans l interface

var (
	mu           sync.RWMutex
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
	// rune getlantern systray
	//RunGl()
}

func Notify(message string) {
	NotifyWalk(message)
}

func onMenuReady() {

	log.Printf("getting localClient...")
	getStatus := localClient.Status
	st, _ := getStatus(context.TODO())

	// compose complete menu with hidden options
	AddConnectionHandlersToMenu()
	AddExitNodeHandlersToMenu()

	sm.SetHandler("ADMIN", func() {
		err := openBrowser(adminUrl)
		if err != nil {
			Notify(err.Error())
		}
	})
	sm.SetHandler("SHOW_ERROR", func() {
		Notify(errorMessage)
	})
	sm.SetHandler("MYIP", func() {
		err := clipboard.WriteAll(myIP)
		if err == nil {
			Notify(fmt.Sprintf("Copy the IP address (%s) to the Clipboard", myIP))
		}
	})

	sm.SetIcon("", "off")

	if st != nil {
		if st.BackendState == "NeedsLogin" {
			Notify("Cyber Vpn needs you to login...")
			doLogin()
		}
	} else {
		// the service should have started prior
		// we need to wait and try periodically until the
		// service is responding
		log.Println("The service CyberVpn does not respond")
	}

	go func() {
		for {
			time.Sleep(3 * time.Second)

			status, err := getStatus(context.TODO())

			if err != nil {
				errorMessage = err.Error()
				log.Printf("%s", errorMessage)
				sm.SetHiddenAll([]string{"CONNECT", "LOGIN", "DISCONNECT", "EXITNODE_ON", "EXINODE_OFF", "LOGOUT"}, true)
				sm.SetHiddenAll([]string{"EXITNODES", "EN1", "EN2", "EN3", "EN4", "EN5"}, true)

				sm.SetHidden("SHOW_ERROR", false)
				sm.SetIcon("", "off")
				continue
			} else {
				errorMessage = ""
				sm.SetHidden("SHOW_ERROR", true)
				sm.SetLabel("STATUS", status.BackendState)
				sm.SetIcon("CYBERVPN", "off16")
				switch status.BackendState {
				case "NeedsLogin":
					sm.SetHiddenAll([]string{"CONNECT", "DISCONNECT", "EXITNODE_ON", "EXINODE_OFF", "LOGOUT"}, true)
					sm.SetHiddenAll([]string{"EXITNODES", "EN1", "EN2", "EN3", "EN4", "EN5"}, true)
					sm.SetHidden("LOGIN", false)
					sm.SetIcon("", "off")
					sm.SetIcon("MYIP", "redballoon")
					continue
				case "Stopped":
					sm.SetHiddenAll([]string{"DISCONNECT", "EXITNODE_ON", "EXINODE_OFF", "LOGIN"}, true)
					sm.SetHiddenAll([]string{"EXITNODES", "EN1", "EN2", "EN3", "EN4", "EN5"}, true)
					sm.SetHiddenAll([]string{"LOGOUT", "CONNECT"}, false)
					sm.SetIcon("", "off")
					sm.SetIcon("MYIP", "greyballoon")
					continue
				case "Running", "Starting":
					sm.SetHiddenAll([]string{"CONNECT", "EXITNODE_ON", "EXINODE_OFF", "LOGIN"}, true)
					if status.ExitNodeStatus != nil {
						sm.SetHidden("EXITNODES", false)
						sm.SetHidden("EXITNODE_OFF", false)
						sm.SetDisabled("EXITNODE_OFF", false)
					} else {
						sm.SetHiddenAll([]string{"EXITNODES", "EN1", "EN2", "EN3", "EN4", "EN5"}, true)
						sm.SetHidden("EXITNODE_ON", false)
						sm.SetHidden("EXITNODE_OFF", true)
					}

					sm.SetHiddenAll([]string{"LOGOUT", "DISCONNECT"}, false)
					sm.SetIcon("", "on")
					sm.SetIcon("MYIP", "blueballoon")
				}
			}

			mu.Lock()

			if len(status.TailscaleIPs) != 0 {
				myIP = status.TailscaleIPs[1].String()
				log.Printf("my ip: %s", myIP)
				sm.SetLabel("MYIP", myIP)
			}
			if wantsToDisableExitNodes {
				setExitNodeOff()
				mu.Unlock()
				continue
			}
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
			mu.Unlock()

			// cette section sera transfer dans la gestion d unr
			// liste dans une fenetre a part
			// -> contenu dans loopInNodes.txt
		}
	}()
}
