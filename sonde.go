package main

type Errors int64

const (
	ErrNone        Errors = iota
	ErrNoOccurence        = 1
	ErrServError          = 2
)

type Sonde struct {
	Name string
	Url  string

	FileName string

	Search string

	Timeout int

	LastStatus    Errors
	LastError     string
	LastErrorTime int64
}
