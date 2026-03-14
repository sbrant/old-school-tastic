package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	mu      sync.Mutex
	logFile *os.File
	enabled bool
)

func Init(enable bool) {
	enabled = enable
	if !enabled {
		return
	}

	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".config", "meshtastic-cli")
	os.MkdirAll(dir, 0755)

	path := filepath.Join(dir, "log")

	// Rotate if > 5MB
	if info, err := os.Stat(path); err == nil && info.Size() > 5*1024*1024 {
		os.Rename(path, path+".old")
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not open log file: %v\n", err)
		return
	}
	logFile = f
}

func Close() {
	mu.Lock()
	defer mu.Unlock()
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

func write(level, component, msg string) {
	if !enabled || logFile == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	ts := time.Now().Format("2006-01-02 15:04:05.000")
	fmt.Fprintf(logFile, "%s [%s] %s: %s\n", ts, level, component, msg)
}

func Info(component, msg string) {
	write("INFO", component, msg)
}

func Warn(component, msg string) {
	write("WARN", component, msg)
}

func Error(component, msg string) {
	write("ERROR", component, msg)
}

func Debug(component, msg string) {
	write("DEBUG", component, msg)
}

// LogError writes to a separate error log file for crash diagnostics
func LogError(msg string, err error) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".config", "meshtastic-cli")
	os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, "error.log")

	// Rotate if > 1MB
	if info, e := os.Stat(path); e == nil && info.Size() > 1024*1024 {
		os.Rename(path, path+".old")
	}

	f, e := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if e != nil {
		return
	}
	defer f.Close()

	ts := time.Now().Format("2006-01-02 15:04:05.000")
	fmt.Fprintf(f, "%s ERROR: %s: %v\n", ts, msg, err)
}
