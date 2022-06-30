package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
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

	// every minutes check sondes
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for {
		select {
		case <-quit:
			fmt.Println("Quit")
			return
		case <-time.After(time.Minute):
			sondes, err := loadSondes(*dirSondes)
			if err != nil {
				panic(err)
			}

			watch := NewWatch(sondes)
			watchSondes(watch)
		}
	}

}
