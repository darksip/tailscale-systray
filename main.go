package main

//go:generate goversioninfo

import (
	"context"
	"strings"
	"sync"

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
	mu           sync.RWMutex
	myIP         string
	localClient  tailscale.LocalClient
	errorMessage = ""
	myVersion    = "1.20.4"
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
	if IsWindowsServer() {
		log.Printf("Execution sur une plateforme serveur\non utilise la presharedkey")
	}
	// load environement parameters from %programdata%\.env
	loadEnv(false)

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
		sm.SetDisabled("LOGIN", false)
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
		sm.SetDisabled("LOGIN", true)
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

// fonction appelee des que le menu est pret
func onMenuReady() {

	log.Printf("getting localClient...")
	getStatus := localClient.Status
	st, _ := getStatus(context.TODO())

	// add handlers to menu items
	addMenuHandlers()
	// set default icon to gray logo
	sm.SetIcon("", "off")

	if st != nil {
		if st.BackendState == "NeedsLogin" || st.BackendState == "NoState" {
			Notify("Cyber Vpn needs you to login...\nPlease wait while trying to reach the server...", "needslogin")
			sm.SetDisabled("LOGIN", true)
			go doLogin()
		}
		if strings.ToLower(st.BackendState) == "stopped" {
			Notify(fmt.Sprintf("Cyber Vpn is disconnected\nRight Ckick on systray icon\n and choose Connect"), "disconnected")
			sm.SetDisabled("LOGIN", false)
		}
	} else {
		log.Println("The service CyberVpn does not respond")
	}

	// launch monitor and auto-update loop
	go func() {
		lastUpdateNotification := time.Now().AddDate(0, 0, -2)
		launchMsi := ""

		sm.SetDisabled("UPDATE", false)
		sm.SetHidden("UPDATE", true)
		sm.SetIcon("UPDATE", "caution")
		for {
			log.Printf("local client version: %s", myVersion)
			// call monitoring function to report status

			// if not already waiting for install check for newer version
			status, newVersionPath, err := checkAndDownload()
			if err == nil {
				log.Printf("status: %s", status)
				// si up to date -> on passe
				if status == "up to date" {
					sm.SetHidden("UPDATE", false)
					continue
				} else if status == "successful download" {
					Notify(fmt.Sprintf("Une  mise à jour du logciel est disponible.\nCliquez droit sur l'icone du systray et choisissez\nMise a jour"), "caution")
					launchMsi = newVersionPath
					lastUpdateNotification = time.Now()
					sm.SetHidden("UPDATE", false)
					sm.SetHandler("UPDATE", func() {
						log.Printf("launch %s ...", launchMsi)

						_, err := execCommand("msiexec", "/i", launchMsi)
						if err == nil {
							os.Exit(0)
						} else {
							errorMessage = err.Error()
							sm.SetHidden("SHOW_ERROR", false)
							sm.SetDisabled("SHOW_ERROR", false)
						}
					})
				} else if status == "already downloaded" {
					if time.Since(lastUpdateNotification).Hours() >= 24 {
						Notify(fmt.Sprintf("Une  mise à jour du logciel est disponible.\nCliquez droit sur l'icone du systray et choisissez\nMise a jour"), "caution")
						lastUpdateNotification = time.Now()
						launchMsi = newVersionPath
						sm.SetHandler("UPDATE", func() {
							log.Printf("launch %s ...", launchMsi)
							_, err := execCommand("msiexec", "/i", launchMsi)
							if err == nil {
								os.Exit(0)
							} else {
								errorMessage = err.Error()
								sm.SetHidden("SHOW_ERROR", false)
								sm.SetDisabled("SHOW_ERROR", false)
							}
						})
					}
					sm.SetHidden("UPDATE", false)

				} else {
					log.Printf("status innatendu: %s", status)
				}
			}
			// else check if we have to notify or add to menu
			time.Sleep(15 * time.Second)
		}
	}()
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
					// if the status is NeedsLogin or NoState and manualLogout==0
					// probably neeeds login after token expiration in sleep mode
					if status.BackendState == "NeedsLogin" || status.BackendState == "NoState" {
						if !loginIsProcessing && manualLogout == 0 {
							// user did not asked for loggout an no login is already processing
							log.Printf("got to log in, token expired...")
							go func() {
								sm.SetDisabled("LOGIN", true)
								doLogin()
								sm.SetDisabled("LOGIN", false)
							}()
						} else {
							log.Printf("don't have to log in : loginIsProcessing[%t] manualLogout[%d]", loginIsProcessing, manualLogout)
						}
					}
					// if the state is not Running don't do exitNodes Check
					continue
				}
			}

			if noExitNode > 0 {
				sm.SetHiddenAll([]string{"EXITNODE_ON", "EXITNODE_OFF", "EXITNODES", "EN1", "EN2", "EN3", "EN4", "EN5"}, true)
			}
			mu.Lock()

			if len(status.TailscaleIPs) != 0 {
				myIP = status.TailscaleIPs[1].String()
				log.Printf("my ip: %s", myIP)
				sm.SetLabel("MYIP", myIP)
			}
			if wantsToDisableExitNodes || (noExitNode > 0) {
				log.Println("wants exit nodes to be disabled...")
				setExitNodeOff()
				mu.Unlock()
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

			mu.Unlock()
			// gestion des Peers dans une fenetre separée pour ne faire
			// l'interrogation qu'à l'ouverture de la fenêtre
		}
	}()
}
