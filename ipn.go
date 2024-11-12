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

// Fonction pour lancer le processus "up" avec la connexion VPN.
func runUp(ctx context.Context, cmd string, prefs *ipn.Prefs,
	forceReauth bool, authKey string, timeout time.Duration,
	success SuccessCallback, failure FailureCallback) (retErr error) {

	var simpleUp = false

	// Logging de la tentative de connexion.
	log.Printf("tentative de connexion...[%s]", cmd)
	st, err := localClient.Status(ctx)
	if err != nil {
		log.Println(err.Error())
		if failure != nil {
			failure(err)
		}
		return err
	}
	origAuthURL := st.AuthURL
	log.Printf("origAuthURL: %s", origAuthURL)

	// printAuthURL: Fonction pour déterminer si l'URL d'authentification doit être affichée.
	printAuthURL := func(url string) bool {
		if authKey != "" {
			// Si une authKey est fournie, ne pas afficher l'URL d'authentification.
			return false
		}
		if forceReauth && url == origAuthURL {
			log.Printf("force re-auth: %t", forceReauth)
			return false
		}
		return true
	}

	// Récupération des préférences actuelles du client local.
	curPrefs, err := localClient.GetPrefs(ctx)
	if err != nil {
		return err
	}
	if cmd == "up" {
		// "up" simple sans modification des préférences du profil.
		prefs.ProfileName = curPrefs.ProfileName
		simpleUp = true
	} else {
		// Modification des préférences pour changer le login.
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

	// Création d'un contexte annulable pour la surveillance de l'état.
	watchCtx, cancelWatch := context.WithCancel(ctx)
	defer cancelWatch()
	watcher, err := localClient.WatchIPNBus(watchCtx, 0)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	defer watcher.Close()

	// Gestion des interruptions (ex: SIGINT, SIGTERM).
	go func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-interrupt:
			cancelWatch()
		case <-watchCtx.Done():
		}
	}()

	running := make(chan bool, 1) // Signal pour indiquer que l'état est "Running".
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
					log.Printf("should start login interactive")
					startLoginInteractive()
				case ipn.NeedsMachineAuth:
					log.Printf("\nTo authorize your machine, visit (as admin):\n\n\t%s\n\n", prefs.AdminPageURL())
				case ipn.Running:
					// Authentification complète terminée.
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
				log.Printf("have to print url : %t", haveToPrint)
			}

			if url != nil {
				log.Printf("\nTo authenticate, visit:\n\n\t%s\n\n", *url)
				if success != nil {
					success(*url)
				}
			}
		}
	}()

	// Cas spécial : commande "up" simple pour démarrer.
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

	// Attente de l'état "Running" ou d'une erreur.
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

// Fonction pour arrêter le service VPN.
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
