package main

import (
	"bufio"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
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
	LastResponseDelay int
	LastStatus        Errors
	LastError         string
	LastErrorTime     time.Time
	NextExecution     time.Time
}

/**
 * Check if everything is OK
 */
func (sonde *Sonde) Check(ch chan *Sonde) {
	sonde.NextExecution = time.Now().Add(sonde.DelayMinute * time.Minute)

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
	}

	defer res.Body.Close()

	responseTime := time.Since(start).Seconds()

	sonde.LastResponseDelay = int(responseTime)
	sonde.LastHttpCode = res.StatusCode

	// Code HTTP invalide
	if res.StatusCode != 200 {
		sonde.LastStatus = ErrServError
		sonde.LastError = fmt.Sprintf("Reponse code : %d", res.StatusCode)
		sonde.LastErrorTime = time.Now()
		ch <- sonde
	}

	// Hors délai attendu pour la réponse
	if responseTime > float64(sonde.Timeout) {
		sonde.LastStatus = ErrDelay
		sonde.LastError = fmt.Sprintf("Reponse duration too hight %ds vs %fs", sonde.Timeout, responseTime)
		sonde.LastErrorTime = time.Now()
		ch <- sonde
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
		ch <- sonde
	}
	if hasNoIndex && sonde.Index {
		sonde.LastStatus = ErrNoIndex
		sonde.LastError = "No index found"
		sonde.LastErrorTime = time.Now()
		ch <- sonde
	}

	if !sonde.Index && !hasNoIndex {
		sonde.LastStatus = ErrNoIndex
		sonde.LastError = "Index found but not expected"
		sonde.LastErrorTime = time.Now()
		ch <- sonde
	}

	sonde.LastStatus = ErrNone
	sonde.LastError = ""
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

func (sonde *Sonde) DisplayInformations(wasOnError bool) {
	fmt.Println("----------------------------------------------------")
	fmt.Printf("Name : %s\n", sonde.Name)
	fmt.Printf("Was On error : %s\n", strconv.FormatBool(wasOnError))
	fmt.Printf("Url : %s\n", sonde.Url)
	fmt.Printf("Search : %s\n", sonde.Search)
	fmt.Printf("Timeout : %d\n", sonde.Timeout)
	fmt.Printf("DelayMinute : %s\n", sonde.DelayMinute)
	fmt.Printf("Index : %t\n", sonde.Index)
	fmt.Printf("LastHttpCode : %d\n", sonde.LastHttpCode)
	fmt.Printf("LastResponseDelay : %d\n", sonde.LastResponseDelay)
	fmt.Printf("LastStatus : %d\n", sonde.LastStatus)
	fmt.Printf("LastError : %s\n", sonde.LastError)
	fmt.Printf("LastErrorTime : %s\n", sonde.LastErrorTime)
	fmt.Printf("NextExecution : %s\n", sonde.NextExecution)
	fmt.Println("----------------------------------------------------")
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
