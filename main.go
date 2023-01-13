package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
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
	mu   sync.RWMutex
	myIP string
)

// add logout entry (hidden in prod)

// set login-url as a variable in registry

// implement the OIDC scenario to

// add an entry to specify a preshared key

// tailscale local client to use for IPN
var localClient tailscale.LocalClient

func main() {

	log.Printf("getting localClient...")
	getStatus := localClient.Status
	st, err := getStatus(context.TODO())

	if err == nil {
		log.Printf("api client version %s", st.Version)
		log.Printf("api auth url:  %s", st.AuthURL)
		//cfg := localClient.

	} else {
		log.Printf("%s", err.Error())
	}
	log.Printf("launching systray...")
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

func onReady() {
	systray.SetIcon(iconOff)

	mConnect := systray.AddMenuItem("Connect", "")
	mConnect.Enable()
	mDisconnect := systray.AddMenuItem("Disconnect", "")
	mDisconnect.Disable()

	go doConnectionControl(mConnect, "up")
	go doConnectionControl(mDisconnect, "down")

	systray.AddSeparator()

	mThisDevice := systray.AddMenuItem("This device:", "")
	go func(mThisDevice *systray.MenuItem) {
		for {
			_, ok := <-mThisDevice.ClickedCh
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
	}(mThisDevice)

	mNetworkDevices := systray.AddMenuItem("Network Devices", "")
	mMyDevices := mNetworkDevices.AddSubMenuItem("My Devices", "")
	mTailscaleServices := mNetworkDevices.AddSubMenuItem("Tailscale Services", "")

	systray.AddSeparator()
	mAdminConsole := systray.AddMenuItem("Admin Console...", "")

	go func() {
		for {
			_, ok := <-mAdminConsole.ClickedCh
			if !ok {
				break
			}
			openBrowser("https://login.tailscale.com/admin/machines")
		}
	}()

	systray.AddSeparator()

	mExit := systray.AddMenuItem("Exit", "")
	go func() {
		<-mExit.ClickedCh
		systray.Quit()
	}()

	systray.AddSeparator()
	mLogout := systray.AddMenuItem("Logout...", "")
	go doConnectionControl(mLogout, "logout")

	go func() {
		type Item struct {
			menu  *systray.MenuItem
			title string
			ip    string
			found bool
		}
		items := map[string]*Item{}

		enabled := false
		setDisconnected := func() {
			if enabled {
				systray.SetTooltip("Tailscale: Disconnected")
				mConnect.Enable()
				mDisconnect.Disable()
				systray.SetIcon(iconOff)
				enabled = false
			}
		}

		for {
			rawStatus, err := exec.Command("tailscale", "status", "--json").Output()
			if err != nil {
				setDisconnected()
				continue
			}

			status := new(Status)
			if err := json.Unmarshal(rawStatus, status); err != nil {
				setDisconnected()
				continue
			}

			mu.Lock()
			if len(status.Self.TailscaleIPs) != 0 {
				myIP = status.Self.TailscaleIPs[0]
			}
			mu.Unlock()

			if status.TailscaleUp && !enabled {
				systray.SetTooltip("Tailscale: Connected")
				mConnect.Disable()
				mDisconnect.Enable()
				systray.SetIcon(iconOn)
				enabled = true
			} else if !status.TailscaleUp && enabled {
				setDisconnected()
			}

			for _, v := range items {
				v.found = false
			}

			mThisDevice.SetTitle(fmt.Sprintf("This device: %s (%s)", status.Self.DisplayName.String(), myIP))

			for _, peer := range status.Peers {
				ip := peer.TailscaleIPs[0]
				peerName := peer.DisplayName
				title := peerName.String()

				var sub *systray.MenuItem
				switch peerName.(type) {
				case DNSName:
					sub = mMyDevices
				case HostName:
					sub = mTailscaleServices
				}

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
									"Tailscale",
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

			time.Sleep(10 * time.Second)
		}
	}()
}
