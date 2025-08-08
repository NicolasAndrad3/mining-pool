package logs

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	logFile      *os.File
	outputFormat = "text"
	silent       = false
	mutex        sync.Mutex
)

const maxMessageSize = 2048

type logMeta struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	File      string `json:"file"`
	Func      string `json:"func"`
	Line      int    `json:"line"`
	Message   string `json:"message"`
}

// Init initializes the logger according to the environment
func Init(env string) {
	if logPath := os.Getenv("POOL_LOG_FILE"); logPath != "" {
		if err := InitFileOutput(logPath); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		}
	}

	if strings.ToLower(env) == "production" {
		EnableJSONFormat()
	}

	Info("Logger initialized - format: %s, environment: %s", outputFormat, env)
}

func InitFileOutput(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("logger: failed to open log file: %w", err)
	}
	logFile = f
	log.SetOutput(f)
	return nil
}

func CloseLogFile() {
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

func EnableJSONFormat() {
	outputFormat = "json"
}

func SilenceLogs() {
	silent = true
}

func logInternal(level string, msg string, args ...any) {
	if silent {
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	now := time.Now()
	timestamp := now.Format(time.RFC3339)

	pc, file, line, ok := runtime.Caller(3)
	fnName := "unknown"
	if ok {
		if fn := runtime.FuncForPC(pc); fn != nil {
			fnName = filepath.Base(fn.Name())
		}
	}
	fileName := filepath.Base(file)

	fullMsg := fmt.Sprintf(msg, args...)
	if len(fullMsg) > maxMessageSize {
		fullMsg = fullMsg[:maxMessageSize] + "...[truncated]"
	}

	if outputFormat == "json" {
		entry := logMeta{
			Timestamp: timestamp,
			Level:     level,
			File:      fileName,
			Func:      fnName,
			Line:      line,
			Message:   fullMsg,
		}
		if encoded, err := json.Marshal(entry); err == nil {
			fmt.Fprintln(os.Stdout, string(encoded))
		} else {
			fmt.Fprintf(os.Stderr, "log encoding error: %v\n", err)
		}
		return
	}

	color := levelColor(level)
	tag := paddedLevel(level)
	reset := "\033[0m"

	fmt.Printf("%s[%s]%s %s [%s:%d > %s] %s\n",
		color, tag, reset, timestamp, fileName, line, fnName, fullMsg)
}

func Debug(msg string, args ...any) { logInternal("DEBUG", msg, args...) }
func Info(msg string, args ...any)  { logInternal("INFO", msg, args...) }
func Warn(msg string, args ...any)  { logInternal("WARN", msg, args...) }
func Error(msg string, args ...any) { logInternal("ERROR", msg, args...) }
func Fatal(msg string, args ...any) {
	logInternal("FATAL", msg, args...)
	os.Exit(1)
}

func Debugf(format string, args ...any) { logInternal("DEBUG", format, args...) }
func Infof(format string, args ...any)  { logInternal("INFO", format, args...) }
func Warnf(format string, args ...any)  { logInternal("WARN", format, args...) }
func Errorf(format string, args ...any) { logInternal("ERROR", format, args...) }
func Fatalf(format string, args ...any) {
	logInternal("FATAL", format, args...)
	os.Exit(1)
}

func levelColor(level string) string {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return "\033[36m"
	case "INFO":
		return "\033[32m"
	case "WARN":
		return "\033[33m"
	case "ERROR":
		return "\033[31m"
	case "FATAL":
		return "\033[41m"
	default:
		return "\033[0m"
	}
}

func paddedLevel(level string) string {
	upper := strings.ToUpper(level)
	if len(upper) < 5 {
		return fmt.Sprintf("%-5s", upper)
	}
	return upper
}

func WithFields(fields map[string]interface{}) *logEntry {
	return &logEntry{fields: fields}
}

type logEntry struct {
	fields map[string]interface{}
}

func (e *logEntry) WithError(err error) {
	if e.fields == nil {
		e.fields = map[string]interface{}{}
	}
	e.fields["error"] = err.Error()
}

func (e *logEntry) log(level string, msg string, args ...any) {
	prefix := ""
	if id, ok := e.fields["request_id"]; ok {
		prefix = fmt.Sprintf("[req:%v] ", id)
	}
	if errStr, ok := e.fields["error"]; ok && errStr != "" {
		prefix += fmt.Sprintf("[error:%v] ", errStr)
	}

	formatted := fmt.Sprintf(msg, args...)
	fullMsg := prefix + formatted

	logInternal(level, fullMsg)
}

func (e *logEntry) Debug(msg string, args ...any) { e.log("DEBUG", msg, args...) }
func (e *logEntry) Info(msg string, args ...any)  { e.log("INFO", msg, args...) }
func (e *logEntry) Warn(msg string, args ...any)  { e.log("WARN", msg, args...) }
func (e *logEntry) Error(msg string, args ...any) { e.log("ERROR", msg, args...) }
func (e *logEntry) Fatal(msg string, args ...any) {
	e.log("FATAL", msg, args...)
	os.Exit(1)
}
