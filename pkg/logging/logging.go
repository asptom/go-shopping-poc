package logging

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
)

var (
	level      = INFO
	logger     = log.New(os.Stdout, "", 0)
	levelNames = []string{"DEBUG", "INFO", "WARNING", "ERROR"}
	mu         sync.RWMutex
)

// SetLevel sets the global log level (e.g., "DEBUG", "INFO", "WARNING", "ERROR").
func SetLevel(lvl string) {
	mu.Lock()
	defer mu.Unlock()
	switch strings.ToUpper(lvl) {
	case "DEBUG":
		level = DEBUG
	case "INFO":
		level = INFO
	case "WARNING":
		level = WARNING
	case "ERROR":
		level = ERROR
	default:
		level = INFO
	}
}

// logf logs a message at the given level.
func logf(lvl LogLevel, format string, v ...any) {
	mu.RLock()
	defer mu.RUnlock()
	if lvl < level {
		return
	}
	timestamp := time.Now().Format(time.RFC3339)
	prefix := fmt.Sprintf("[%s] %s: ", timestamp, levelNames[lvl])
	logger.Printf(prefix+format, v...)
}

// Debug logs a debug message.
func Debug(format string, v ...any) {
	logf(DEBUG, format, v...)
}

// Info logs an info message.
func Info(format string, v ...any) {
	logf(INFO, format, v...)
}

// Warning logs a warning message.
func Warning(format string, v ...any) {
	logf(WARNING, format, v...)
}

// Error logs an error message.
func Error(format string, v ...any) {
	logf(ERROR, format, v...)
}
