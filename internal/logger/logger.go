package logger

import (
	"log"
	"os"
)

// Logger of the service.
type Logger interface {
	Debug(msg string)
	Debugf(format string, args ...interface{})

	Info(msg string)
	Infof(format string, args ...interface{})

	Err(msg string)
	Errf(format string, args ...interface{})
}

type logger struct{}

func New() Logger {
	return logger{}
}

// nolint:gochecknoglobals
var dlogger = log.New(os.Stdout, "[DEBU] ", log.Lmicroseconds)

// nolint:gochecknoglobals
var ilogger = log.New(os.Stdout, "[INFO] ", log.Lmicroseconds)

// nolint:gochecknoglobals
var elogger = log.New(os.Stderr, "[ERRO] ", log.Lmicroseconds)

func (logger) Debug(msg string)                          { dlogger.Print(msg) }
func (logger) Debugf(format string, args ...interface{}) { dlogger.Printf(format, args...) }

func (logger) Info(msg string)                          { ilogger.Print(msg) }
func (logger) Infof(format string, args ...interface{}) { ilogger.Printf(format, args...) }

func (logger) Err(msg string)                          { elogger.Print(msg) }
func (logger) Errf(format string, args ...interface{}) { elogger.Printf(format, args...) }
