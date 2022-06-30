package main

import (
	"io/ioutil"
	"os"

	"github.com/BurntSushi/toml"
)

func loadSondes(dirSondes string) ([]Sonde, error) {
	if _, err := os.Stat(dirSondes); err != nil {
		return nil, err
	}

	files, err := ioutil.ReadDir(dirSondes)
	if err != nil {
		return nil, err
	}

	sondes := make([]Sonde, 0)
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		sonde, err := loadSonde(dirSondes + "/" + file.Name())
		if err != nil {
			return nil, err
		}

		sondes = append(sondes, sonde)
	}

	return sondes, nil
}

func loadSonde(fileSonde string) (Sonde, error) {
	var sonde Sonde
	_, err := toml.DecodeFile(fileSonde, &sonde)

	if err != nil {
		return sonde, err
	}

	sonde.FileName = fileSonde

	return sonde, err
}
