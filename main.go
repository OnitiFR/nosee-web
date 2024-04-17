package main

import (
	"flag"
	"fmt"
)

const Version = "1.0.0"

func main() {
	dirSondes := flag.String("d", "", "Directory with sondes")
	version := flag.Bool("v", false, "Print version")
	test := flag.Bool("t", false, "Test mode - execute test part only")
	debug := flag.Bool("debug", false, "Debug mode")

	flag.Parse()
	debugMode := false

	if *test {
		fmt.Println("Test mode")
		OnitiMainEntry()
		return
	}

	if *version {
		fmt.Println(Version)
		return
	}

	if *dirSondes == "" {
		flag.PrintDefaults()
		return
	}

	if *debug {
		fmt.Println("Debug mode")
		debugMode = true
	}

	worker := NewWorker(*dirSondes, debugMode)

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
