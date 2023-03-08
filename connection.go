package main

import (
	"context"
	"log"
	"time"
)

var (
	loginIsProcessing = false
)

func getBackenState() string {
	st, _ := localClient.Status(context.TODO())
	return st.BackendState
}

func disconnectReconnect() {
	doDisconnect()
	time.Sleep(5 * time.Second)
	doConnect()
}

func success(url string) {

	log.Printf("url: %s", url)
	Notify("I'm opening your browser for identification\nYour authentication may be automatic\n or you may be asked for credentials", "browser")

	// open the browser
	errb := openBrowser(url)
	if errb != nil {
		Notify(errb.Error(), "error")
	}
}

func failure(err error) {
	log.Printf("%s", err.Error())
}

// TODO implementer un fct asynchrone pur que le menu
// continue de fctner
func doLogin() {

	manualLogout = 0
	loginIsProcessing = true
	forceReauth := false

	prefs, err := localClient.GetPrefs(context.TODO())
	if err != nil {
		log.Printf("%s", err.Error())
	}
	// start from old prefs and set the new value
	prefs.ForceDaemon = true
	if prefs.ControlURL != rootUrl {
		prefs.ControlURL = rootUrl
		forceReauth = true
	}

	errlogin := runUp(context.TODO(), "login", prefs, forceReauth, authKey,
		0, success, failure)
	if errlogin != nil {
		Notify(err.Error(), "error")
	}
	loginIsProcessing = false
	// check for the needs of a needs of an exit node
	setExitNode()
	//log.Print(exitNodeParam)
}

func doLogout() {
	localClient.Logout(context.TODO())
}

func doDisconnect() {
	err := runDown(context.TODO())
	if err != nil {
		log.Printf("%s", err.Error())
	}
}

func doConnect() {
	pref, err := localClient.GetPrefs(context.TODO())
	if err != nil {
		log.Printf("%s", err.Error())
	} else {
		// on change les prefs
		runUp(context.TODO(), "up", pref, false, "",
			0, success, failure)
	}
}

func doDeactivateExitNode() {

}

func AddConnectionHandlersToMenu() {
	sm.SetHandler("LOGIN", func() {
		sm.SetDisabled("LOGIN", true)
		go func() {
			doLogin()
			sm.SetDisabled("LOGIN", false)
		}()
	})
	sm.SetHandler("LOGOUT", func() {
		manualLogout = 1
		doLogout()
		sm.SetDisabled("LOGIN", false)
	})
	sm.SetHandler("CONNECT", func() { doConnect() })
	sm.SetHandler("DISCONNECT", func() { doDisconnect() })
}
