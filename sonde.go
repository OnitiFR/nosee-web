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
	WarnLimit          int
}

func (sonde *Sonde) checkServError(err error) {
	// error on http request
	if err != nil {
		sonde.DeclareError(ErrServError, ErrLvlcritical, err.Error(), "http request error")
	} else {
		sonde.DeclareErrorResolved(ErrServError)
	}
}

func (sonde *Sonde) checkHttpResponseCode(res http.Response) {
	// http code is not 200
	if res.StatusCode != 200 {
		sonde.DeclareError(ErrServError, ErrLvlcritical, fmt.Sprintf("response code : %d", res.StatusCode), fmt.Sprintf("response code : %d", res.StatusCode))
	} else {
		sonde.DeclareErrorResolved(ErrServError)
	}
}

func (sonde *Sonde) checkHttpResponseTime() {
	// response time is too long
	if sonde.LastResponseDelay > sonde.WarnTime.Duration.Seconds() {
		sonde.DeclareError(ErrDelay, ErrLvlwarning, fmt.Sprintf("response duration too high %.2fs vs %.2fs", sonde.WarnTime.Duration.Seconds(), sonde.LastResponseDelay), "response duration too high")
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
	sonde.LastResponseDelay = time.Since(start).Seconds()

	sonde.checkServError(err_)

	// no body
	if err_ != nil {
		return
	}

	defer res.Body.Close()

	sonde.LastHttpCode = res.StatusCode

	sonde.checkHttpResponseCode(*res)
	sonde.checkHttpResponseTime()
	sonde.checkContentAndIndex(*res)
}

func (sonde *Sonde) DeclareError(err SondeErrorStatus, lvl SondeErrorLevel, msg string, subject string) {
	if sonde.Errors[err] == nil {
		sonde.Errors[err] = NewSondeError(err, lvl, msg, subject, time.Now(), sonde.CheckInteration)
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
	// err is Critical or Warning with 2 consecutive errors
	can_notify := err.ErrLvl == ErrLvlcritical || (err.ErrLvl == ErrLvlwarning && sonde.CheckInteration-err.CheckInteration >= sonde.WarnLimit)
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
}
