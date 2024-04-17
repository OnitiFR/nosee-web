package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

func (w *Worker) AppendSonde(sonde *Sonde) {
	for _, s := range w.sondes {
		if s.Name == sonde.Name || s.Url == sonde.Url {
			NotifySlack(fmt.Sprintf("Erreur lors du chargement de la sonde %s : une sonde portant le même nom ou url existe déjà fichier : %s", sonde.FileName, s.FileName), false)
			return
		}
	}
	w.sondes[sonde.FileName] = sonde

	if w.NotifySondeUpdate {
		NotifySlack(fmt.Sprintf("La sonde %s a été ajoutée", sonde.Name), true)
	}
}

func (w *Worker) RemoveSonde(filename string) {
	sonde := w.sondes[filename]
	delete(w.sondes, filename)

	if w.NotifySondeUpdate {
		NotifySlack(fmt.Sprintf("La sonde %s a été supprimée", sonde.Name), true)
	}
}

/**
 * Load a sonde from a file
 */
func LoadFromToml(fileSonde string) (*Sonde, error) {
	var sonde *Sonde
	_, err := toml.DecodeFile(fileSonde, &sonde)

	if err != nil {
		return sonde, err
	}
	// get basename from absolute path
	filename := fileSonde[strings.LastIndex(fileSonde, "/")+1:]
	sonde.FileName = filename
	sonde.NextExecution = time.Now()
	sonde.Errors = make(map[SondeErrorStatus]*SondeError)

	if sonde.NbRetentionsCritical <= 0 {
		sonde.NbRetentionsCritical = 1
	}
	if sonde.NbRetentionsWarning <= 0 {
		sonde.NbRetentionsWarning = 2
	}

	return sonde, err
}

/*
* Display sondes list
 */
func (w *Worker) DisplaySondesList() {
	fmt.Println("Liste des sondes surveillées :")
	for _, sonde := range w.sondes {
		fmt.Printf("%s\n", sonde.Name)
	}
}

/**
* Initial load of sondes
 */
func (w *Worker) InitialLoadSondes() error {
	fmt.Println("Chargement des sondes...")
	err := w.ScanSondeDirectory()
	if err == nil {
		w.DisplaySondesList()
	} else {
		fmt.Printf("Aucune sonde chargée : %s\n", err.Error())

	}

	return err
}

/**
* Observe sonde directory every 10 seconds
 */
func (w *Worker) ObserveSondeDir() {
	// Activate sonde update Notification
	w.NotifySondeUpdate = true

	var running_errors = make(map[string]bool)
	for {
		err := w.ScanSondeDirectory()
		if err != nil {
			if _, ok := running_errors[err.Error()]; !ok {
				NotifySlack(err.Error(), false)
				running_errors[err.Error()] = true
			}
		} else {
			running_errors = make(map[string]bool)
		}
		time.Sleep(1 * time.Hour)
	}
}

/**
* scan directory in order to update sondes list
 */
func (w *Worker) ScanSondeDirectory() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if _, err := os.Stat(w.dirSondes); err != nil {
		return err
	}

	files, err := ioutil.ReadDir(w.dirSondes)
	if err != nil {
		return err
	}

	var filesSondes map[string]bool = make(map[string]bool)

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".toml") {
			continue
		}

		sonde, err := LoadFromToml(w.dirSondes + "/" + file.Name())
		if err != nil {
			message := fmt.Sprintf("Erreur lors du chargement de la sonde %s : %s", file.Name(), err.Error())
			return errors.New(message)
		}

		// check if sonde already exists
		if _, ok := w.sondes[sonde.FileName]; !ok {
			w.AppendSonde(sonde)
			filesSondes[sonde.FileName] = true
		} else {
			if w.sondes[sonde.FileName].Update(sonde) && w.NotifySondeUpdate {
				NotifySlack(fmt.Sprintf("La sonde %s a été mise à jour", sonde.Name), true)
			}
			filesSondes[sonde.FileName] = true
		}
	}

	// check if some sondes have been removed
	for filename, _ := range w.sondes {
		if _, ok := filesSondes[filename]; !ok {
			w.RemoveSonde(filename)
		}
	}

	return nil
}
