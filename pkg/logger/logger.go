package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RotatingLogger handles log rotation based on date.
type RotatingLogger struct {
	logDir       string
	retentionDays int
	currentDate  string
	currentFile  *os.File
	mu           sync.Mutex
}

// NewRotatingLogger creates a new rotating logger.
func NewRotatingLogger(logDir string, retentionDays int) (*RotatingLogger, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	rl := &RotatingLogger{
		logDir:        logDir,
		retentionDays: retentionDays,
		currentDate:   time.Now().Format("2006-01-02"),
	}

	// Open initial log file
	if err := rl.openLogFile(); err != nil {
		return nil, err
	}

	// Start cleanup goroutine
	go rl.cleanupOldLogs()

	return rl, nil
}

// openLogFile opens the current day's log file.
func (rl *RotatingLogger) openLogFile() error {
	logFile := filepath.Join(rl.logDir, fmt.Sprintf("microcron-ce-%s.log", rl.currentDate))
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	if rl.currentFile != nil {
		rl.currentFile.Close()
	}
	rl.currentFile = f
	return nil
}

// Log writes a log entry to both stdout and the log file.
func (rl *RotatingLogger) Log(message string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	timestamp := time.Now()
	formattedMessage := fmt.Sprintf("[%s] %s\n", timestamp.Format("2006-01-02 15:04:05"), message)

	// Write to stdout
	fmt.Print(formattedMessage)

	// Check if date has changed and rotate log file if needed
	currentDate := timestamp.Format("2006-01-02")
	if currentDate != rl.currentDate {
		rl.currentDate = currentDate
		if err := rl.openLogFile(); err != nil {
			fmt.Printf("Error rotating log file: %v\n", err)
			return
		}
	}

	// Write to log file
	if _, err := io.WriteString(rl.currentFile, formattedMessage); err != nil {
		fmt.Printf("Error writing to log file: %v\n", err)
	}
}

// cleanupOldLogs periodically removes log files older than the retention period.
func (rl *RotatingLogger) cleanupOldLogs() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		rl.removeOldLogs()
		rl.mu.Unlock()
	}
}

// removeOldLogs deletes log files older than the retention period.
func (rl *RotatingLogger) removeOldLogs() {
	entries, err := os.ReadDir(rl.logDir)
	if err != nil {
		fmt.Printf("Error reading log directory: %v\n", err)
		return
	}

	cutoffDate := time.Now().AddDate(0, 0, -rl.retentionDays)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoffDate) {
			logFile := filepath.Join(rl.logDir, entry.Name())
			if err := os.Remove(logFile); err != nil {
				fmt.Printf("Error removing old log file %s: %v\n", logFile, err)
			} else {
				fmt.Printf("Removed old log file: %s\n", logFile)
			}
		}
	}
}

// Close closes the current log file.
func (rl *RotatingLogger) Close() error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.currentFile != nil {
		return rl.currentFile.Close()
	}
	return nil
}
