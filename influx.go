package main

import (
	"bytes"
	"fmt"
	"net/http"
)

func LogToNoseeInfluxDB(host string, measurement string, val float64) error {

	url := "http://localhost:8086/write?db=nosee"

	payload := fmt.Sprintf("%s,host=%s value=%f", measurement, host, val)

	fmt.Printf("payload %s\n", payload)

	res, err := http.Post(url, "text/plain", bytes.NewBuffer([]byte(payload)))

	if res != nil {
		defer res.Body.Close()
	}

	return err
}
