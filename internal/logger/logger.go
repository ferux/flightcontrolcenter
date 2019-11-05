package logger

import (
	"log"
	"os"
)

// Logger implements logger for yandex client.
type Logger struct {
	debu *log.Logger
	info *log.Logger
	erro *log.Logger
}

func New() Logger {
	return Logger{
		debu: log.New(os.Stdout, "[DEBU] ", log.Lmicroseconds),
		info: log.New(os.Stdout, "[INFO] ", log.Lmicroseconds),
		erro: log.New(os.Stdout, "[ERRO] ", log.Lmicroseconds),
	}
}

func (l Logger) Debug(msg string)                          { l.debu.Print(msg) }
func (l Logger) Debugf(format string, args ...interface{}) { l.debu.Printf(format, args...) }

func (l Logger) Info(msg string)                          { l.info.Print(msg) }
func (l Logger) Infof(format string, args ...interface{}) { l.info.Printf(format, args...) }

func (l Logger) Err(msg string)                          { l.erro.Print(msg) }
func (l Logger) Errf(format string, args ...interface{}) { l.erro.Printf(format, args...) }
