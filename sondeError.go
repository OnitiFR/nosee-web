package main

import "time"

type SondeError struct {
	FileName     string
	LastError    string
	OnErrorSince time.Time
}

func NewSondeError(sonde *Sonde) *SondeError {
	return &SondeError{
		FileName:     sonde.FileName,
		LastError:    sonde.LastError,
		OnErrorSince: sonde.OnErrorSince,
	}
}
