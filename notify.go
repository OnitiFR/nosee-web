package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

func NotifySlack(message string, resolved bool) error {
	mark := ":exclamation:"
	if resolved {
		mark = ":heavy_check_mark:"
	}
	hook := os.Getenv("SONDE_SLACK_WEBHOOK_URL")

	final_message := fmt.Sprintf("%s %s", mark, message)

	payload_json := map[string]string{
		"text": final_message,
	}

	payload, err := json.Marshal(payload_json)
	if err != nil {
		return err
	}
	res, errPost := http.Post(hook, "application/json", bytes.NewBuffer(payload))
	if res != nil {
		defer res.Body.Close()
	}
	return errPost

}

func NotifyNoseeConsole(sonde *Sonde, sonde_err *SondeError) error {

	nosee_url := os.Getenv("SONDE_NOSEE_URL")
	payload_type := sonde_err.GetNoseeType()
	subject := sonde_err.GetNoseeSubject(sonde)
	details := sonde_err.GetNoseeDetail(sonde)
	classes := fmt.Sprintf("%s", sonde_err.ErrLvl)
	hostname := sonde.Url
	nosee_srv := "sonde wp - Prod"
	uniqueid := sonde_err.GetUuid()
	datetime := time.Now().Format(time.RFC3339)

	res, err := http.PostForm(nosee_url, url.Values{
		"type":      {payload_type},
		"subject":   {subject},
		"details":   {details},
		"classes":   {classes},
		"hostname":  {hostname},
		"nosee_srv": {nosee_srv},
		"uniqueid":  {uniqueid},
		"datetime":  {datetime},
	})

	if res != nil {
		defer res.Body.Close()
	}

	return err
}
