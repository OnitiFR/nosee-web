package main

import (
	"flag"
	"fmt"
)

const Version = "1.0.0"

func main() {
	dirSondes := flag.String("d", "", "Directory with sondes")
	warnLimit := flag.Int("w", 2, "Number of warning before alert")
	version := flag.Bool("v", false, "Print version")
	oldNoseeSondesDirectory := flag.String("c", "", "Duplicate old nosee sondes - abs path")
	destDir := flag.String("o", "", "Destination directory for new toml files - abs path")
	test := flag.Bool("t", false, "Test mode - execute test part only")

	flag.Parse()

	if *test {
		fmt.Println("Test mode")
		OnitiMainEntry()
		return
	}

	if *version {
		fmt.Println(Version)
		return
	}

	if *oldNoseeSondesDirectory != "" {
		if *destDir == "" {
			fmt.Println("Destination directory is required")
			return
		}
		err := DuplicateSondes(*oldNoseeSondesDirectory, *destDir)
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
	worker.WarnLimit = *warnLimit

	//check env
	worker.CheckRequiredEnv()

	err := worker.InitialLoadSondes()
	if err != nil {
		panic(err)
	}

	// parrallelize directory observation
	go worker.ObserveSondeDir()

	// parrallelize signals listening
	go listenSignals(worker)

	// start the worker
	worker.Run()

}
