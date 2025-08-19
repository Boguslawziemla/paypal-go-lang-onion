package http

import (
	"context"
	"fmt"
	"os"
	"paypal-proxy/internal/domain/interfaces"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Logger implements the Logger interface using logrus with enhanced features
type Logger struct {
	logger      *logrus.Logger
	serviceName string
	version     string
	environment string
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level       string
	Format      string // "json" or "text"
	ServiceName string
	Version     string
	Environment string
	Output      string // "stdout", "stderr", or file path
}

// NewLogger creates a new logger with enhanced configuration
func NewLogger(config LoggerConfig) interfaces.Logger {
	logger := logrus.New()
	
	// Set output
	switch config.Output {
	case "stderr":
		logger.SetOutput(os.Stderr)
	case "stdout", "":
		logger.SetOutput(os.Stdout)
	default:
		// File output
		if file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
			logger.SetOutput(file)
		} else {
			logger.SetOutput(os.Stdout)
			logger.WithError(err).Warn("Failed to open log file, using stdout")
		}
	}
	
	// Set formatter
	if config.Format == "text" {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	} else {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
		})
	}
	
	// Set level
	switch strings.ToLower(config.Level) {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info", "":
		logger.SetLevel(logrus.InfoLevel)
	case "warn", "warning":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	case "fatal":
		logger.SetLevel(logrus.FatalLevel)
	case "panic":
		logger.SetLevel(logrus.PanicLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}
	
	// Set report caller for better debugging
	logger.SetReportCaller(true)
	
	return &Logger{
		logger:      logger,
		serviceName: config.ServiceName,
		version:     config.Version,
		environment: config.Environment,
	}
}

// NewDefaultLogger creates a logger with default configuration
func NewDefaultLogger(level string) interfaces.Logger {
	return NewLogger(LoggerConfig{
		Level:       level,
		Format:      "json",
		ServiceName: "paypal-proxy",
		Version:     "1.0.0",
		Environment: "development",
		Output:      "stdout",
	})
}

// Debug logs a debug message with enhanced context
func (l *Logger) Debug(message string, fields map[string]interface{}) {
	entry := l.logger.WithFields(l.addContextFields(fields))
	entry.Debug(message)
}

// Info logs an info message with enhanced context
func (l *Logger) Info(message string, fields map[string]interface{}) {
	entry := l.logger.WithFields(l.addContextFields(fields))
	entry.Info(message)
}

// Warn logs a warning message with enhanced context
func (l *Logger) Warn(message string, fields map[string]interface{}) {
	entry := l.logger.WithFields(l.addContextFields(fields))
	entry.Warn(message)
}

// Error logs an error message with enhanced context and stack trace
func (l *Logger) Error(message string, err error, fields map[string]interface{}) {
	entry := l.logger.WithFields(l.addContextFields(fields))
	if err != nil {
		entry = entry.WithError(err)
		// Add stack trace for errors
		if l.logger.GetLevel() <= logrus.DebugLevel {
			entry = entry.WithField("stack_trace", l.getStackTrace())
		}
	}
	entry.Error(message)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(message string, err error, fields map[string]interface{}) {
	entry := l.logger.WithFields(l.addContextFields(fields))
	if err != nil {
		entry = entry.WithError(err)
	}
	entry.Fatal(message)
}

// addContextFields adds service context to log fields
func (l *Logger) addContextFields(fields map[string]interface{}) logrus.Fields {
	enhancedFields := logrus.Fields{}
	
	// Add service context
	if l.serviceName != "" {
		enhancedFields["service"] = l.serviceName
	}
	if l.version != "" {
		enhancedFields["version"] = l.version
	}
	if l.environment != "" {
		enhancedFields["environment"] = l.environment
	}
	
	// Add caller information for debugging
	if l.logger.GetLevel() <= logrus.DebugLevel {
		if pc, file, line, ok := runtime.Caller(2); ok {
			if fn := runtime.FuncForPC(pc); fn != nil {
				enhancedFields["caller"] = fmt.Sprintf("%s:%d", file, line)
				enhancedFields["function"] = fn.Name()
			}
		}
	}
	
	// Add user fields
	for k, v := range fields {
		enhancedFields[k] = v
	}
	
	return enhancedFields
}

// getStackTrace returns a stack trace string
func (l *Logger) getStackTrace() string {
	buf := make([]byte, 1024)
	for {
		n := runtime.Stack(buf, false)
		if n < len(buf) {
			return string(buf[:n])
		}
		buf = make([]byte, 2*len(buf))
	}
}

// WithContext creates a logger with context information
func (l *Logger) WithContext(ctx context.Context) *logrus.Entry {
	entry := l.logger.WithFields(l.addContextFields(nil))
	
	// Add context values if available
	if traceID := ctx.Value("trace_id"); traceID != nil {
		entry = entry.WithField("trace_id", traceID)
	}
	if requestID := ctx.Value("request_id"); requestID != nil {
		entry = entry.WithField("request_id", requestID)
	}
	
	return entry
}

// WithFields creates a logger entry with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *logrus.Entry {
	return l.logger.WithFields(l.addContextFields(fields))
}

// SetLevel dynamically sets the log level
func (l *Logger) SetLevel(level string) {
	switch strings.ToLower(level) {
	case "debug":
		l.logger.SetLevel(logrus.DebugLevel)
	case "info":
		l.logger.SetLevel(logrus.InfoLevel)
	case "warn", "warning":
		l.logger.SetLevel(logrus.WarnLevel)
	case "error":
		l.logger.SetLevel(logrus.ErrorLevel)
	case "fatal":
		l.logger.SetLevel(logrus.FatalLevel)
	case "panic":
		l.logger.SetLevel(logrus.PanicLevel)
	}
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() string {
	return l.logger.GetLevel().String()
}