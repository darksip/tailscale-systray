package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

func getBackenState() string {
	st, _ := localClient.Status(context.TODO())
	return st.BackendState
}

func disconnectReconnect() {
	_, err := execCommand(cliExecutable, "down")
	if err != nil {
		Notify(err.Error())
	}
	time.Sleep(5 * time.Second)
	_, err = execCommand(cliExecutable, "up")
	if err != nil {
		Notify(err.Error())
	}
}

func doLogin() {

	log.Printf("Do login by opening browser")
	//Notify("Login process, \na browser window should open...")
	out, _ := execCommand(cliExecutable, "login", "--login-server", rootUrl, "--accept-routes", "--unattended", "--timeout", "3s")
	// check Authurl
	log.Println(string(out))
	var urlLogin = ""
	// get func to query status
	getStatus := localClient.Status
	var ntry = 0
	// wait for the link to be available or timeout
	for {
		status, errc := getStatus(context.TODO())
		if errc != nil {
			Notify(errc.Error())
			return
		}
		log.Printf("status: %s", status.BackendState)
		log.Printf("url: %s", status.AuthURL)
		if len(status.AuthURL) > 0 {
			urlLogin = status.AuthURL
			break
		}
		time.Sleep(1 * time.Second)
		ntry++
		if ntry > 120 {
			Notify("Login Timeout")
			return
		}
	}
	// open the browser
	errb := openBrowser(urlLogin)
	if errb != nil {
		Notify(errb.Error())
	} else {
		Notify("I'm opening your browser for identification\nYour authentication may be automatic\n or you may be asked for credentials")
	}
	// wait for status change
	for {
		time.Sleep(2 * time.Second)
		//st, _ := localClient.Status(context.TODO())
		if getBackenState() == "Running" {
			Notify("Authentication complete")
			break
		}
	}

	// check for the needs of a needs of an exit node
	setExitNode()
	//log.Print(exitNodeParam)
}

func doConnection(verb string) {
	bsBefore := getBackenState()
	log.Printf("state before : %s", bsBefore)
	//log.Printf("launch command: tailscale %s", verb)
	_, err := execCommand(cliExecutable, verb)
	if err != nil {
		Notify(err.Error())
	}
	bsAfter := getBackenState()
	log.Printf("state after : %s", bsAfter)
	if bsBefore != bsAfter {
		if bsAfter == "Running" {
			setExitNode()
			Notify("Cyber Vpn is active with exit node")
		} else {
			// TODO: faire plutot un switch avec default
			if strings.ToLower(bsAfter) == "needslogin" {
				Notify(fmt.Sprintf("Cyber Vpn needs login ,\n click on systray icon to log"))
			}
			if strings.ToLower(bsAfter) == "stopped" {
				Notify(fmt.Sprintf("Cyber Vpn is disconnected"))
			}
			if strings.ToLower(bsAfter) == "logedout" {
				Notify(fmt.Sprintf("Cyber Vpn is loged out \nClick on Login when you want to activate"))
			}
		}

	}
}

func AddConnectionHandlersToMenu() {
	sm.SetHandler("LOGIN", func() {
		sm.SetDisabled("LOGIN", true)
		doLogin()
		sm.SetDisabled("LOGIN", false)
	})
	sm.SetHandler("LOGOUT", func() { doConnection("logout") })
	sm.SetHandler("CONNECT", func() { doConnection("up") })
	sm.SetHandler("DISCONNECT", func() { doConnection("down") })
}
