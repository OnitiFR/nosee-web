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
	FileName          string
	Name              string
	Url               string
	Search            string
	Timeout           Duration
	WarnTime          Duration
	Delay             Duration
	Index             bool
	LastHttpCode      int
	LastResponseDelay time.Duration
	NextExecution     time.Time
	Errors            map[SondeErrorStatus]*SondeError
	warnLimit         int // number of warning before alert, set by worker
}

func (sonde *Sonde) checkServError(err error) {
	// error on http request
	if err != nil {
		sonde.DeclareError(ErrServError, ErrLvlcritical, err.Error(), "http request error")
	} else {
		sonde.DeclareErrorResolved(ErrServError)
	}
}

func (sonde *Sonde) checkHttpResponseCode(res http.Response) error {
	// http code is not 200
	if res.StatusCode != 200 {
		sonde.DeclareError(ErrServErrorHTTP, ErrLvlcritical, fmt.Sprintf("response code : %d", res.StatusCode), fmt.Sprintf("response code : %d", res.StatusCode))

		return errors.New("response code not 200")
	}

	sonde.DeclareErrorResolved(ErrServErrorHTTP)
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
func (sonde *Sonde) CheckAll() {

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

	sonde.checkServError(err)

	// no body
	if err != nil {
		return
	}

	defer res.Body.Close()

	sonde.LastHttpCode = res.StatusCode

	err = sonde.checkHttpResponseCode(*res)
	if err != nil {
		return
	}
	sonde.checkHttpResponseTime()
	sonde.checkContentAndIndex(*res)
}

func (sonde *Sonde) DeclareError(err SondeErrorStatus, lvl SondeErrorLevel, msg string, subject string) {
	if sonde.Errors[err] == nil {
		sonde.Errors[err] = NewSondeError(err, lvl, msg, subject, time.Now())
	} else {
		sonde.Errors[err].NbTimeErrors++
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
	// err is Critical or Warning with 2 consecutive errors
	can_notify := err.ErrLvl == ErrLvlcritical || (err.ErrLvl == ErrLvlwarning && err.NbTimeErrors >= sonde.warnLimit)
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

func (sonde *Sonde) Update(s *Sonde) bool {
	hasDifferances := sonde.Name != s.Name ||
		sonde.Url != s.Url ||
		sonde.Search != s.Search ||
		sonde.Delay != s.Delay ||
		sonde.Index != s.Index ||
		sonde.Timeout != s.Timeout ||
		sonde.WarnTime != s.WarnTime

	if hasDifferances {
		sonde.Name = s.Name
		sonde.Url = s.Url
		sonde.Search = s.Search
		sonde.Timeout = s.Timeout
		sonde.Delay = s.Delay
		sonde.Index = s.Index
		sonde.WarnTime = s.WarnTime
	}

	return hasDifferances
}
