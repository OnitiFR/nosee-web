package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Worker struct {
	sondes    []*Sonde
	dirSondes string
}

/**
* Initial load of sondes
 */
func (w *Worker) InitialLoadSondes() error {
	if _, err := os.Stat(w.dirSondes); err != nil {
		return err
	}

	files, err := ioutil.ReadDir(w.dirSondes)
	if err != nil {
		return err
	}
	sondes := make([]*Sonde, 0)
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".toml") {
			continue
		}

		sonde, err := LoadFromToml(w.dirSondes + "/" + file.Name())
		if err != nil {
			return err
		}

		sondes = append(sondes, sonde)
	}

	w.sondes = sondes

	return nil
}

/**
* Observe the directory for update / create /remove sondes
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
					for i, sonde := range w.sondes {
						if sonde.FileName == event.Name {
							w.sondes = append(w.sondes[:i], w.sondes[i+1:]...)
							break
						}
					}
					fmt.Printf("Sonde %s supprimée\n", event.Name)
				} else {
					sonde, err := LoadFromToml(event.Name)
					if err != nil {
						fmt.Println(err)
						continue
					}
					hasBeenUpdated := false
					for _, sondeExist := range w.sondes {
						if sondeExist.FileName == event.Name {
							sondeExist.Update(sonde)
							hasBeenUpdated = true
							break
						}
					}
					if !hasBeenUpdated {
						w.sondes = append(w.sondes, sonde)
					}
					fmt.Printf("Sonde %s ajoutée ou mise à jour\n", sonde.Name)
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

func (w *Worker) Run() error {
	ch := make(chan *Sonde)
	var errorsSondes []*Sonde

	for {
		for _, sonde := range w.sondes {
			if sonde.NextExecution.Before(time.Now()) {
				go sonde.Check(ch)
				sonde := <-ch

				wasOnError := false
				for i, sondeError := range errorsSondes {
					if sondeError.FileName == sonde.FileName {
						wasOnError = true
						errorsSondes = append(errorsSondes[:i], errorsSondes[i+1:]...)
						break
					}
				}

				if sonde.LastStatus != ErrNone {
					errorsSondes = append(errorsSondes, sonde)
				}

				sonde.DisplayInformations(wasOnError)
			}
		}

		time.Sleep(time.Second * 1)
	}
}

func NewWorker(dirSondes string) *Worker {
	return &Worker{
		dirSondes: dirSondes,
	}
}
