package main

import (
	"flag"
	"fmt"
)

const Version = "0.0.1"

func main() {
	dirSondes := flag.String("d", "", "Directory with sondes")
	version := flag.Bool("v", false, "Print version")

	flag.Parse()

	if *version {
		fmt.Println(Version)
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
