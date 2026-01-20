package bridge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel string

const (
	LogLevelInfo    LogLevel = "INFO"
	LogLevelWarn    LogLevel = "WARN"
	LogLevelError   LogLevel = "ERROR"
	LogLevelSuccess LogLevel = "SUCCESS"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// Logger handles structured logging to file and console
type Logger struct {
	logDir  string
	logFile *os.File
}

// NewLogger creates a new logger instance
func NewLogger(logDir string) (*Logger, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("20060102-150405")
	logPath := filepath.Join(logDir, fmt.Sprintf("bridge-%s.log", timestamp))

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	return &Logger{
		logDir:  logDir,
		logFile: logFile,
	}, nil
}

// log writes a log entry to file and console
func (l *Logger) log(level LogLevel, message string, data map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Data:      data,
	}

	// Write to console with color
	l.printConsole(entry)

	// Write to file as JSON
	l.writeToFile(entry)
}

// printConsole prints formatted log to console
func (l *Logger) printConsole(entry LogEntry) {
	var color string
	switch entry.Level {
	case LogLevelInfo:
		color = "\033[36m" // Cyan
	case LogLevelWarn:
		color = "\033[33m" // Yellow
	case LogLevelError:
		color = "\033[31m" // Red
	case LogLevelSuccess:
		color = "\033[32m" // Green
	}
	reset := "\033[0m"

	fmt.Printf("%s[%s]%s [%s] %s",
		color,
		entry.Level,
		reset,
		entry.Timestamp,
		entry.Message,
	)

	if len(entry.Data) > 0 {
		dataJSON, _ := json.Marshal(entry.Data)
		fmt.Printf(" %s", string(dataJSON))
	}

	fmt.Println()
}

// writeToFile writes log entry to file as JSON
func (l *Logger) writeToFile(entry LogEntry) {
	jsonData, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal log entry: %v\n", err)
		return
	}

	if _, err := l.logFile.Write(append(jsonData, '\n')); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write to log file: %v\n", err)
	}
}

// Info logs an informational message
func (l *Logger) Info(message string, data map[string]interface{}) {
	l.log(LogLevelInfo, message, data)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, data map[string]interface{}) {
	l.log(LogLevelWarn, message, data)
}

// Error logs an error message
func (l *Logger) Error(message string, data map[string]interface{}) {
	l.log(LogLevelError, message, data)
}

// Success logs a success message
func (l *Logger) Success(message string, data map[string]interface{}) {
	l.log(LogLevelSuccess, message, data)
}

// Close closes the log file
func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}
