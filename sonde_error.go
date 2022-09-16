package main

import (
	"fmt"
	"time"

	"github.com/google/uuid"
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
	ErrLvlwarning  = "warning"
	ErrLvlcritical = "critical"
)

type SondeError struct {
	uuid            string
	Status          SondeErrorStatus
	ErrLvl          SondeErrorLevel
	Subject         string
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

func (s *SondeError) GetUuid() string {
	return s.uuid
}

func (s *SondeError) GetNoseeType() string {
	if s.IsResolved() {
		return "GOOD"
	}
	return "BAD"
}

func (s *SondeError) GetNoseeSubject(sonde *Sonde) string {
	subject := fmt.Sprintf("[BAD] %s (web %s) \n", s.Subject, sonde.Url)
	if s.IsResolved() {
		subject = fmt.Sprintf("[GOOD] %s (web %s)\n", s.Subject, sonde.Url)
	}

	return subject
}

func (s *SondeError) GetNoseeDetail(sonde *Sonde) string {
	alert_status := "is"
	if s.IsResolved() {
		alert_status = "no more"
	}
	detail := fmt.Sprintf("An alert **%s** ringing. \n\n", alert_status)
	detail += fmt.Sprintf("Failure time: %s\n", s.OnErrorSince.Format("2006-01-02 15:04:05"))
	if s.IsResolved() {
		detail += fmt.Sprintf("Failure time: %s\n", s.OnErrorSince.Format("2006-01-02 15:04:05"))
		detail += fmt.Sprintf("Resolved time: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	} else {
		detail += fmt.Sprintf("Next task time: %s\n", sonde.NextExecution.Format("2006-01-02 15:04:05"))
	}
	detail += fmt.Sprintf("Class(es): %s\n", s.ErrLvl)
	detail += fmt.Sprintf("Error was: %s", s.Error)

	return detail
}

func (s *SondeError) GetMessage(sonde *Sonde) string {
	message := fmt.Sprintf("[BAD] %s : %s (web %s) \n", sonde.Name, s.Subject, sonde.Url)
	if s.IsResolved() {
		message = fmt.Sprintf("[GOOD] %s : %s (web %s) error duration : %.2fm\n", sonde.Name, s.Subject, sonde.Url, time.Since(s.OnErrorSince).Minutes())
	}

	return message
}

func NewSondeError(Status SondeErrorStatus, ErrLvl SondeErrorLevel, Error string, Subject string, OnErrorSince time.Time, CheckInteration int) *SondeError {
	return &SondeError{
		uuid:            uuid.New().String(),
		Status:          Status,
		ErrLvl:          ErrLvl,
		Error:           Error,
		Subject:         Subject,
		OnErrorSince:    OnErrorSince,
		CheckInteration: CheckInteration,
		Solved:          false,
	}
}
