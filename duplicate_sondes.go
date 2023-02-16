package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

type tomlDefault struct {
	Name  string
	Value int
}

type OldSondeNosee struct {
	Name      string
	Delay     string
	Timeout   string
	Arguments string
	FileName  string
	Default   []tomlDefault
}

type NewSondeToml struct {
	Name     string
	Url      string
	Search   string
	Timeout  string
	WarnTime string
	Delay    string
	Index    bool
}

/*
Create a new toml file from an old nosee sonde
*/
func (s *OldSondeNosee) createTomlFile(destDir string) {

	arguments := strings.Split(s.Arguments, " ")
	search := strings.Trim(strings.Join(arguments[1:], " "), "'")

	var newSonde NewSondeToml
	newSonde.Name = strings.Replace(s.Name, "web ", "", -1)
	newSonde.Url = arguments[0]
	newSonde.Search = search
	newSonde.Delay = s.Delay
	newSonde.Timeout = s.Timeout
	newSonde.Index = true
	for _, tomlDefault := range s.Default {
		switch tomlDefault.Name {
		case "web_warn_time":
			newSonde.WarnTime = fmt.Sprintf("%ds", tomlDefault.Value)
		}
	}
	f, err := os.Create(destDir + "/" + s.FileName)
	if err != nil {
		panic(err)
	}
	toml.NewEncoder(f).Encode(newSonde)
}

/*
* Duplicate old nosee sondes
 */
func DuplicateSondes(oldNoseeSondesDirectory string, destDir string) error {
	if _, err := os.Stat(oldNoseeSondesDirectory); err != nil {
		return err
	}

	files, err := os.ReadDir(oldNoseeSondesDirectory)
	if err != nil {
		return err
	}
	oldSondes := make([]*OldSondeNosee, 0)
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".toml") || !strings.HasPrefix(file.Name(), "web_") {
			continue
		}
		fullPath := oldNoseeSondesDirectory + "/" + file.Name()

		var oldSonde *OldSondeNosee
		_, err := toml.DecodeFile(fullPath, &oldSonde)

		if err != nil {
			return err
		}

		oldSonde.FileName = file.Name()

		oldSondes = append(oldSondes, oldSonde)
	}

	for _, oldSonde := range oldSondes {
		oldSonde.createTomlFile(destDir)
	}

	return nil
}
