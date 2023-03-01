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
		Notify(err.Error(), "error")
	}
	time.Sleep(5 * time.Second)
	_, err = execCommand(cliExecutable, "up")
	if err != nil {
		Notify(err.Error(), "error")
	}
}

func doLogin() {

	if authKey != "" {
		out, _ := execCommand(cliExecutable, "login", "--login-server", rootUrl, "--authkey", authKey, "--accept-routes", "--unattended", "--timeout", "3s")
		log.Println(string(out))
	} else {
		log.Printf("Do login by opening browser")
		//Notify("Login process, \na browser window should open...")
		out, _ := execCommand(cliExecutable, "login", "--login-server", rootUrl, "--accept-routes", "--unattended", "--timeout", "3s")
		// check Authurl
		log.Println(string(out))
	}

	var urlLogin = ""
	// get func to query status
	getStatus := localClient.Status
	var ntry = 0
	// wait for the link to be available or timeout
	for {
		status, errc := getStatus(context.TODO())
		if errc != nil {
			Notify(errc.Error(), "error")
			return
		}
		log.Printf("status: %s", status.BackendState)
		if status.BackendState == "Running" || status.BackendState == "Starting" {
			Notify("Autentication Complete", "connected")
			return
		}
		log.Printf("url: %s", status.AuthURL)
		if len(status.AuthURL) > 0 {
			urlLogin = status.AuthURL
			break
		}
		time.Sleep(1 * time.Second)
		ntry++
		if ntry > 120 {
			Notify("Login Timeout", "error")
			return
		}
	}

	Notify("I'm opening your browser for identification\nYour authentication may be automatic\n or you may be asked for credentials", "browser")

	// open the browser
	errb := openBrowser(urlLogin)
	if errb != nil {
		Notify(errb.Error(), "error")
	}
	// wait for status change
	for {
		time.Sleep(2 * time.Second)
		//st, _ := localClient.Status(context.TODO())
		if getBackenState() == "Running" {
			Notify("Authentication complete", "connected")
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
		Notify(err.Error(), "error")
	}
	bsAfter := getBackenState()
	log.Printf("state after : %s", bsAfter)
	if bsBefore != bsAfter {
		if bsAfter == "Running" {
			setExitNode()
			Notify("You connection is active with exit node", "connected")
		} else {
			// TODO: faire plutot un switch avec default
			if strings.ToLower(bsAfter) == "needslogin" {
				Notify(fmt.Sprintf("Cyber Vpn needs you to authenticate ,\n click on systray icon to Log in"), "needslogin")
			}
			if strings.ToLower(bsAfter) == "stopped" {
				Notify(fmt.Sprintf("Cyber Vpn is disconnected\nRight Ckick on systray icon\n and choose Connect"), "disconnected")
			}
			if strings.ToLower(bsAfter) == "logged out" {
				Notify(fmt.Sprintf("Cyber Vpn is logged out \nClick on Login when you want to activate"), "unknown")
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
