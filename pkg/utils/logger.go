package utils

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// NewLogger creates a new configured logger instance
func NewLogger() *logrus.Logger {
	logger := logrus.New()

	// Set output to stdout
	logger.SetOutput(os.Stdout)

	// Set log level from environment or default to Info
	level := getLogLevel()
	logger.SetLevel(level)

	// Set formatter based on environment
	if isJSONFormat() {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			DisableColors:   !isColorEnabled(),
		})
	}

	return logger
}

// getLogLevel determines log level from environment
func getLogLevel() logrus.Level {
	levelStr := strings.ToLower(os.Getenv("CLOUDVIEW_LOG_LEVEL"))

	switch levelStr {
	case "trace":
		return logrus.TraceLevel
	case "debug":
		return logrus.DebugLevel
	case "info":
		return logrus.InfoLevel
	case "warn", "warning":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	case "fatal":
		return logrus.FatalLevel
	case "panic":
		return logrus.PanicLevel
	default:
		return logrus.InfoLevel
	}
}

// isJSONFormat checks if JSON log format is requested
func isJSONFormat() bool {
	format := strings.ToLower(os.Getenv("CLOUDVIEW_LOG_FORMAT"))
	return format == "json"
}

// isColorEnabled checks if colored output is enabled
func isColorEnabled() bool {
	// Disable colors if NO_COLOR is set or if not a TTY
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check if explicitly disabled
	if strings.ToLower(os.Getenv("CLOUDVIEW_LOG_COLOR")) == "false" {
		return false
	}

	// Default to enabled for TTY
	return true
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level  string `yaml:"level" json:"level"`
	Format string `yaml:"format" json:"format"`
	Color  bool   `yaml:"color" json:"color"`
	File   string `yaml:"file" json:"file"`
}

// NewLoggerWithConfig creates a logger with specific configuration
func NewLoggerWithConfig(config LoggerConfig) *logrus.Logger {
	logger := logrus.New()

	// Set log level
	if level, err := logrus.ParseLevel(config.Level); err == nil {
		logger.SetLevel(level)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	// Set output
	if config.File != "" {
		if file, err := os.OpenFile(config.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
			logger.SetOutput(file)
		}
	}

	// Set formatter
	if config.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			DisableColors:   !config.Color,
		})
	}

	return logger
}

// WithField creates a logger with a field
func WithField(logger *logrus.Logger, key string, value interface{}) *logrus.Entry {
	return logger.WithField(key, value)
}

// WithFields creates a logger with multiple fields
func WithFields(logger *logrus.Logger, fields map[string]interface{}) *logrus.Entry {
	return logger.WithFields(fields)
}

// WithError creates a logger with an error field
func WithError(logger *logrus.Logger, err error) *logrus.Entry {
	return logger.WithError(err)
}
