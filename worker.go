package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Worker struct {
	sondes    map[string]*Sonde
	errors    map[string]*SondeError
	dirSondes string
	mutex     *sync.Mutex
}

/**
* Initial load of sondes
 */
func (w *Worker) InitialLoadSondes() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if _, err := os.Stat(w.dirSondes); err != nil {
		return err
	}

	files, err := ioutil.ReadDir(w.dirSondes)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".toml") {
			continue
		}

		sonde, err := LoadFromToml(w.dirSondes + "/" + file.Name())
		if err != nil {
			return err
		}
		w.sondes[sonde.FileName] = sonde
	}

	w.DisplaySondesList()

	return nil
}

func (w *Worker) AppendSonde(sonde *Sonde) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.sondes[sonde.FileName] = sonde
}

func (w *Worker) RemoveSonde(filename string) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	delete(w.sondes, filename)
}

/**
* Observe le dossier des sondes pour détecter les ajouts et suppressions de fichiers
 */
func (w *Worker) ObserveSondeDir() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}

	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			// watch for events
			case event := <-watcher.Events:
				if !strings.HasSuffix(event.Name, ".toml") {
					continue
				}

				if event.Op == fsnotify.Remove {
					w.RemoveSonde(event.Name)
					fmt.Printf("Sonde %s supprimée\n", event.Name)
					w.DisplaySondesList()
				} else {
					sonde, err := LoadFromToml(event.Name)
					if err != nil {
						fmt.Println(err)
						continue
					}
					hasBeenUpdated := false
					for _, sondeExist := range w.sondes {
						if sondeExist.FileName == event.Name {
							w.mutex.Lock()
							sondeExist.Update(sonde)
							w.mutex.Unlock()

							hasBeenUpdated = true
							break
						}
					}
					if !hasBeenUpdated {
						w.AppendSonde(sonde)
					}
					fmt.Printf("Sonde %s ajoutée ou mise à jour\n", sonde.Name)
					w.DisplaySondesList()
				}
				// watch for errors
			case err := <-watcher.Errors:
				fmt.Println("ERROR", err)
			}
		}
	}()

	if err := watcher.Add(w.dirSondes); err != nil {
		fmt.Println("ERROR", err)
		panic(err)
	}

	<-done
}

/*
* Affiche la liste des sondes chargées
 */
func (w *Worker) DisplaySondesList() {
	fmt.Println("Liste des sondes surveillées :")
	for _, sonde := range w.sondes {
		fmt.Printf("%s\n", sonde.Name)
	}
}

/*
* Go routine qui écoute le channel des sondes
* Afin de traiter les nouvelles erreurs ou les erreurs résolues
 */
func (w *Worker) ListenChanSonde(chSonde chan *Sonde) {
	for {
		sonde := <-chSonde
		// Détection des erreurs qui ont disparu
		for hash, oldSerr := range w.errors {
			if oldSerr.FileName == sonde.FileName && oldSerr.IsErrorSolved(sonde.Errors) {
				delete(w.errors, hash)
				oldSerr.DisplayResolvedError(sonde)
			}
		}

		// On ajoute les nouvelles erreurs
		for _, sondeError := range sonde.Errors {
			if w.errors[sondeError.Hash] == nil {
				w.errors[sondeError.Hash] = sondeError
				sondeError.DisplayNewError(sonde)
			}
		}
	}
}

/*
* Lance le check sur toutes les sondes
 */
func (w *Worker) RunAllCheck(chSonde chan *Sonde) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	for _, sonde := range w.sondes {
		if time.Now().After(sonde.NextExecution) {
			time.Sleep(time.Millisecond * time.Duration((100 / len(w.sondes))))
			go sonde.Check(chSonde)
		}
	}
}

/*
* Point d'entrée du worker
 */
func (w *Worker) Run() error {
	chSonde := make(chan *Sonde)
	defer close(chSonde)

	go w.ListenChanSonde(chSonde)

	for {
		w.RunAllCheck(chSonde)
		time.Sleep(time.Second * 1)
	}
}

func NewWorker(dirSondes string) *Worker {
	return &Worker{
		dirSondes: dirSondes,
		mutex:     &sync.Mutex{},
		sondes:    make(map[string]*Sonde),
		errors:    make(map[string]*SondeError),
	}
}
