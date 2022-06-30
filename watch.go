package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Watch struct {
	Sondes []Sonde
	wg     sync.WaitGroup
}

func NewWatch(sondes []Sonde) *Watch {
	return &Watch{
		Sondes: sondes,
	}
}

func watchSondes(watch *Watch) {
	for _, sonde := range watch.Sondes {
		watch.wg.Add(1)
		go watchSonde(watch, sonde)
	}
	watch.wg.Wait()
}

func watchSonde(watch *Watch, sonde Sonde) (s Sonde, err error) {
	defer watch.wg.Done()

	fmt.Printf("Checking %s\n", sonde.Name)

	client := &http.Client{
		Timeout: time.Duration(sonde.Timeout) * time.Millisecond,
	}

	res, err_ := client.Get(sonde.Url)

	if err_ != nil {
		sonde.LastStatus = ErrServError
		sonde.LastError = err_.Error()
		sonde.LastErrorTime = time.Now().Unix()
		alertSonde(sonde)
		return sonde, err_
	}

	defer res.Body.Close()

	body, errRead := ioutil.ReadAll(res.Body)
	if errRead != nil {
		sonde.LastStatus = ErrServError
		sonde.LastError = errRead.Error()
		sonde.LastErrorTime = time.Now().Unix()
		alertSonde(sonde)
		return sonde, errRead
	}

	// check if sonde.Search is in resp.Body
	if !strings.Contains(string(body), sonde.Search) {
		sonde.LastStatus = ErrNoOccurence
		sonde.LastError = "No occurence for : " + sonde.Search
		sonde.LastErrorTime = time.Now().Unix()
		alertSonde(sonde)
		return sonde, nil
	}

	sonde.LastStatus = ErrNone
	sonde.LastError = ""
	sonde.LastErrorTime = 0
	fmt.Println("OK pour " + sonde.Name)
	return sonde, err
}

func alertSonde(sonde Sonde) {
	fmt.Printf("Alert for %s, Error %s", sonde.Name, sonde.LastError)
}
