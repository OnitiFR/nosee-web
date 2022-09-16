package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
)

func LogToNoseeInfluxDB(host string, measurement string, val float64) error {

	url := os.Getenv("SONDE_NOSEE_INFLUXDB_URL")

	payload := fmt.Sprintf("%s,host=%s value=%f", measurement, host, val)

	res, err := http.Post(url, "text/plain", bytes.NewBuffer([]byte(payload)))

	if res != nil {
		defer res.Body.Close()
	}

	return err
}
