package main

import (
	"bufio"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

type Duration struct {
	time.Duration
}

// UnmarshalText is needed to satisfy the encoding.TextUnmarshaler interface
func (d *Duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

type Sonde struct {
	FileName          string
	Name              string
	Url               string
	Search            string
	Timeout           Duration
	WarnTime          Duration
	Delay             Duration
	Index             bool
	LastHttpCode      int
	LastResponseDelay float64
	NextExecution     time.Time
	Errors            []*SondeError
}

/**
 * Check if everything is OK
 */
func (sonde *Sonde) Check(chSonde chan *Sonde) {
	sonde.NextExecution = time.Now().Add(sonde.Delay.Duration)

	var sondeErrors []*SondeError

	client := &http.Client{
		Timeout: sonde.Timeout.Duration,
	}

	start := time.Now()

	res, err_ := client.Get(sonde.Url)

	// Erreur lors de l'appel au serveur
	if err_ != nil {
		sondeErrorSrv := NewSondeError(sonde.FileName, ErrServError, ErrLvlcritical, err_.Error(), time.Now())
		sondeErrors = append(sondeErrors, sondeErrorSrv)
	}

	// Si le serveur n'a pas répondu
	if res == nil {
		return
	}

	defer res.Body.Close()

	responseTime := time.Since(start).Seconds()

	sonde.LastResponseDelay = responseTime
	sonde.LastHttpCode = res.StatusCode

	// Code HTTP invalide
	if res.StatusCode != 200 {
		sondeErrorStatus := NewSondeError(sonde.FileName, ErrServError, ErrLvlcritical, fmt.Sprintf("Reponse code : %d", res.StatusCode), time.Now())
		sondeErrors = append(sondeErrors, sondeErrorStatus)
	}

	// Hors délai attendu pour la réponse
	if responseTime > sonde.WarnTime.Duration.Seconds() {
		sondeErrorResponse := NewSondeError(sonde.FileName, ErrDelay, ErrLvlwarnning, fmt.Sprintf("Reponse duration too hight %fs vs %fs", sonde.WarnTime.Duration.Seconds(), responseTime), time.Now())
		sondeErrors = append(sondeErrors, sondeErrorResponse)
	}

	// Vérification de la présence du texte dans la réponse
	// et de la présence ou non de la balise noindex
	reader := bufio.NewReader(res.Body)
	hasSearchContent := false
	hasNoIndex := false
	hasFoundCloseHead := false

	var validNoIndex = regexp.MustCompile(`\<meta[ ]+name=["|']robots["|'][ ]+content=["|'].*noindex.*["|']`)
	var validCloseHead = regexp.MustCompile(`\<\/head\>`)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		if strings.Contains(line, sonde.Search) {
			hasSearchContent = true
		}

		if validCloseHead.MatchString(line) {
			hasFoundCloseHead = true
		}

		if validNoIndex.MatchString(line) {
			hasNoIndex = true
		}

		if (hasNoIndex || hasFoundCloseHead) && hasSearchContent {
			break
		}
	}
	if !hasSearchContent {
		sondeErrorSearch := NewSondeError(sonde.FileName, ErrNoOccurence, ErrLvlcritical, fmt.Sprintf("No occurence : %s ", sonde.Search), time.Now())
		sondeErrors = append(sondeErrors, sondeErrorSearch)
	}
	if hasNoIndex && sonde.Index {
		sondeErrorNoIndex := NewSondeError(sonde.FileName, ErrNoIndex, ErrLvlwarnning, "No index found", time.Now())
		sondeErrors = append(sondeErrors, sondeErrorNoIndex)
	}

	if !sonde.Index && !hasNoIndex {
		sondeErrorNoIndexExpected := NewSondeError(sonde.FileName, ErrNoIndex, ErrLvlwarnning, "Index found but not expected", time.Now())
		sondeErrors = append(sondeErrors, sondeErrorNoIndexExpected)
	}

	sonde.Errors = sondeErrors
	chSonde <- sonde
}

func (sonde *Sonde) Update(s *Sonde) {
	sonde.Name = s.Name
	sonde.Url = s.Url
	sonde.Search = s.Search
	sonde.Timeout = s.Timeout
	sonde.Delay = s.Delay
	sonde.Index = s.Index
	sonde.LastHttpCode = s.LastHttpCode
	sonde.LastResponseDelay = s.LastResponseDelay
	sonde.NextExecution = s.NextExecution
}

/**
 * Load a sonde from a file
 */
func LoadFromToml(fileSonde string) (*Sonde, error) {
	fmt.Printf("Loading sonde from %s\n", fileSonde)
	var sonde *Sonde
	_, err := toml.DecodeFile(fileSonde, &sonde)

	if err != nil {
		return sonde, err
	}

	sonde.FileName = fileSonde
	sonde.NextExecution = time.Now()

	return sonde, err
}
