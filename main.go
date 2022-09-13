package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

const Version = "0.0.1"

// listen signals from the OS
func listenSignals(worker *Worker) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGUSR1, syscall.SIGUSR2)
	for {
		signal := <-c
		switch signal {
		case syscall.SIGUSR1:
			// Pas utile pour le moment
		case syscall.SIGUSR2:
			writeGoroutineStacks(os.Stdout)
		}
	}
}

// from https://golang.org/src/runtime/pprof/pprof.go
func writeGoroutineStacks(w io.Writer) error {
	fmt.Fprintf(w, "-- Goroutines:\n")

	// We don't know how big the buffer needs to be to collect
	// all the goroutines. Start with 1 MB and try a few times, doubling each time.
	// Give up and use a truncated trace if 64 MB is not enough.
	buf := make([]byte, 1<<20)
	for i := 0; ; i++ {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			buf = buf[:n]
			break
		}
		if len(buf) >= 64<<20 {
			// Filled 64 MB - stop there.
			break
		}
		buf = make([]byte, 2*len(buf))
	}
	_, err := w.Write(buf)
	return err
}

func main() {
	dirSondes := flag.String("d", "", "Directory with sondes")
	version := flag.Bool("v", false, "Print version")
	oldNoseeSondesDirectory := flag.String("c", "", "Duplicate old nosee sondes - abs path")
	destDir := flag.String("o", "", "Destination directory for new toml files - abs path")

	flag.Parse()

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
	err := worker.InitialLoadSondes()
	if err != nil {
		panic(err)
	}

	// On parrallélise l'observation du répertoire
	go worker.ObserveSondeDir()

	// On parrallélise l'écoute des signaux
	go listenSignals(worker)

	// On commence a observer les sondes
	worker.Run()

}
