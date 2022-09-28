package main

import (
	"log"
	"os"
	"sync"
	"time"
)

type Worker struct {
	sondes            map[string]*Sonde
	errors            map[string]*SondeError
	dirSondes         string
	mutex             *sync.Mutex
	WarnLimit         int
	NotifySondeUpdate bool
}

func (w *Worker) CheckRequiredEnv() {
	requieredEnv := []string{
		"SONDE_SLACK_WEBHOOK_URL",
		"SONDE_NOSEE_URL",
		"SONDE_NOSEE_INFLUXDB_URL",
	}
	missingEnv := []string{}
	for _, env := range requieredEnv {
		if os.Getenv(env) == "" {
			missingEnv = append(missingEnv, env)
		}
	}
	if len(missingEnv) > 0 {
		log.Fatalf("Missing required env vars: %s", missingEnv)
	}
}

/*
* Run all checks
 */
func (w *Worker) RunAllCheck() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	sondesToCheck := []*Sonde{}

	for _, sonde := range w.sondes {
		if time.Now().After(sonde.NextExecution) {
			sondesToCheck = append(sondesToCheck, sonde)
		}
	}
	if len(sondesToCheck) > 0 {
		// add x seconds between each check
		intervalBetweenChecks := 30 / len(sondesToCheck)
		for _, sonde := range sondesToCheck {
			time.Sleep(time.Second * time.Duration(intervalBetweenChecks))
			go sonde.CheckAll()
		}
	}
}

/*
* Enter point of the worker
 */
func (w *Worker) Run() error {
	for {
		w.RunAllCheck()
		time.Sleep(time.Minute * 1)
	}
}

func NewWorker(dirSondes string) *Worker {
	return &Worker{
		dirSondes:         dirSondes,
		mutex:             &sync.Mutex{},
		sondes:            make(map[string]*Sonde),
		errors:            make(map[string]*SondeError),
		NotifySondeUpdate: false,
	}
}
