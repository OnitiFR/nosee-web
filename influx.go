package main

func LogToNoseeInfluxDB(host string, probe string, key string, val float64) error {

	return nil

	// url := "http://localhost:8086/write"
	// payload := fmt.Sprintf("db=nosee,host=%s,probe=%s value=%f", host, probe, val)

	// _, err := http.Post(url, "application/x-www-form-urlencoded", strings.NewReader(payload))

	// return err
}
