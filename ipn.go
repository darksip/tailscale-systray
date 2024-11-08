package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"tailscale.com/ipn"
)

type SuccessCallback func(url string)
type FailureCallback func(err error)

var (
	prefsOfFlag = map[string][]string{} // "exit-node" => ExitNodeIP, ExitNodeID
)

func runUp(ctx context.Context, cmd string, prefs *ipn.Prefs,
	forceReauth bool, authKey string, timeout time.Duration,
	success SuccessCallback, failure FailureCallback) (retErr error) {

	var simpleUp = false

	log.Printf("tentative de connexion...[%s]", cmd)
	st, err := localClient.Status(ctx)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	origAuthURL := st.AuthURL
	log.Printf("origAuthURL: %s", origAuthURL)
	// printAuthURL reports whether we should print out the
	// provided auth URL from an IPN notify.
	printAuthURL := func(url string) bool {
		if authKey != "" {
			// Issue 1755: when using an authkey, don't
			// show an authURL that might still be pending
			// from a previous non-completed interactive
			// login.
			return false
		}
		if forceReauth && url == origAuthURL {
			if url == origAuthURL {
				log.Printf("no change in url ... skipping")
			}
			log.Printf("force re-auth: %t", forceReauth)
			return false
		}
		return true
	}

	curPrefs, err := localClient.GetPrefs(ctx)
	if err != nil {
		return err
	}
	if cmd == "up" {
		// "tailscale up" should not be able to change the
		// profile name.
		prefs.ProfileName = curPrefs.ProfileName
		simpleUp = true
	} else {
		// on veut changer le login

		justEditMP := new(ipn.MaskedPrefs)
		justEditMP.Prefs = *prefs
		justEditMP.ControlURLSet = true
		justEditMP.WantRunning = true
		justEditMP.WantRunningSet = true
		justEditMP.ForceDaemon = true
		justEditMP.ForceDaemonSet = true
		_, err := localClient.EditPrefs(ctx, justEditMP)
		if err != nil {
			log.Println(err.Error())
			return err
		}
		log.Printf("Prefs Edited...")
		forceReauth = true
	}

	watchCtx, cancelWatch := context.WithCancel(ctx)
	defer cancelWatch()
	watcher, err := localClient.WatchIPNBus(watchCtx, 0)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	defer watcher.Close()

	go func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-interrupt:
			cancelWatch()
		case <-watchCtx.Done():
		}
	}()

	running := make(chan bool, 1) // gets value once in state ipn.Running
	pumpErr := make(chan error, 1)
	var loginOnce sync.Once
	startLoginInteractive := func() { loginOnce.Do(func() { localClient.StartLoginInteractive(ctx) }) }
	log.Printf("launch watcher loop...")
	go func() {
		for {
			n, err := watcher.Next()
			if err != nil {
				pumpErr <- err
				return
			}
			if n.ErrMessage != nil {
				msg := *n.ErrMessage
				log.Printf("backend error: %v\n", msg)
			}
			if s := n.State; s != nil {
				log.Printf("watcher state: %s", s)
				switch *s {
				case ipn.NeedsLogin, ipn.NoState:
					log.Printf("should start login interacive")
					startLoginInteractive()
				case ipn.NeedsMachineAuth:
					log.Printf("\nTo authorize your machine, visit (as admin):\n\n\t%s\n\n", prefs.AdminPageURL())
				case ipn.Running:
					// Done full authentication process
					log.Printf("Success.\n")
					select {
					case running <- true:
					default:
					}
					cancelWatch()
				}
			}
			url := n.BrowseToURL
			if url != nil {
				haveToPrint := printAuthURL(*url)
				log.Printf("have to  print url : %t", haveToPrint)
			}

			if url != nil {
				log.Printf("\nTo authenticate, visit:\n\n\t%s\n\n", *url)
				if success != nil {
					success(*url)
				}
			}
		}
	}()

	// Special case: bare "tailscale up" means to just start
	// running, if there's ever been a login.
	if simpleUp {
		_, err := localClient.EditPrefs(ctx, &ipn.MaskedPrefs{
			Prefs: ipn.Prefs{
				WantRunning: true,
			},
			WantRunningSet: true,
		})
		if err != nil {
			return err
		}
	} else {
		if err := localClient.CheckPrefs(ctx, prefs); err != nil {
			return err
		}

		if err := localClient.Start(ctx, ipn.Options{
			AuthKey:     authKey,
			UpdatePrefs: prefs,
		}); err != nil {
			return err
		}
		if forceReauth {
			startLoginInteractive()
		}
	}

	// This whole 'up' mechanism is too complicated and results in
	// hairy stuff like this select. We're ultimately waiting for
	// 'running' to be done, but even in the case where
	// it succeeds, other parts may shut down concurrently so we
	// need to prioritize reads from 'running' if it's
	// readable; its send does happen before the pump mechanism
	// shuts down. (Issue 2333)
	var timeoutCh <-chan time.Time
	if timeout > 0 {
		timeoutTimer := time.NewTimer(timeout)
		defer timeoutTimer.Stop()
		timeoutCh = timeoutTimer.C
	}
	select {
	case <-running:
		return nil
	case <-watchCtx.Done():
		select {
		case <-running:
			return nil
		default:
		}
		return watchCtx.Err()
	case err := <-pumpErr:
		select {
		case <-running:
			return nil
		default:
		}
		return err
	case <-timeoutCh:
		return errors.New(`timeout waiting for CyberVpn service to enter a Running state; check health with "cybervpn-cli status"`)
	}
}

// func effectiveGOOS() string {
// 	if v := os.Getenv("TS_DEBUG_UP_FLAG_GOOS"); v != "" {
// 		return v
// 	}
// 	return runtime.GOOS
// }

// // exitNodeIP returns the exit node IP from p, using st to map
// // it from its ID form to an IP address if needed.
// func exitNodeIP(p *ipn.Prefs, st *ipnstate.Status) (ip netip.Addr) {
// 	if p == nil {
// 		return
// 	}
// 	if p.ExitNodeIP.IsValid() {
// 		return p.ExitNodeIP
// 	}
// 	id := p.ExitNodeID
// 	if id.IsZero() {
// 		return
// 	}
// 	for _, p := range st.Peer {
// 		if p.ID == id {
// 			if len(p.TailscaleIPs) > 0 {
// 				return p.TailscaleIPs[0]
// 			}
// 			break
// 		}
// 	}
// 	return
// }

func runDown(ctx context.Context) error {

	st, err := localClient.Status(ctx)
	if err != nil {
		return fmt.Errorf("error fetching current status: %w", err)
	}
	if st.BackendState == "Stopped" {
		log.Printf("Tailscale was already stopped.\n")
		return nil
	}
	_, err = localClient.EditPrefs(ctx, &ipn.MaskedPrefs{
		Prefs: ipn.Prefs{
			WantRunning: false,
		},
		WantRunningSet: true,
	})
	return err
}
