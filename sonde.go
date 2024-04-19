package main

import (
	"bufio"
	"errors"
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
	FileName             string
	Name                 string
	Url                  string
	Search               string
	Timeout              Duration
	WarnTime             Duration
	Delay                Duration
	Index                bool
	LastHttpCode         int
	LastResponseDelay    time.Duration
	NextExecution        time.Time
	Errors               map[SondeErrorStatus]*SondeError
	NbRetentionsWarning  int
	NbRetentionsCritical int
}

func (sonde *Sonde) checkHttpResponseCode(res http.Response) error {
	// http code is not 200
	if res.StatusCode != 200 {
		sonde.DeclareError(ErrServError, ErrLvlcritical, fmt.Sprintf("response code : %d", res.StatusCode), fmt.Sprintf("response code : %d", res.StatusCode))

		return errors.New("response code not 200")
	}

	sonde.DeclareErrorResolved(ErrServError)
	return nil

}

func (sonde *Sonde) checkHttpResponseTime() {
	// response time is too long
	if sonde.LastResponseDelay > sonde.WarnTime.Duration {
		sonde.DeclareError(ErrDelay, ErrLvlwarning, fmt.Sprintf("response duration too high %s vs %s", sonde.WarnTime.Duration, sonde.LastResponseDelay), "response duration too high")
	} else {
		sonde.DeclareErrorResolved(ErrDelay)
	}

	// Log to influxdb
	go LogToNoseeInfluxDB(sonde.Name, "response_time", sonde.LastResponseDelay)
}

func (sonde *Sonde) checkContentAndIndex(res http.Response) {
	// searching keywords in body
	// checking index page
	scanner := bufio.NewScanner(res.Body)
	hasSearchContent := false
	hasNoIndex := false
	hasFoundCloseHead := false

	var validNoIndex = regexp.MustCompile(`\<meta[ ]+name=["|']robots["|'][ ]+content=["|'].*noindex.*["|']`)
	var validCloseHead = regexp.MustCompile(`\<\/head\>`)
	for scanner.Scan() {
		line := scanner.Text()

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
		sonde.DeclareError(ErrNoOccurence, ErrLvlcritical, fmt.Sprintf("no occurence : %s ", sonde.Search), "no occurence found")
	} else {
		sonde.DeclareErrorResolved(ErrNoOccurence)
	}

	if hasNoIndex && sonde.Index {
		sonde.DeclareError(ErrNoIndex, ErrLvlwarning, "search engine indexing not allowed", "search engine indexing not allowed")
	} else {
		sonde.DeclareErrorResolved(ErrNoIndex)
	}

	if !sonde.Index && !hasNoIndex {
		sonde.DeclareError(ErrIndexNotExpected, ErrLvlwarning, "search engine indexing not expected", "search engine indexing not expected")
	} else {
		sonde.DeclareErrorResolved(ErrIndexNotExpected)
	}
}

/**
 * Check if everything is OK
 */
func (sonde *Sonde) CheckAll(debug bool) {

	if debug {
		fmt.Printf("Checking %s\n", sonde.Name)
	}

	defer sonde.AfterCheck()

	sonde.NextExecution = time.Now().Add(sonde.Delay.Duration)

	client := &http.Client{
		Timeout: sonde.Timeout.Duration,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			MaxConnsPerHost:   1,
		},
	}

	defer client.CloseIdleConnections()

	start := time.Now()

	res, err := client.Get(sonde.Url)
	sonde.LastResponseDelay = time.Since(start)

	// no body
	if err != nil {
		sonde.checkHttpResponseCode(http.Response{StatusCode: 600})
		// Log to influxdb
		sonde.logDefautTimeOutInfluxDB()
		return
	}

	defer res.Body.Close()

	sonde.LastHttpCode = res.StatusCode

	err = sonde.checkHttpResponseCode(*res)
	if err != nil {
		// Log to influxdb
		sonde.logDefautTimeOutInfluxDB()
		return
	}
	sonde.checkHttpResponseTime()
	sonde.checkContentAndIndex(*res)
}

func (sonde *Sonde) logDefautTimeOutInfluxDB() {
	go LogToNoseeInfluxDB(sonde.Name, "response_time", sonde.WarnTime.Duration)
}

func (sonde *Sonde) DeclareError(err SondeErrorStatus, lvl SondeErrorLevel, msg string, subject string) {
	if sonde.Errors[err] == nil {
		nbRetentions := 1
		if lvl == ErrLvlwarning {
			nbRetentions = sonde.NbRetentionsWarning
		} else {
			nbRetentions = sonde.NbRetentionsCritical
		}
		sonde.Errors[err] = NewSondeError(err, lvl, msg, subject, time.Now(), nbRetentions)
	} else {
		sonde.Errors[err].NbTimeErrors++
		sonde.Errors[err].NbTimeSolved = 0
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

func (sonde *Sonde) AfterCheck() {
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
	if err.CanNotify() {
		// Slack notification
		slackerr := NotifySlack(err.GetMessage(sonde), err.IsResolved())
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

func (sonde *Sonde) Update(s *Sonde) bool {
	hasDifferances := sonde.Name != s.Name || sonde.Url != s.Url || sonde.Search != s.Search || sonde.Delay != s.Delay || sonde.Index != s.Index || sonde.Timeout != s.Timeout || sonde.NbRetentionsWarning != s.NbRetentionsWarning || sonde.NbRetentionsCritical != s.NbRetentionsCritical

	if hasDifferances {
		sonde.Name = s.Name
		sonde.Url = s.Url
		sonde.Search = s.Search
		sonde.Timeout = s.Timeout
		sonde.Delay = s.Delay
		sonde.Index = s.Index
		sonde.NbRetentionsCritical = s.NbRetentionsCritical
		sonde.NbRetentionsWarning = s.NbRetentionsWarning
	}

	return hasDifferances
}
