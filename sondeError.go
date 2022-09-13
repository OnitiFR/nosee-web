package main

import (
	"fmt"
	"os"
	"time"
)

type SondeErrorStatus string

const (
	ErrNone        SondeErrorStatus = "none"
	ErrNoOccurence                  = "no occurence"
	ErrServError                    = "server error"
	ErrDelay                        = "delay"
	ErrNoIndex                      = "no index"
)

type SondeErrorLevel string

const (
	ErrLvlwarnning = "warnning"
	ErrLvlcritical = "critical"
)

type SondeError struct {
	FileName     string
	Status       SondeErrorStatus
	ErrLvl       SondeErrorLevel
	Error        string
	OnErrorSince time.Time
	Hash         string
}

/**
Display error Message
*/
func (s *SondeError) DisplayNewError(sonde *Sonde) {
	file := getLogFile()
	defer file.Close()

	fmt.Fprintf(file, "[%s] [BAD] %s : %s (web %s) \n", time.Now().Format("2006-01-02 15:04:05"), sonde.Name, s.Error, sonde.Url)
}

/**
Display error Message
*/
func (s *SondeError) DisplayResolvedError(sonde *Sonde) {
	file := getLogFile()
	defer file.Close()

	fmt.Fprintf(file, "[%s] [GOOD] %s : %s (web %s) error duration : %fm\n", time.Now().Format("2006-01-02 15:04:05"), sonde.Name, s.Error, sonde.Url, time.Since(s.OnErrorSince).Minutes())
}

func (s *SondeError) IsErrorSolved(errors map[string]*SondeError) bool {
	_, ok := errors[s.Error]
	return ok
}

func getLogFile() *os.File {
	file, err := os.OpenFile("sondes.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("Error opening file:", err)
		panic(err)
	}
	return file
}

func NewSondeError(FileName string, Status SondeErrorStatus, ErrLvl SondeErrorLevel, Error string, OnErrorSince time.Time) *SondeError {
	return &SondeError{
		FileName:     FileName,
		Status:       Status,
		ErrLvl:       ErrLvl,
		Error:        Error,
		OnErrorSince: OnErrorSince,
		Hash:         FileName + Error,
	}
}
