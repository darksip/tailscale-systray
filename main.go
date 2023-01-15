package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/atotto/clipboard"
	"github.com/gen2brain/beeep"
	"github.com/getlantern/systray"

	"tailscale.com/client/tailscale"
)

var (
	//go:embed icon/on.png
	iconOn []byte
	//go:embed icon/off.png
	iconOff []byte
)

var (
	mu          sync.RWMutex
	myIP        string
	localClient tailscale.LocalClient
	//loadError    = false
	//needsLogin   = false
	errorMessage = ""
)

// set login-url as a variable in registry

// implement the OIDC scenario to

// add an entry to specify a preshared key

// tailscale local client to use for IPN

func main() {
	systray.Run(onReady, nil)
}

// change the function to pass mandatory parameters with login-url
func doConnectionControl(m *systray.MenuItem, verb string) {
	for {
		if _, ok := <-m.ClickedCh; !ok {
			break
		}
		log.Printf("launch command: tailscale %s", verb)
		b, err := execCommand("tailscale", verb)
		if err != nil {
			beeep.Notify(
				"Cyber Vpn",
				string(b),
				"",
			)
		}
	}
}

func exitSystray(m *systray.MenuItem) {
	<-m.ClickedCh
	systray.Quit()
}

func waitForLogin(m *systray.MenuItem) {
	for {
		<-m.ClickedCh
		log.Printf("Do login by opening browser")
		m.Disable()
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
		openBrowser(url)
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
	bs, _ := json.Marshal(st)
	fmt.Println(string(bs))

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
	mNetworkDevices.Hide()
	mMyDevices := mNetworkDevices.AddSubMenuItem("My Devices", "")
	//mTailscaleServices := mNetworkDevices.AddSubMenuItem("Tailscale Services", "")

	systray.AddSeparator()
	mAdminConsole := systray.AddMenuItem("Admin Console...", "")
	go waitForClickAndOpenBrowser(mAdminConsole, adminUrl)

	systray.AddSeparator()
	mExit := systray.AddMenuItem("Exit", "")
	go exitSystray(mExit)

	systray.AddSeparator()
	mLogout := systray.AddMenuItem("Logout...", "")
	mLogout.Hide()
	go doConnectionControl(mLogout, "logout")

	systray.AddSeparator()

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
					mMyDevices.Show()
					mThisDevice.Show()
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
				ip := ps.TailscaleIPs[0].String()
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
