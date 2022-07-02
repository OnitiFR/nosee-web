package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

type Errors int64

const (
	ErrNone        Errors = iota
	ErrNoOccurence        = 1
	ErrServError          = 2
	ErrDelay              = 3
	ErrNoIndex            = 4
)

type Sonde struct {
	FileName          string
	Name              string
	Url               string
	Search            string
	Timeout           int
	DelayMinute       time.Duration
	Index             bool
	LastHttpCode      int
	LastResponseDelay float64
	LastStatus        Errors
	LastError         string
	LastErrorTime     time.Time
	OnErrorSince      time.Time
	NextExecution     time.Time
}

/**
 * Check if everything is OK
 */
func (sonde *Sonde) Check(ch chan *Sonde) {
	sonde.NextExecution = time.Now().Add(sonde.DelayMinute * time.Minute)
	fmt.Printf("Checking %s => time : %s\n", sonde.Url, time.Now().String())

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	start := time.Now()

	res, err_ := client.Get(sonde.Url)

	// Erreur lors de l'appel au serveur
	if err_ != nil {
		sonde.LastStatus = ErrServError
		sonde.LastError = err_.Error()
		sonde.LastErrorTime = time.Now()
		ch <- sonde
		return
	}

	defer res.Body.Close()

	responseTime := time.Since(start).Seconds()

	sonde.LastResponseDelay = responseTime
	sonde.LastHttpCode = res.StatusCode

	// Code HTTP invalide
	if res.StatusCode != 200 {
		sonde.LastStatus = ErrServError
		sonde.LastError = fmt.Sprintf("Reponse code : %d", res.StatusCode)
		sonde.LastErrorTime = time.Now()
		if sonde.OnErrorSince.IsZero() {
			sonde.OnErrorSince = time.Now()
		}
		ch <- sonde
		return
	}

	// Hors délai attendu pour la réponse
	if responseTime > float64(sonde.Timeout) {
		sonde.LastStatus = ErrDelay
		sonde.LastError = fmt.Sprintf("Reponse duration too hight %ds vs %fs", sonde.Timeout, responseTime)
		sonde.LastErrorTime = time.Now()
		if sonde.OnErrorSince.IsZero() {
			sonde.OnErrorSince = time.Now()
		}
		ch <- sonde
		return
	}

	// Vérification de la présence du texte dans la réponse
	// et de la présence ou non de la balise noindex
	reader := bufio.NewReader(res.Body)
	hasSearchContent := false
	hasNoIndex := false

	var validNoIndex = regexp.MustCompile(`\<meta[ ]+name=["|']robots["|'][ ]+content=["|'].*noindex.*["|']`)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		if strings.Contains(line, sonde.Search) {
			hasSearchContent = true
		}
		if validNoIndex.MatchString(line) {
			hasNoIndex = true
		}

		if hasNoIndex && hasSearchContent {
			break
		}
	}
	if !hasSearchContent {
		sonde.LastStatus = ErrNoOccurence
		sonde.LastError = "No occurence for : " + sonde.Search
		sonde.LastErrorTime = time.Now()
		if sonde.OnErrorSince.IsZero() {
			sonde.OnErrorSince = time.Now()
		}
		ch <- sonde
		return
	}
	if hasNoIndex && sonde.Index {
		sonde.LastStatus = ErrNoIndex
		sonde.LastError = "No index found"
		sonde.LastErrorTime = time.Now()
		if sonde.OnErrorSince.IsZero() {
			sonde.OnErrorSince = time.Now()
		}
		ch <- sonde
		return
	}

	if !sonde.Index && !hasNoIndex {
		sonde.LastStatus = ErrNoIndex
		sonde.LastError = "Index found but not expected"
		sonde.LastErrorTime = time.Now()
		if sonde.OnErrorSince.IsZero() {
			sonde.OnErrorSince = time.Now()
		}
		ch <- sonde
		return
	}

	sonde.LastStatus = ErrNone
	sonde.LastError = ""
	sonde.OnErrorSince = time.Time{}

	ch <- sonde
}

func (sonde *Sonde) Update(s *Sonde) {
	sonde.Name = s.Name
	sonde.Url = s.Url
	sonde.Search = s.Search
	sonde.Timeout = s.Timeout
	sonde.DelayMinute = s.DelayMinute
	sonde.Index = s.Index
	sonde.LastHttpCode = s.LastHttpCode
	sonde.LastResponseDelay = s.LastResponseDelay
	sonde.LastStatus = s.LastStatus
	sonde.LastError = s.LastError
	sonde.LastErrorTime = s.LastErrorTime
	sonde.NextExecution = s.NextExecution
}

/**
Display the sonde information
*/
func (sonde *Sonde) DisplayInformations(lasError string, lastErrorTime time.Time) {

	if sonde.LastStatus != ErrNone {
		file, err := os.OpenFile("sondes.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Println("Error opening file:", err)
			return
		}
		defer file.Close()
		fmt.Fprintf(file, "[BAD] nosee: %s (web %s) \n", sonde.LastError, sonde.Url)
	} else if sonde.LastStatus == ErrNone && lasError != "" {
		file, err := os.OpenFile("sondes.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Println("Error opening file:", err)
			return
		}
		defer file.Close()
		fmt.Fprintf(file, "[GOOD] nosee: %s (web %s) error duration : %fm\n", lasError, sonde.Url, time.Since(lastErrorTime).Minutes())
	}
}

/**
 * Load a sonde from a file
 */
func LoadFromToml(fileSonde string) (*Sonde, error) {
	var sonde *Sonde
	_, err := toml.DecodeFile(fileSonde, &sonde)

	if err != nil {
		return sonde, err
	}

	sonde.FileName = fileSonde
	sonde.NextExecution = time.Now()

	return sonde, err
}
