package main

import (
	"log"

	"github.com/fsnotify/fsnotify"
)

// StartWatch surveille les modifications sur un fichier ou un répertoire donné
// et exécute une fonction de callback lors de tout événement.
func StartWatch(path string, callback func() (int, error), quit chan bool) {
	// Crée un nouvel objet Watcher pour surveiller les changements de fichiers.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println("Erreur lors de la création du watcher:", err)
		return // En cas d'erreur, retourne pour ne pas stopper toute l'application.
	}
	defer watcher.Close() // Assure que le watcher est fermé proprement à la fin de l'exécution.

	// Crée un canal pour garder la fonction active tant que la surveillance est en cours.
	done := make(chan bool)
	// Lance une goroutine pour traiter les événements de surveillance en arrière-plan.
	go func() {
		for {
			select {
			// Écoute les événements du watcher.
			case event, ok := <-watcher.Events:
				if !ok {
					return // Si le canal est fermé, arrête la goroutine.
				}
				// Si un fichier est créé, modifié ou supprimé, appelle la fonction callback.
				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) != 0 {
					log.Println("Événement détecté :", event)
					if _, err := callback(); err != nil {
						log.Println("Erreur lors du callback:", err)
					}
				}

			// Écoute les erreurs du watcher.
			case err, ok := <-watcher.Errors:
				if !ok {
					return // Si le canal est fermé, arrête la goroutine.
				}
				log.Println("Erreur du watcher:", err) // Log les erreurs si elles surviennent.

			// Écoute les demandes d'arrêt via le canal quit.
			case <-quit:
				log.Println("Arrêt du watcher demandé")
				done <- true
				return
			}
		}
	}()

	// Ajoute le chemin du fichier ou répertoire à surveiller.
	err = watcher.Add(path)
	if err != nil {
		log.Println("Erreur lors de l'ajout du chemin:", err)
		return // En cas d'erreur, retourne pour ne pas stopper toute l'application.
	}

	// Bloque l'exécution jusqu'à la fermeture du canal done (ce qui ne se produit que sur demande via quit).
	<-done
}
