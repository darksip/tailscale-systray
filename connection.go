package main

import (
	"context"
	"log"
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
