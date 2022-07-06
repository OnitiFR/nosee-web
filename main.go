package main

import (
	"flag"
	"fmt"
)

const Version = "0.0.1"

func main() {
	dirSondes := flag.String("d", "", "Directory with sondes")
	version := flag.Bool("v", false, "Print version")
	oldNoseeSondesDirectory := flag.String("c", "", "Duplicate old nosee sondes - abs path")

	flag.Parse()

	if *version {
		fmt.Println(Version)
		return
	}

	if *oldNoseeSondesDirectory != "" {
		err := DuplicateSondes(*oldNoseeSondesDirectory)
		if err != nil {
			panic(err)
		}
		return
	}

	if *dirSondes == "" {
		flag.PrintDefaults()
		return
	}

	worker := NewWorker(*dirSondes)
	err := worker.InitialLoadSondes()
	if err != nil {
		panic(err)
	}

	// On parrallélise l'observation du répertoire
	go worker.ObserveSondeDir()

	// On commence a observer les sondes
	worker.Run()

}
