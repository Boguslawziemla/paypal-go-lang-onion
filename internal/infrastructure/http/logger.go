package http

import (
	"paypal-proxy/internal/domain/interfaces"

	"github.com/sirupsen/logrus"
)

// Logger implements the Logger interface using logrus
type Logger struct {
	logger *logrus.Logger
}

// NewLogger creates a new logger
func NewLogger(level string) interfaces.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	
	switch level {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}
	
	return &Logger{logger: logger}
}

// Debug logs a debug message
func (l *Logger) Debug(message string, fields map[string]interface{}) {
	l.logger.WithFields(logrus.Fields(fields)).Debug(message)
}

// Info logs an info message
func (l *Logger) Info(message string, fields map[string]interface{}) {
	l.logger.WithFields(logrus.Fields(fields)).Info(message)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, fields map[string]interface{}) {
	l.logger.WithFields(logrus.Fields(fields)).Warn(message)
}

// Error logs an error message
func (l *Logger) Error(message string, err error, fields map[string]interface{}) {
	entry := l.logger.WithFields(logrus.Fields(fields))
	if err != nil {
		entry = entry.WithError(err)
	}
	entry.Error(message)
}