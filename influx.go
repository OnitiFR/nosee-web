package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"time"
)

func LogToNoseeInfluxDB(host string, measurement string, val time.Duration) error {

	url := os.Getenv("SONDE_NOSEE_INFLUXDB_URL")

	payload := fmt.Sprintf("%s,host=%s value=%f", measurement, host, val.Seconds())

	res, err := http.Post(url, "text/plain", bytes.NewBuffer([]byte(payload)))

	if res != nil {
		defer res.Body.Close()
	}

	return err
}
