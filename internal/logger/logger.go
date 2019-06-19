package logger

// Logger of the service. 
type Logger interface {
	Debug(msg string)
	Debugf(format string, args ...interface{})
}
