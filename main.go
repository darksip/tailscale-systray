package main

//go:generate goversioninfo

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/atotto/clipboard"

	"tailscale.com/client/tailscale"
	"tailscale.com/ipn/ipnstate"
)

// il faudrait faire une struct pour refleter l etat de la struct dans l interface

var (
	mu           sync.RWMutex
	myIP         string
	localClient  tailscale.LocalClient
	errorMessage = ""
	myVersion    = "1.20.4"
	// array of string containing pre-shared keys for authentification
	presharedKeys = map[string]string{}
	pskIds        = map[string]string{}
	currentPsk    = ""
	pskDir        = filepath.Join(appdatapath, "psks")
)

// tailscale local client to use for IPN

// exitIfAlreadyRunning vérifie si le programme est déjà en cours d'exécution et quitte s'il l'est.
func exitIfAlreadyRunning() {
	addr := "localhost:25169"
	l, err := net.Listen("tcp", addr)

	if err != nil {
		log.Print("Le programme est déjà en cours d'exécution.")
		os.Exit(1)
	}
	defer l.Close() // Ferme le listener quand la fonction est terminée.

	// Garder l'écoute indéfiniment pour bloquer le port.
	go func() {
		for {
			// Essayez de réécouter sur le même port, s'il y a une erreur, loggez-la.
			l, err := net.Listen("tcp", addr)
			if err != nil {
				log.Printf("Erreur lors de l'écoute sur %s : %s", addr, err)
				return
			}
			defer l.Close()
			for {
				time.Sleep(3 * time.Second)
			}
		}
	}()
}

// copyEnvFile copie un fichier d'environnement vers le répertoire spécifié.
func copyEnvFile(path string) error {
	// Vérifie si le répertoire cible existe, sinon le crée.
	if _, err := os.Stat(pskDir); os.IsNotExist(err) {
		err = os.MkdirAll(pskDir, 0755)
		if err != nil {
			return err // Retourne une erreur si la création échoue.
		}
	}

	// Construit le chemin de sortie à partir du nom de base du fichier source.
	bname := filepath.Base(path)
	pathOut := filepath.Join(pskDir, bname)

	// Ouvre le fichier source en lecture seule.
	src, err := os.Open(path)
	if err != nil {
		return err // Retourne une erreur si l'ouverture échoue.
	}
	defer func() {
		if cerr := src.Close(); cerr != nil {
			log.Printf("Erreur lors de la fermeture du fichier source: %s", cerr)
		}
	}()

	// Crée le fichier de destination en écriture.
	dst, err := os.Create(pathOut)
	if err != nil {
		return err // Retourne une erreur si la création échoue.
	}
	defer func() {
		if cerr := dst.Close(); cerr != nil {
			log.Printf("Erreur lors de la fermeture du fichier destination: %s", cerr)
		}
	}()

	log.Printf("Copie du fichier %s vers %s", path, pathOut)

	// Copie le contenu du fichier source vers le fichier de destination.
	_, err = io.Copy(dst, src)
	if err != nil {
		log.Println("Erreur lors de la copie du fichier:", err)
		return err
	}

	return nil
}

func main() {

	if IsWindowsServer() {
		log.Printf("Execution sur une plateforme serveur\non utilise la presharedkey")
	}
	// load environement parameters from %programdata%\.env
	loadEnv(false)

	latencies = make(map[string][]float64)
	movLatencies = map[string]float64{}
	nping = 0

	iconOn = iconOnIco
	iconOff = iconOffIco

	if len(os.Args) > 1 {
		// en cas d'appel suite a double clic sur un fichier psk
		err := copyEnvFile(os.Args[1])
		if err != nil {
			log.Println(err)
		}
	}

	exitIfAlreadyRunning()

	RunWalk()
	// run getlantern systray
	//RunGl()
}

func Notify(message string, iconame string) {
	NotifyWalk(message, iconame)
	//NotifyGL(message)
}

func addMenuHandlers() {
	// compose complete menu with hidden options
	AddConnectionHandlersToMenu()

	if noExitNode == 0 {
		AddExitNodeHandlersToMenu()
	}

	sm.SetHandler("ADMIN", func() {
		err := openBrowser(adminUrl)
		if err != nil {
			Notify(err.Error(), "error")
		}
	})
	sm.SetHandler("SHOW_ERROR", func() {
		Notify(errorMessage, "error")
	})

	sm.SetHandler("MYIP", func() {
		err := clipboard.WriteAll(myIP)
		if err == nil {
			Notify(fmt.Sprintf("Copy the IP address (%s) to the Clipboard", myIP), "info")
		}
	})
}

func setMenuPreSharedKeys() (pskNumber int, err error) {
	// si le fichier authUrl.txt dans appdata existe, recupere le ConnectionInfo
	var cId string
	ci, err := readAuthUrl()
	if err == nil {
		cId = getClientId(ci)
	}
	// by default hide all presaredkey entry in menu
	sm.SetHiddenAll([]string{"PSK1", "PSK2", "PSK3", "PSK4", "PSK5"}, true)
	// search for preshared key files (*.psk) in appdata folder

	files, err := os.ReadDir(pskDir)
	if err != nil {
		log.Printf("Error reading preshared key directory: %s", err)
		return 0, err
	} else {
		idpsk := 1
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".psk") {
				pskName := strings.TrimSuffix(file.Name(), ".psk")
				pskId := "PSK" + strconv.Itoa(idpsk)
				if idpsk > 5 {
					log.Println("maximum de fichiers psk atteints")
					break
				}

				sm.SetHandler(pskId, func() {

					log.Printf("Déconnexion en cours...")
					if err := localClient.Logout(context.TODO()); err != nil {
						log.Printf("Erreur lors de la déconnexion: %s", err.Error())
					}

					rootUrl = fmt.Sprintf("https://head.%s.cyberfile.fr", pskName)
					authKey = presharedKeys[pskName]

					log.Printf("Login avec la clé prépartagée %s (%s)", pskName, authKey)
					log.Printf("Sur le domaine %s", rootUrl)
					doLogin()

					// loop on presharedKeys keys
					icon := "empty"
					for name := range presharedKeys {
						if name == pskName {
							icon = "blueballoon"
						}
						sm.SetIcon(pskIds[name], icon)
					}

					Notify(fmt.Sprintf("Connexion effectuée sur %s avec la clé %s", rootUrl, pskName), "info")
				})
				sm.SetLabel(pskId, pskName)
				sm.SetHidden(pskId, false)
				if pskName == cId {
					sm.SetIcon(pskId, "blueballoon")
				} else {
					sm.SetIcon(pskId, "empty")
				}
				pskNumber++
				//get authkey from content of psk file
				//open file
				pskFile, err := os.Open(filepath.Join(pskDir, file.Name()))
				// read content in authkey variable
				if err == nil {
					defer pskFile.Close()
					authKey, err := io.ReadAll(pskFile)
					if err != nil {
						log.Printf("Error reading preshared key file: %s", err)
						return 0, err
					} else {
						// add authkey to map
						presharedKeys[pskName] = string(authKey)
						pskIds[pskName] = pskId
						idpsk++
					}
				} else {
					log.Printf("Error reading preshared key file: %s", err)
				}
			}
		}
	}
	return pskNumber, nil
}

func setMenuState(status *ipnstate.Status) (exit bool) {
	switch status.BackendState {
	case "NeedsLogin", "NoState":
		sm.SetHiddenAll([]string{"CONNECT", "DISCONNECT", "EXITNODE_ON", "EXITNODE_OFF", "LOGOUT"}, true)
		sm.SetHiddenAll([]string{"EXITNODES", "EN1", "EN2", "EN3", "EN4", "EN5"}, true)
		sm.SetHidden("LOGIN", false)
		sm.SetDisabled("LOGIN", false)
		sm.SetIcon("", "off")
		sm.SetIcon("MYIP", "redballoon")
		return true
	case "Stopped":
		sm.SetHiddenAll([]string{"DISCONNECT", "EXITNODE_ON", "EXITNODE_OFF", "LOGIN"}, true)
		sm.SetHiddenAll([]string{"EXITNODES", "EN1", "EN2", "EN3", "EN4", "EN5"}, true)
		sm.SetHiddenAll([]string{"LOGOUT", "CONNECT"}, false)
		sm.SetIcon("", "off")
		sm.SetIcon("MYIP", "greyballoon")
		return true
	case "Running", "Starting":
		sm.SetHiddenAll([]string{"CONNECT", "EXITNODE_ON", "EXITNODE_OFF", "LOGIN"}, true)
		sm.SetDisabled("LOGIN", true)
		if status.ExitNodeStatus != nil {
			sm.SetHidden("EXITNODES", false)
			sm.SetHidden("EXITNODE_OFF", false)
			//sm.SetDisabled("EXITNODE_OFF", false)
		} else {
			sm.SetHiddenAll([]string{"EXITNODES", "EN1", "EN2", "EN3", "EN4", "EN5"}, true)
			sm.SetHidden("EXITNODE_ON", false)
			sm.SetHidden("EXITNODE_OFF", true)
			//sm.SetDisabled("EXITNODE_OFF", false)
		}
		sm.SetHiddenAll([]string{"LOGOUT", "DISCONNECT"}, false)
		sm.SetIcon("", "on")
		sm.SetIcon("MYIP", "blueballoon")
	}
	return false
}

// fonction appelee des que le menu est pret
func onMenuReady() {

	log.Printf("getting localClient...")
	getStatus := localClient.Status
	st, _ := getStatus(context.TODO())

	// add handlers to menu items
	addMenuHandlers()
	// set default icon to gray logo
	sm.SetIcon("", "off")

	//TODO: ajouter un retour pour ne pas lancer le login oAuth si psk trouvées
	pskNb, err := setMenuPreSharedKeys()
	if err != nil {
		log.Printf("setMenuPreSharedKeys failed : %s", err.Error())
	}

	if st != nil {
		if st.BackendState == "NeedsLogin" || st.BackendState == "NoState" {
			if pskNb == 0 {
				Notify("Cyber Vpn needs you to login...\nPlease wait while trying to reach the server...", "needslogin")
				sm.SetDisabled("LOGIN", true)
				go doLogin()
			}
		}
		if strings.ToLower(st.BackendState) == "stopped" {
			Notify("Cyber Vpn is disconnected\nRight Ckick on systray icon\n and choose Connect", "disconnected")
			sm.SetDisabled("LOGIN", false)
		}
	} else {
		log.Println("The service CyberVpn does not respond")
	}

	// Canal pour arrêter proprement la surveillance des fichiers.
	quit := make(chan bool)

	// Surveillance du répertoire `pskDir` pour des modifications, suppressions, ou créations.
	go func() {
		if _, err := os.Stat(pskDir); os.IsNotExist(err) {
			err = os.MkdirAll(pskDir, 0755)
			if err != nil {
				log.Printf("Erreur lors de la création du répertoire %s : %s", pskDir, err)
				return
			}
		}
		StartWatch(pskDir, setMenuPreSharedKeys, quit)
	}()

	// launch monitor and auto-update loop
	go func() {
		lastUpdateNotification := time.Now().AddDate(0, 0, -2)
		launchMsi := ""

		sm.SetDisabled("UPDATE", false)
		sm.SetHidden("UPDATE", true)
		sm.SetIcon("UPDATE", "caution")
		for {
			log.Printf("local client version: %s", myVersion)
			// call monitoring function to report status

			// if not already waiting for install check for newer version
			status, newVersionPath, err := checkAndDownload()
			if err == nil {
				log.Printf("status: %s", status)
				// si up to date -> on passe
				if status == "up to date" {
					sm.SetHidden("UPDATE", true)
					time.Sleep(15 * time.Second)
					continue
				} else if status == "successful download" {
					Notify("Une  mise à jour du logciel est disponible.\nCliquez droit sur l'icone du systray et choisissez\nMise a jour", "caution")
					launchMsi = newVersionPath
					lastUpdateNotification = time.Now()
					sm.SetHidden("UPDATE", false)
					sm.SetHandler("UPDATE", func() {
						log.Printf("launch %s ...", launchMsi)

						_, err := execCommand("msiexec", "/i", launchMsi)
						if err == nil {
							os.Exit(0)
						} else {
							errorMessage = err.Error()
							sm.SetHidden("SHOW_ERROR", false)
							sm.SetDisabled("SHOW_ERROR", false)
						}
					})
				} else if status == "already downloaded" {
					if time.Since(lastUpdateNotification).Hours() >= 24 {
						Notify("Une  mise à jour du logciel est disponible.\nCliquez droit sur l'icone du systray et choisissez\nMise a jour", "caution")
						lastUpdateNotification = time.Now()
						launchMsi = newVersionPath
						sm.SetHandler("UPDATE", func() {
							log.Printf("launch %s ...", launchMsi)
							_, err := execCommand("msiexec", "/i", launchMsi)
							if err == nil {
								os.Exit(0)
							} else {
								errorMessage = err.Error()
								sm.SetHidden("SHOW_ERROR", false)
								sm.SetDisabled("SHOW_ERROR", false)
							}
						})
					}
					sm.SetHidden("UPDATE", false)

				} else {
					log.Printf("status innatendu: %s", status)
				}
			}
			// else check if we have to notify or add to menu
			time.Sleep(15 * time.Second)
		}
	}()
	// base deamon looping forever
	go func() {
		for {
			time.Sleep(3 * time.Second)

			status, err := getStatus(context.TODO())
			// update sytray menu regarding the Backend State
			if err != nil {
				errorMessage = err.Error()
				log.Printf("%s", errorMessage)
				sm.SetHiddenAll([]string{"CONNECT", "LOGIN", "DISCONNECT", "EXITNODE_ON", "EXINODE_OFF", "LOGOUT"}, true)
				sm.SetHiddenAll([]string{"EXITNODES", "EN1", "EN2", "EN3", "EN4", "EN5"}, true)

				sm.SetHidden("SHOW_ERROR", false)
				sm.SetIcon("", "off")
				continue
			} else {
				log.Printf("State: %s", status.BackendState)
				errorMessage = ""

				sm.SetHidden("SHOW_ERROR", true)
				sm.SetLabel("STATUS", status.BackendState)
				sm.SetIcon("CYBERVPN", "off16")
				if setMenuState(status) {
					if noExitNode > 0 {
						sm.SetHiddenAll([]string{"EXITNODE_ON", "EXITNODE_OFF", "EXITNODES", "EN1", "EN2", "EN3", "EN4", "EN5"}, true)
					}
					// if the status is NeedsLogin or NoState and manualLogout==0
					// probably neeeds login after token expiration in sleep mode
					if status.BackendState == "NeedsLogin" || status.BackendState == "NoState" {
						if len(presharedKeys) == 0 {
							if !loginIsProcessing && manualLogout == 0 {
								// user did not asked for loggout an no login is already processing
								log.Printf("got to log in, token expired...")
								go func() {
									sm.SetDisabled("LOGIN", true)
									doLogin()
									sm.SetDisabled("LOGIN", false)
								}()
							} else {
								log.Printf("don't have to log in : loginIsProcessing[%t] manualLogout[%d]", loginIsProcessing, manualLogout)
							}
						}
					}
					// if the state is not Running don't do exitNodes Check
					continue
				}
			}

			if noExitNode > 0 {
				sm.SetHiddenAll([]string{"EXITNODE_ON", "EXITNODE_OFF", "EXITNODES", "EN1", "EN2", "EN3", "EN4", "EN5"}, true)
			}
			mu.Lock()

			if len(status.TailscaleIPs) != 0 {
				myIP = status.TailscaleIPs[1].String()
				log.Printf("my ip: %s", myIP)
				sm.SetLabel("MYIP", myIP)
			}
			if wantsToDisableExitNodes || (noExitNode > 0) {
				log.Println("wants exit nodes to be disabled...")
				setExitNodeOff()
				mu.Unlock()
				// do not check the best exit node if disabled wanted
				continue
			}
			if noExitNode == 0 {
				refreshExitNodes()
				bestIp := checkLatency()
				showOrderedExitNode(bestIp)
				if status.ExitNodeStatus != nil {
					if len(status.ExitNodeStatus.TailscaleIPs) > 1 {
						activeExitNode = status.ExitNodeStatus.TailscaleIPs[1].Addr().String()
						checkActiveNodeAndSetExitNode()
					}
				} else {
					setExitNode()
				}
			}

			mu.Unlock()
			// gestion des Peers dans une fenetre separée pour ne faire
			// l'interrogation qu'à l'ouverture de la fenêtre
		}
	}()
}
