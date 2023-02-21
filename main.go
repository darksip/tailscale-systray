package main

//go:generate goversioninfo

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/atotto/clipboard"
	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"

	"tailscale.com/client/tailscale"
)

var (
	//go:embed icon/on.png
	iconOnPng []byte
	//go:embed icon/off.png
	iconOffPng []byte
	//go:embed icon/on256.ico
	iconOnIco []byte
	//go:embed icon/off256.ico
	iconOffIco []byte
	iconOn     []byte
	iconOff    []byte
)

// il faudrait faire une struct pour refleter l etat de la struct dans l interface

var (
	mu          sync.RWMutex
	myIP        string
	localClient tailscale.LocalClient
	//loadError    = false
	//needsLogin   = false
	errorMessage = ""
	menuExitNode *systray.MenuItem
	//exitNodePing   = 0.0
)

// set login-url as a variable in registry

// implement the OIDC scenario to

// add an entry to specify a preshared key

// tailscale local client to use for IPN

func main() {

	addr := "localhost:25169"
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Print("Program is already running.")
		os.Exit(1)
	}
	defer l.Close()
	// your program logic here
	loadEnv()

	latencies = make(map[string][]float64)
	movLatencies = map[string]float64{}
	nping = 0

	iconOn = iconOnIco
	iconOff = iconOffIco

	systray.Run(onReady, nil)
}

func Notify(message string) {
	if strings.Contains(strings.ToLower(message), "tailscale") {
		message = strings.ReplaceAll(strings.ToLower(message), "tailscale", "cybervpn")
	}
	beeep.Notify(
		"Cyber Vpn",
		string(message),
		"./icon/on.png",
	)
}

// change the function to pass mandatory parameters with login-url
func doConnectionControl(m *systray.MenuItem, verb string) {
	for {
		if _, ok := <-m.ClickedCh; !ok {
			break
		}
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
		// TODO: loop with timeout for changing state
	}
}

func exitSystray(m *systray.MenuItem) {
	<-m.ClickedCh
	systray.Quit()
}

func parseForHttps(out []byte) string {
	lines := strings.Split(string(out), "\n")
	for _, l := range lines {
		if strings.Contains(l, "https") {
			return l
		}
	}
	return ""
}

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

func setExitNodeOff() {
	for {
		if _, ok := <-menuExitNode.ClickedCh; !ok {
			break
		}
		if len(activeExitNode) > 0 {
			o, errset := execCommand(cliExecutable, "set", `--exit-node=`)
			if errset != nil {
				log.Printf("%s", o)
				log.Printf(errset.Error())
			}
			activeExitNode = ""
			menuExitNode.SetTitle("Set Exit Node On")
		} else {
			setExitNode()
		}
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

func waitForLogin(m *systray.MenuItem) {
	for {
		<-m.ClickedCh
		m.Disable()
		doLogin()
		// enable menu Login (it will be hidden by another routine)
		m.Enable()
	}
}

func waitForClickAndNotify(m *systray.MenuItem) {
	for {
		<-m.ClickedCh
		beeep.Notify(
			appName,
			errorMessage,
			"",
		)
	}
}

func waitForClickAndOpenBrowser(m *systray.MenuItem, url string) {
	for {
		_, ok := <-m.ClickedCh
		if !ok {
			break
		}
		err := openBrowser(url)
		if err != nil {
			Notify(err.Error())
		}
	}
}

func waitForClickAndCopyIpToClipboard(m *systray.MenuItem) {
	for {
		_, ok := <-m.ClickedCh
		if !ok {
			break
		}
		mu.RLock()
		if myIP == "" {
			mu.RUnlock()
			continue
		}
		err := clipboard.WriteAll(myIP)
		if err == nil {
			beeep.Notify(
				"This device",
				fmt.Sprintf("Copy the IP address (%s) to the Clipboard", myIP),
				"",
			)
		}
		mu.RUnlock()
	}
}

func onReady() {

	log.Printf("getting localClient...")
	getStatus := localClient.Status

	st, _ := getStatus(context.TODO())
	//bs, _ := json.Marshal(st)
	//fmt.Println(string(bs))

	systray.SetIcon(iconOff)

	// compose complete menu with hidden options

	mError := systray.AddMenuItem("Show Error", "")
	mError.Hide()
	go waitForClickAndNotify(mError)

	mLogin := systray.AddMenuItem("Login...", "")
	mLogin.Hide()
	go waitForLogin(mLogin)

	systray.AddSeparator()
	mConnect := systray.AddMenuItem("Connect", "")
	mConnect.Hide()
	go doConnectionControl(mConnect, "up")

	mDisconnect := systray.AddMenuItem("Disconnect", "")
	mDisconnect.Hide()
	go doConnectionControl(mDisconnect, "down")

	menuExitNode = systray.AddMenuItem("Exit Node Off", "")
	go setExitNodeOff()

	systray.AddSeparator()
	mThisDevice := systray.AddMenuItem("This device:", "")
	mThisDevice.Hide()
	go waitForClickAndCopyIpToClipboard(mThisDevice)

	systray.AddSeparator()
	mNetworkDevices := systray.AddMenuItem("Network Devices", "")
	mNetworkDevices.Show()
	mMyDevices := mNetworkDevices.AddSubMenuItem("My Devices", "")
	//mTailscaleServices := mNetworkDevices.AddSubMenuItem("Tailscale Services", "")

	systray.AddSeparator()
	mAdminConsole := systray.AddMenuItem("Admin Console...", "")
	if adminMode != "on" {
		mAdminConsole.Disable()
	}
	go waitForClickAndOpenBrowser(mAdminConsole, adminUrl)

	systray.AddSeparator()
	mExit := systray.AddMenuItem("Exit", "")
	go exitSystray(mExit)

	systray.AddSeparator()
	mLogout := systray.AddMenuItem("Logout...", "")
	mLogout.Hide()
	go doConnectionControl(mLogout, "logout")

	systray.AddSeparator()

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
		type Item struct {
			menu  *systray.MenuItem
			title string
			ip    string
			found bool
		}
		items := map[string]*Item{}

		for {
			time.Sleep(3 * time.Second)
			status, err := getStatus(context.TODO())
			//log.Printf("%s", status.Self.HostName)
			// find exit nodes in peers

			if err != nil {
				//loadError = true
				errorMessage = err.Error()
				log.Printf("%s", errorMessage)
				systray.SetTooltip(appName + ": Api Error")
				mConnect.Hide()
				mDisconnect.Hide()
				mError.Show()
				mLogin.Hide()
				mLogout.Hide()
				mThisDevice.Hide()
				mMyDevices.Hide()
				systray.SetIcon(iconOff)
				continue
			} else {
				//loadError = false
				errorMessage = ""
				mError.Hide()
				switch status.BackendState {
				case "NeedsLogin":
					mLogin.Show()
					mLogout.Hide()
					mThisDevice.Hide()
					mConnect.Hide()
					mDisconnect.Hide()
					mMyDevices.Hide()
					systray.SetTooltip(appName + ": Needs Login")
					systray.SetIcon(iconOff)
					continue
				case "Stopped":
					mLogin.Hide()
					mLogout.Show()
					mConnect.Show()
					mDisconnect.Hide()
					mMyDevices.Hide()
					mThisDevice.Hide()
					systray.SetIcon(iconOff)
					systray.SetTooltip(appName + ": Stopped")
					continue
				case "Running", "Starting":
					mLogin.Hide()
					mLogout.Show()
					mConnect.Hide()
					mDisconnect.Show()
					mMyDevices.Show()
					mThisDevice.Show()
					systray.SetIcon(iconOn)
					systray.SetTooltip(appName + ": " + status.BackendState)
				}
			}

			mu.Lock()

			if len(status.TailscaleIPs) != 0 {
				myIP = status.TailscaleIPs[1].String()
				log.Printf("my ip: %s", myIP)
			}
			mu.Unlock()

			for _, v := range items {
				v.found = false
			}

			mThisDevice.SetTitle(fmt.Sprintf("This device: %s (%s)", status.Self.HostName, myIP))

			mu.Lock()

			refreshExitNodes()
			checkLatency()
			bestExitNode := getBestExitNodeIp()
			log.Printf("best exit node : %s", bestExitNode)
			if status.ExitNodeStatus != nil {
				activeExitNode = status.ExitNodeStatus.TailscaleIPs[1].Addr().String()
				log.Printf("active exit node : %s", activeExitNode)
				if !isStillActive(activeExitNode) {
					log.Printf("ouch! activeExitNode is unreachable ! let' choose another one")
					//disconnectReconnect()
					setExitNode()
				} else {
					if nping > 10 {
						// TODO : demand at least 30% best in latency to change
						if bestExitNode != activeExitNode {
							setExitNode()
						}
					}
				}
			} else {
				setExitNode()
			}
			mu.Unlock()

			for _, ps := range status.Peer {
				if ips := ps.TailscaleIPs; ips == nil || len(ips) < 2 {
					continue
				}
				ip := ps.TailscaleIPs[1].String()
				peerName := ps.DNSName
				title := peerName

				sub := mMyDevices

				if item, ok := items[title]; ok {
					item.found = true
				} else {
					items[title] = &Item{
						menu:  sub.AddSubMenuItem(title, title),
						title: title,
						ip:    ip,
						found: true,
					}
					go func(item *Item) {
						// TODO fix race condition
						for {
							_, ok := <-item.menu.ClickedCh
							if !ok {
								break
							}
							err := clipboard.WriteAll(item.ip)
							if err != nil {
								beeep.Notify(
									appName,
									err.Error(),
									"",
								)
								return
							}
							beeep.Notify(
								item.title,
								fmt.Sprintf("Copy the IP address (%s) to the Clipboard", item.ip),
								"",
							)
						}
					}(items[title])
				}
			}

			for k, v := range items {
				if !v.found {
					// TODO fix race condition
					v.menu.Hide()
					delete(items, k)
				}
			}

		}
	}()
}
