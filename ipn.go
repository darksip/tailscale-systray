package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/netip"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"sync"
	"syscall"
	"time"

	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/version/distro"
)

type SuccessCallback func(url string)
type FailureCallback func(err error)

var (
	prefsOfFlag = map[string][]string{} // "exit-node" => ExitNodeIP, ExitNodeID
)

//	func getStatus() {
//		return
//	}
//
// updatePrefs returns how to edit preferences based on the
// flag-provided 'prefs' and the currently active 'curPrefs'.
//
// It returns a non-nil justEditMP if we're already running and none of
// the flags require a restart, so we can just do an EditPrefs call and
// change the prefs at runtime (e.g. changing hostname, changing
// advertised routes, etc).
//
// It returns simpleUp if we're running a simple "tailscale up" to
// transition to running from a previously-logged-in but down state,
// without changing any settings.
func updatePrefs(prefs, curPrefs *ipn.Prefs, env upCheckEnv, backendState string, authKey string, forceReauth bool) (simpleUp bool, justEditMP *ipn.MaskedPrefs, err error) {
	// if !env.upArgs.reset {
	// 	applyImplicitPrefs(prefs, curPrefs, env)
	// }

	controlURLChanged := curPrefs.ControlURL != prefs.ControlURL

	tagsChanged := !reflect.DeepEqual(curPrefs.AdvertiseTags, prefs.AdvertiseTags)

	simpleUp = curPrefs.Persist != nil &&
		curPrefs.Persist.LoginName != "" &&
		backendState != ipn.NeedsLogin.String()

	justEdit := backendState == ipn.Running.String() &&
		!forceReauth &&
		authKey == "" &&
		!controlURLChanged &&
		!tagsChanged

	if justEdit {
		justEditMP = new(ipn.MaskedPrefs)
		justEditMP.WantRunningSet = true
		justEditMP.Prefs = *prefs

		visitFlags := env.flagSet.Visit
		if env.upArgs.reset {
			visitFlags = env.flagSet.VisitAll
		}
		visitFlags(func(f *flag.Flag) {
			updateMaskedPrefsFromUpOrSetFlag(justEditMP, f.Name)
		})
	}

	return simpleUp, justEditMP, nil
}

func updateMaskedPrefsFromUpOrSetFlag(mp *ipn.MaskedPrefs, flagName string) {
	if preflessFlag(flagName) {
		return
	}
	if prefs, ok := prefsOfFlag[flagName]; ok {
		for _, pref := range prefs {
			reflect.ValueOf(mp).Elem().FieldByName(pref + "Set").SetBool(true)
		}
		return
	}
	panic(fmt.Sprintf("internal error: unhandled flag %q", flagName))
}

// preflessFlag reports whether flagName is a flag that doesn't
// correspond to an ipn.Pref.
func preflessFlag(flagName string) bool {
	switch flagName {
	case "auth-key", "force-reauth", "reset", "qr", "json", "timeout", "accept-risk":
		return true
	}
	return false
}

const accidentalUpPrefix = "Error: changing settings via 'tailscale up' requires mentioning all\n" +
	"non-default flags. To proceed, either re-run your command with --reset or\n" +
	"use the command below to explicitly mention the current value of\n" +
	"all non-default settings:\n\n" +
	"\ttailscale up"

// upCheckEnv are extra parameters describing the environment as
// needed by checkForAccidentalSettingReverts and friends.
type upCheckEnv struct {
	goos          string
	user          string
	flagSet       *flag.FlagSet
	upArgs        upArgsT
	backendState  string
	curExitNodeIP netip.Addr
	distro        distro.Distro
}

type upArgsT struct {
	qr                     bool
	reset                  bool
	server                 string
	acceptRoutes           bool
	acceptDNS              bool
	singleRoutes           bool
	exitNodeIP             string
	exitNodeAllowLANAccess bool
	shieldsUp              bool
	runSSH                 bool
	forceReauth            bool
	forceDaemon            bool
	advertiseRoutes        string
	advertiseDefaultRoute  bool
	advertiseTags          string
	snat                   bool
	netfilterMode          string
	authKeyOrFile          string // "secret" or "file:/path/to/secret"
	hostname               string
	opUser                 string
	json                   bool
	timeout                time.Duration
	acceptedRisks          string
	profileName            string
}

// applyImplicitPrefs mutates prefs to add implicit preferences for the user operator.
// If the operator flag is passed no action is taken, otherwise this only needs to be set if it doesn't
// match the current user.
//
// curUser is os.Getenv("USER"). It's pulled out for testability.
func applyImplicitPrefs(prefs, oldPrefs *ipn.Prefs, env upCheckEnv) {
	if env.flagSet == nil {
		return
	}
	explicitOperator := false
	env.flagSet.Visit(func(f *flag.Flag) {
		if f.Name == "operator" {
			explicitOperator = true
		}
	})

	if prefs.OperatorUser == "" && oldPrefs.OperatorUser == env.user && !explicitOperator {
		prefs.OperatorUser = oldPrefs.OperatorUser
	}
}

func runUp(ctx context.Context, cmd string, prefs *ipn.Prefs,
	forceReauth bool, authKey string, timeout time.Duration,
	success SuccessCallback, failure FailureCallback) (retErr error) {

	st, err := localClient.Status(ctx)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	origAuthURL := st.AuthURL

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
			return false
		}
		return true
	}

	if len(prefs.AdvertiseRoutes) > 0 {
		if err := localClient.CheckIPForwarding(context.Background()); err != nil {
			log.Printf("%v", err)
		}
	}

	curPrefs, err := localClient.GetPrefs(ctx)
	if err != nil {
		return err
	}
	if cmd == "up" {
		// "tailscale up" should not be able to change the
		// profile name.
		prefs.ProfileName = curPrefs.ProfileName
	}

	upArgs := &upArgsT{
		forceReauth:   forceReauth,
		authKeyOrFile: authKey,
	}

	env := upCheckEnv{
		goos:          effectiveGOOS(),
		distro:        distro.Get(),
		upArgs:        *upArgs,
		user:          os.Getenv("USER"),
		backendState:  st.BackendState,
		curExitNodeIP: exitNodeIP(curPrefs, st),
	}

	simpleUp, justEditMP, err := updatePrefs(prefs, curPrefs, env, st.BackendState, authKey, forceReauth)
	if err != nil {
		log.Printf("%s", err)
	}
	if justEditMP != nil {
		//justEditMP.EggSet = egg
		_, err := localClient.EditPrefs(ctx, justEditMP)
		return err
	}

	watchCtx, cancelWatch := context.WithCancel(ctx)
	defer cancelWatch()
	watcher, err := localClient.WatchIPNBus(watchCtx, 0)
	if err != nil {
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
				switch *s {
				case ipn.NeedsLogin:
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
			if url := n.BrowseToURL; url != nil && printAuthURL(*url) {
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
		return errors.New(`timeout waiting for Tailscale service to enter a Running state; check health with "tailscale status"`)
	}
}

func effectiveGOOS() string {
	if v := os.Getenv("TS_DEBUG_UP_FLAG_GOOS"); v != "" {
		return v
	}
	return runtime.GOOS
}

// exitNodeIP returns the exit node IP from p, using st to map
// it from its ID form to an IP address if needed.
func exitNodeIP(p *ipn.Prefs, st *ipnstate.Status) (ip netip.Addr) {
	if p == nil {
		return
	}
	if p.ExitNodeIP.IsValid() {
		return p.ExitNodeIP
	}
	id := p.ExitNodeID
	if id.IsZero() {
		return
	}
	for _, p := range st.Peer {
		if p.ID == id {
			if len(p.TailscaleIPs) > 0 {
				return p.TailscaleIPs[0]
			}
			break
		}
	}
	return
}

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
