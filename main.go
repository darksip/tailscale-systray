package main

//go:generate goversioninfo

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/atotto/clipboard"
	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"
	"github.com/joho/godotenv"

	"tailscale.com/client/tailscale"
)

var (
	//go:embed icon/on.png
	iconOnPng []byte
	//go:embed icon/off.png
	iconOffPng []byte
	//go:embed icon/on.ico
	iconOnIco []byte
	//go:embed icon/off.ico
	iconOffIco []byte
	iconOn     []byte
	iconOff    []byte
)

var (
	mu          sync.RWMutex
	myIP        string
	localClient tailscale.LocalClient
	//loadError    = false
	//needsLogin   = false
	errorMessage = ""
	exitNode     = ""
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

	iconOn = iconOnIco
	iconOff = iconOffIco
	if _, err := os.Stat(appdatapath); os.IsNotExist(err) {
		err := os.Mkdir(appdatapath, os.ModePerm)
		if err != nil {
			log.Println(err)
		}
	}
	errenv := godotenv.Load(path.Join(appdatapath, ".env"))
	if errenv != nil {
		log.Printf(".env file not found - create default values")
		f, ferr := os.Create(path.Join(appdatapath, ".env"))
		if ferr == nil {
			f.WriteString("CLIENTID=juvise\n")
			f.WriteString("BROWSER_METHOD=RUNDLL\n")
			//f.WriteString("ADMIN_MODE=off\n")
			f.Close()
		} else {
			log.Print(ferr.Error())
		}
	} else {
		val := os.Getenv("CLIENTID")
		if val != "" {
			rootUrl = fmt.Sprintf("https://head.%s.cyberfile.fr", val)
		} else {
			rootUrl = "https://head.cyberfile.fr"
		}
		val = os.Getenv("BROWSER_METHOD")
		if val != "" {
			browserMethod = val
		}
		val = os.Getenv("ADMIN_MODE")
		if val != "" {
			adminMode = val
		}
	}
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

func setExitNode() {
	refreshExitNode()
	if len(exitNode) > 0 {
		log.Printf("we have an exit node : %s", exitNode)
		// set exit and allow lan access local
		exitNodeParam := fmt.Sprintf("--exit-node=%s --exit-node-allow-lan-access", exitNode)
		_, errset := execCommand(cliExecutable, "set", exitNodeParam)
		if errset != nil {
			log.Printf(errset.Error())
		}
	}
}

func doLogin() {

	log.Printf("Do login by opening browser")
	Notify("Login process, \na browser window should open...")
	out, err := execCommand(cliExecutable, "login", "--login-server", rootUrl, "--accept-routes", "--unattended", "--timeout", "3s")
	// check Authurl
	if err != nil {
		urlLogin := strings.TrimSpace(parseForHttps(out))
		log.Printf("%s", string(urlLogin))
		if urlLogin != "" {
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
				if getBackenState() != "NeedsLogin" {
					Notify("Authentication complete")
					break
				}
			}
			// check for the needs of a needs of an exit node
			setExitNode()
			//log.Print(exitNodeParam)
		}
	} else {
		// ouvrir un dialog avec un lien cliquable
		Notify(err.Error())
	}
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

func refreshExitNode() {

	getStatus := localClient.Status
	status, err := getStatus(context.TODO())
	if err == nil {
		//log.Print("----------------------------------")
		for _, ps := range status.Peer {
			if len(ps.TailscaleIPs) != 0 {
				peerIP := ps.TailscaleIPs[1].String()
				//log.Printf("peer %s (%s): EN: %t ENOption: %t", ps.HostName, peerIP, ps.ExitNode, ps.ExitNodeOption)
				if ps.ExitNodeOption {
					exitNode = peerIP
					break
				}
			}
		}
	}
}

func onReady() {

	log.Printf("parsing args")

	flag.Parse()

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

	if st.BackendState == "NeedsLogin" {
		Notify("Cyber Vpn needs you to login...")
		doLogin()
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
				case "Stopped":
					mLogin.Hide()
					mLogout.Show()
					mConnect.Show()
					mDisconnect.Hide()
					mMyDevices.Hide()
					mThisDevice.Hide()
					systray.SetIcon(iconOff)
					systray.SetTooltip(appName + ": Stopped")
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

			for _, ps := range status.Peer {
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
