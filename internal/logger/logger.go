package logger

// Logger of the service.
type Logger interface {
	Debug(msg string)
	Debugf(format string, args ...interface{})

	Info(msg string)
	Infof(format string, args ...interface{})

	Err(msg string)
	Errf(format string, args ...interface{})
}
