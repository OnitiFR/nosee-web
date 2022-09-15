package main

import (
	"fmt"
	"time"
)

type SondeErrorStatus string

const (
	ErrNone             SondeErrorStatus = "none"
	ErrNoOccurence                       = "no occurence"
	ErrServError                         = "server error"
	ErrDelay                             = "delay"
	ErrNoIndex                           = "no index"
	ErrIndexNotExpected                  = "index but not expected"
)

type SondeErrorLevel string

const (
	ErrLvlwarnning = "warnning"
	ErrLvlcritical = "critical"
)

type SondeError struct {
	Status          SondeErrorStatus
	ErrLvl          SondeErrorLevel
	Error           string
	OnErrorSince    time.Time
	CheckInteration int
	Solved          bool
	HasBeenNotified bool
}

func (s *SondeError) IsResolved() bool {
	return s.Solved
}

func (s *SondeError) SetResolved() {
	s.Solved = true
}

func (s *SondeError) SetNotified() {
	s.HasBeenNotified = true
}

func (s *SondeError) GetMessage(sonde *Sonde) string {
	message := fmt.Sprintf("[BAD] %s : %s (web %s) \n", sonde.Name, s.Error, sonde.Url)
	if s.IsResolved() {
		message = fmt.Sprintf("[GOOD] %s : %s (web %s) error duration : %fm\n", sonde.Name, s.Error, sonde.Url, time.Since(s.OnErrorSince).Minutes())
	}

	return message
}

func NewSondeError(Status SondeErrorStatus, ErrLvl SondeErrorLevel, Error string, OnErrorSince time.Time, CheckInteration int) *SondeError {
	return &SondeError{
		Status:          Status,
		ErrLvl:          ErrLvl,
		Error:           Error,
		OnErrorSince:    OnErrorSince,
		CheckInteration: CheckInteration,
		Solved:          false,
	}
}
