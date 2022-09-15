package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

func NotifySlack(message string, resolved bool) error {
	mark := ":exclamation:"
	if resolved {
		mark = ":heavy_check_mark:"
	}
	hook := os.Getenv("SONDE_SLACK_WEBHOOK_URL")

	final_message := fmt.Sprintf("%s %s - %s", mark, message, "go-sonde-wp")

	payload := fmt.Sprintf(`{"text": "%s"}`, final_message)

	_, err := http.Post(hook, "application/json", strings.NewReader(payload))
	return err

}

func NotifyNoseeConsole(sonde *Sonde, err *SondeError) error {

	return nil

	// url := os.Getenv("SONDE_NOSEE_URL")
	// payload_type := "" // ?? je sais pas quoi mettre
	// subject := ""      // ?? je sais pas quoi mettre
	// details := err.GetMessage(sonde)
	// classes := "" // ?? je sais pas quoi mettre
	// hostname := sonde.Url
	// nosee_srv := "" // ?? je sais pas quoi mettre
	// uniqueid := ""  // ?? Si resolved je dois renvoyer le même uniqueid que pour le notify initial
	// datetime := time.Now().Format("2006-01-02 15:04:05")

	// // post with content-Type multipart/form-data
	// payload := &bytes.Buffer{}
	// writer := multipart.NewWriter(payload)
	// defer writer.Close()
	// _ = writer.WriteField("payload_type", payload_type)
	// _ = writer.WriteField("subject", subject)
	// _ = writer.WriteField("details", details)
	// _ = writer.WriteField("classes", classes)
	// _ = writer.WriteField("hostname", hostname)
	// _ = writer.WriteField("nosee_srv", nosee_srv)
	// _ = writer.WriteField("uniqueid", uniqueid)
	// _ = writer.WriteField("datetime", datetime)

	// _, errPost := http.Post(url, writer.FormDataContentType(), payload)

	// return errPost
}
