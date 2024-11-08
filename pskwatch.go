package main

import (
	"log"

	"github.com/fsnotify/fsnotify"
)

func StartWatch(path string, callback func() (int, error)) {

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("Fichier modifié :", event.Name)
					// Appeler la fonction de callback pour rafraichir le menu
					callback()
				}
				if event.Op&fsnotify.Remove == fsnotify.Remove {
					log.Println("Fichier supprimé :", event.Name)
					callback()
				}
				// if event.Op&fsnotify.Create == fsnotify.Create {
				// 	log.Println("Fichier crée :", event.Name)
				// 	// Appeler la fonction de callback pour rafraichir le menu
				// 	callback()
				// }
				// TODO: il faudrait traiter la suppression
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Error:", err)
			}
		}
	}()

	err = watcher.Add(path)
	if err != nil {
		log.Fatal(err)
	}

	<-done

}
