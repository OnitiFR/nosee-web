package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

// listen signals from the OS
func listenSignals(worker *Worker) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGUSR1, syscall.SIGUSR2)
	for {
		signal := <-c
		switch signal {
		case syscall.SIGUSR1:
			worker.ScanSondeDirectory()
		case syscall.SIGUSR2:
			displayAllSondesStatus(worker)
		case syscall.SIGQUIT:
			writeGoroutineStacks(os.Stdout)
		}
	}
}

func displayAllSondesStatus(worker *Worker) {
	fmt.Println("Liste des sondes surveillées :")
	for _, sonde := range worker.sondes {
		fmt.Printf("-----------------------------------------------------------\n")
		fmt.Printf("%s\n", sonde.Name)
		fmt.Printf("Url : %s\n", sonde.Url)
		fmt.Printf("Search : %s\n", sonde.Search)
		fmt.Printf("Delay : %d\n", sonde.Delay)
		fmt.Printf("Index : %t\n", sonde.Index)
		fmt.Printf("Timeout : %d\n", sonde.Timeout)
		fmt.Printf("WarnTime : %d\n", sonde.WarnTime)
		fmt.Printf("warnLimit : %d\n", sonde.warnLimit)
		fmt.Printf("LastHttpCode : %d\n", sonde.LastHttpCode)
		fmt.Printf("LastResponseDelay : %s\n", sonde.LastResponseDelay)
		fmt.Printf("NextExecution : %s\n", sonde.NextExecution.Format("2006-01-02 15:04:05"))
		fmt.Printf("Actual Errors :\n")
		for _, error := range sonde.Errors {
			fmt.Printf(" %s - %s - nb err: %d : %s\n", error.Status, error.ErrLvl, error.NbTimeErrors, error.GetMessage(sonde))
		}
		fmt.Printf("-----------------------------------------------------------\n")
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
