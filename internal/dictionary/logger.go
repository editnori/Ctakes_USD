package dictionary

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	LogDebug LogLevel = iota
	LogInfo
	LogWarning
	LogError
	LogFatal
)

// String returns the string representation of a log level
func (l LogLevel) String() string {
	switch l {
	case LogDebug:
		return "DEBUG"
	case LogInfo:
		return "INFO"
	case LogWarning:
		return "WARN"
	case LogError:
		return "ERROR"
	case LogFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Stage     string                 `json:"stage"`
	Message   string                 `json:"message"`
	Progress  float64                `json:"progress,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// BuildLogger provides comprehensive logging for dictionary building
type BuildLogger struct {
	mu           sync.Mutex
	entries      []LogEntry
	logFile      *os.File
	fileWriter   *log.Logger
	callbacks    []func(LogEntry)
	minLevel     LogLevel
	currentStage string
	stageStart   time.Time
	buildStart   time.Time
	// support multiple file outputs
	extraClosers []io.Closer
}

// NewBuildLogger creates a new build logger
func NewBuildLogger(logPath string) (*BuildLogger, error) {
	logger := &BuildLogger{
		entries:    make([]LogEntry, 0, 1000),
		callbacks:  make([]func(LogEntry), 0),
		minLevel:   LogInfo,
		buildStart: time.Now(),
	}

	if logPath != "" {
		// Ensure log directory exists
		logDir := filepath.Dir(logPath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		// Open log file
		file, err := os.Create(logPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create log file: %w", err)
		}
		logger.logFile = file
		logger.fileWriter = log.New(file, "", 0)
		logger.extraClosers = append(logger.extraClosers, file)
	}

	return logger, nil
}

// Close flushes and closes any underlying resources.
func (l *BuildLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	var firstErr error
	if l.logFile != nil {
		if err := l.logFile.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		l.logFile = nil
	}
	for _, c := range l.extraClosers {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	l.extraClosers = nil
	l.fileWriter = nil
	return firstErr
}

// SetMinLevel sets the minimum log level
func (l *BuildLogger) SetMinLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.minLevel = level
}

// AddCallback adds a callback function for log entries
func (l *BuildLogger) AddCallback(callback func(LogEntry)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.callbacks = append(l.callbacks, callback)
}

// StartStage marks the beginning of a new build stage
func (l *BuildLogger) StartStage(stage string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Log end of previous stage if exists
	if l.currentStage != "" {
		duration := time.Since(l.stageStart)
		l.logEntry(LogInfo, l.currentStage, fmt.Sprintf("Stage completed in %v", duration), -1, map[string]interface{}{
			"duration_ms": duration.Milliseconds(),
		})
	}

	l.currentStage = stage
	l.stageStart = time.Now()
	l.logEntry(LogInfo, stage, fmt.Sprintf("Starting stage: %s", stage), -1, nil)
}

// Log logs a message at the specified level
func (l *BuildLogger) Log(level LogLevel, message string, progress float64, details map[string]interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logEntry(level, l.currentStage, message, progress, details)
}

// Debug logs a debug message
func (l *BuildLogger) Debug(message string, details map[string]interface{}) {
	l.Log(LogDebug, message, -1, details)
}

// Info logs an info message
func (l *BuildLogger) Info(message string, progress float64) {
	l.Log(LogInfo, message, progress, nil)
}

// Warning logs a warning message
func (l *BuildLogger) Warning(message string, details map[string]interface{}) {
	l.Log(LogWarning, message, -1, details)
}

// Error logs an error message
func (l *BuildLogger) Error(message string, err error) {
	details := map[string]interface{}{
		"error": err.Error(),
	}
	l.Log(LogError, message, -1, details)
}

// Fatal logs a fatal error and returns an error
func (l *BuildLogger) Fatal(message string, err error) error {
	l.Error(message, err)
	return fmt.Errorf("%s: %w", message, err)
}

// Progress logs a progress update
func (l *BuildLogger) Progress(message string, current, total int) {
	progress := -1.0
	if total > 0 {
		progress = float64(current) / float64(total)
	}

	details := map[string]interface{}{
		"current": current,
		"total":   total,
	}

	l.Log(LogInfo, message, progress, details)
}

// Metric logs a metric or statistic
func (l *BuildLogger) Metric(name string, value interface{}) {
	details := map[string]interface{}{
		"metric": name,
		"value":  value,
	}
	l.Log(LogInfo, fmt.Sprintf("Metric: %s = %v", name, value), -1, details)
}

// logEntry internal method to log an entry
func (l *BuildLogger) logEntry(level LogLevel, stage, message string, progress float64, details map[string]interface{}) {
	if level < l.minLevel {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Stage:     stage,
		Message:   message,
		Progress:  progress,
		Details:   details,
	}

	// Store entry
	l.entries = append(l.entries, entry)

	// Write to file if available
	if l.fileWriter != nil {
		timestamp := entry.Timestamp.Format("2006-01-02 15:04:05.000")
		progressStr := ""
		if progress >= 0 {
			progressStr = fmt.Sprintf(" [%.1f%%]", progress*100)
		}

		logLine := fmt.Sprintf("[%s] %s %s: %s%s",
			timestamp,
			entry.Level.String(),
			stage,
			message,
			progressStr,
		)

		if details != nil && len(details) > 0 {
			logLine += fmt.Sprintf(" | %v", details)
		}

		l.fileWriter.Println(logLine)
	}

	// Call callbacks
	for _, callback := range l.callbacks {
		callback(entry)
	}
}

// GetEntries returns all log entries
func (l *BuildLogger) GetEntries() []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	result := make([]LogEntry, len(l.entries))
	copy(result, l.entries)
	return result
}

// GetSummary returns a build summary
func (l *BuildLogger) GetSummary() BuildSummary {
	l.mu.Lock()
	defer l.mu.Unlock()

	summary := BuildSummary{
		StartTime:    l.buildStart,
		EndTime:      time.Now(),
		Duration:     time.Since(l.buildStart),
		TotalEntries: len(l.entries),
		Stages:       make(map[string]StageSummary),
	}

	// Count entries by level
	for _, entry := range l.entries {
		switch entry.Level {
		case LogDebug:
			summary.DebugCount++
		case LogInfo:
			summary.InfoCount++
		case LogWarning:
			summary.WarningCount++
		case LogError:
			summary.ErrorCount++
		case LogFatal:
			summary.FatalCount++
		}

		// Track stages
		if entry.Stage != "" {
			stage := summary.Stages[entry.Stage]
			stage.Name = entry.Stage
			stage.EntryCount++
			if stage.FirstEntry.IsZero() || entry.Timestamp.Before(stage.FirstEntry) {
				stage.FirstEntry = entry.Timestamp
			}
			if entry.Timestamp.After(stage.LastEntry) {
				stage.LastEntry = entry.Timestamp
			}
			summary.Stages[entry.Stage] = stage
		}
	}

	// Calculate stage durations
	for name, stage := range summary.Stages {
		stage.Duration = stage.LastEntry.Sub(stage.FirstEntry)
		summary.Stages[name] = stage
	}

	return summary
}

// CloseWithSummary closes the logger and flushes any pending writes with a summary
func (l *BuildLogger) CloseWithSummary() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Log final summary
	summary := l.GetSummary()
	l.logEntry(LogInfo, "summary", fmt.Sprintf("Build completed in %v", summary.Duration), 1.0, map[string]interface{}{
		"total_entries": summary.TotalEntries,
		"errors":        summary.ErrorCount,
		"warnings":      summary.WarningCount,
	})

	if l.fileWriter != nil {
		// Write summary to file
		l.fileWriter.Println(strings.Repeat("=", 80))
		l.fileWriter.Printf("BUILD SUMMARY\n")
		l.fileWriter.Printf("Duration: %v\n", summary.Duration)
		l.fileWriter.Printf("Total Entries: %d\n", summary.TotalEntries)
		l.fileWriter.Printf("Errors: %d, Warnings: %d\n", summary.ErrorCount, summary.WarningCount)
		l.fileWriter.Println(strings.Repeat("=", 80))
	}
	// Close all closers
	var firstErr error
	if l.logFile != nil {
		if err := l.logFile.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		l.logFile = nil
	}
	for _, c := range l.extraClosers {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	l.extraClosers = nil
	return firstErr
}

// BuildSummary contains summary statistics for a build
type BuildSummary struct {
	StartTime    time.Time               `json:"start_time"`
	EndTime      time.Time               `json:"end_time"`
	Duration     time.Duration           `json:"duration"`
	TotalEntries int                     `json:"total_entries"`
	DebugCount   int                     `json:"debug_count"`
	InfoCount    int                     `json:"info_count"`
	WarningCount int                     `json:"warning_count"`
	ErrorCount   int                     `json:"error_count"`
	FatalCount   int                     `json:"fatal_count"`
	Stages       map[string]StageSummary `json:"stages"`
}

// StageSummary contains summary for a single stage
type StageSummary struct {
	Name       string        `json:"name"`
	FirstEntry time.Time     `json:"first_entry"`
	LastEntry  time.Time     `json:"last_entry"`
	Duration   time.Duration `json:"duration"`
	EntryCount int           `json:"entry_count"`
}

// MultiWriter writes to multiple io.Writers
type MultiWriter struct {
	writers []io.Writer
}

// NewMultiWriter creates a new MultiWriter
func NewMultiWriter(writers ...io.Writer) *MultiWriter {
	return &MultiWriter{writers: writers}
}

// Write writes to all writers
func (mw *MultiWriter) Write(p []byte) (n int, err error) {
	for _, w := range mw.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
	}
	return len(p), nil
}

// AddFile attaches an additional file to receive logs. Caller is responsible for path validity.
func (l *BuildLogger) AddFile(path string) error {
	if path == "" {
		return nil
	}
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	// Wrap existing writer and new file into MultiWriter
	var writers []io.Writer
	if l.fileWriter != nil {
		// Underlying writer for existing logger.fileWriter is unknown; we rely on Println to write to MultiWriter
	}
	// Build a combined writer from existing logFile (if any) and new file
	if l.logFile != nil {
		writers = append(writers, l.logFile)
	}
	writers = append(writers, f)
	mw := NewMultiWriter(writers...)
	l.fileWriter = log.New(mw, "", 0)
	l.extraClosers = append(l.extraClosers, f)
	return nil
}
