package main

import (
	"bufio"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
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
	FileName           string
	Name               string
	Url                string
	Search             string
	Timeout            Duration
	WarnTime           Duration
	Delay              Duration
	Index              bool
	LastHttpCode       int
	LastResponseDelay  float64
	NextExecution      time.Time
	Errors             map[SondeErrorStatus]*SondeError
	CheckInteration    int
	LastCheckDurations []float64 // in seconds only 5 last
}

/**
 * Check if everything is OK
 */
func (sonde *Sonde) CheckAll() {
	start := time.Now()
	sonde.CheckInteration++

	defer sonde.AfterCheck(start)

	sonde.NextExecution = time.Now().Add(sonde.Delay.Duration)

	client := &http.Client{
		Timeout: sonde.Timeout.Duration,
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}

	res, err_ := client.Get(sonde.Url)

	// Erreur lors de l'appel au serveur
	if err_ != nil {
		sonde.DeclareError(ErrServError, ErrLvlcritical, err_.Error())
	} else {
		sonde.DeclareErrorResolved(ErrServError)
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
		sonde.DeclareError(ErrServError, ErrLvlcritical, fmt.Sprintf("Reponse code : %d", res.StatusCode))
	} else {
		sonde.DeclareErrorResolved(ErrServError)
	}

	// Hors délai attendu pour la réponse
	if responseTime > sonde.WarnTime.Duration.Seconds() {
		sonde.DeclareError(ErrDelay, ErrLvlwarnning, fmt.Sprintf("Reponse duration too hight %fs vs %fs", sonde.WarnTime.Duration.Seconds(), responseTime))
	} else {
		sonde.DeclareErrorResolved(ErrDelay)
	}

	// Log to influxdb
	go LogToNoseeInfluxDB(sonde.Url, sonde.FileName, "response_time", responseTime)

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
		sonde.DeclareError(ErrNoOccurence, ErrLvlcritical, fmt.Sprintf("No occurence : %s ", sonde.Search))
	} else {
		sonde.DeclareErrorResolved(ErrNoOccurence)
	}

	if hasNoIndex && sonde.Index {
		sonde.DeclareError(ErrNoIndex, ErrLvlwarnning, "No index found")
	} else {
		sonde.DeclareErrorResolved(ErrNoIndex)
	}

	if !sonde.Index && !hasNoIndex {
		sonde.DeclareError(ErrIndexNotExpected, ErrLvlwarnning, "Index found but not expected")
	} else {
		sonde.DeclareErrorResolved(ErrIndexNotExpected)
	}

}

func (sonde *Sonde) DeclareError(err SondeErrorStatus, lvl SondeErrorLevel, msg string) {
	if sonde.Errors[err] == nil {
		sonde.Errors[err] = NewSondeError(err, lvl, msg, time.Now(), sonde.CheckInteration)
	}
}

func (sonde *Sonde) DeclareErrorResolved(err SondeErrorStatus) {
	if sonde.Errors[err] != nil {
		sonde.Errors[err].SetResolved()
	}
}

func (sonde *Sonde) GetErrors() map[SondeErrorStatus]*SondeError {
	return sonde.Errors
}

func (sonde *Sonde) AfterCheck(start time.Time) {
	sonde.LastCheckDurations = append(sonde.LastCheckDurations, time.Since(start).Seconds())
	if len(sonde.LastCheckDurations) > 5 {
		sonde.LastCheckDurations = sonde.LastCheckDurations[1:]
	}

	keyToDel := []SondeErrorStatus{}
	for key, err := range sonde.Errors {
		if err.IsResolved() {
			keyToDel = append(keyToDel, key)
		}
		// parraralize notifications
		go sonde.Notify(err)
	}

	for _, key := range keyToDel {
		delete(sonde.Errors, key)
	}
}

func (sonde *Sonde) Notify(err *SondeError) {
	// err is Critical or Warnning with 2 consecutive errors
	can_notify := err.ErrLvl == ErrLvlcritical || (err.ErrLvl == ErrLvlwarnning && sonde.CheckInteration-err.CheckInteration >= 2)
	// is not notified or is resolved
	can_notify = can_notify && (!err.HasBeenNotified || err.IsResolved())

	if can_notify {
		// Slack notification
		slackerr := NotifySlack(err.GetMessage(sonde), err.Solved)
		noseeErr := NotifyNoseeConsole(sonde, err)
		if slackerr != nil {
			fmt.Println(slackerr)
		}
		if noseeErr != nil {
			fmt.Println(noseeErr)
		}
		// every notifications is sent
		if slackerr == nil && noseeErr == nil {
			err.SetNotified()
		}

	}
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
