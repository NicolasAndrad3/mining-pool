package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

type Level string

const (
	LvlDebug Level = "DEBUG"
	LvlInfo  Level = "INFO"
	LvlWarn  Level = "WARN"
	LvlError Level = "ERROR"
	LvlFatal Level = "FATAL"
)

var (
	activeLogger *StructuredLogger
	_pool        = sync.Pool{
		New: func() interface{} {
			return make(map[string]interface{}, 10)
		},
	}
)

type StructuredLogger struct {
	writer      *log.Logger
	threshold   Level
	identifier  string
	environment string
	context     context.Context
	fields      map[string]interface{}
	err         error
}

// Implementações de Info() e Fatal() para StructuredLogger
func (s *StructuredLogger) Info(msg string) {
	s.write(context.Background(), LvlInfo, msg, nil)
}

func (s *StructuredLogger) Fatal(msg string) {
	s.write(context.Background(), LvlFatal, msg, nil)
	os.Exit(1)
}

func Init(service string, env string, min Level, w io.Writer) {
	if w == nil {
		w = os.Stdout
	}
	activeLogger = &StructuredLogger{
		writer:      log.New(w, "", 0),
		threshold:   min,
		identifier:  service,
		environment: env,
		context:     context.Background(),
	}
}

func ensure() *StructuredLogger {
	if activeLogger == nil {
		Init("anonymous", "unset", LvlInfo, os.Stdout)
	}
	return activeLogger
}

func getTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func (s *StructuredLogger) shouldEmit(level Level) bool {
	priority := map[Level]int{
		LvlDebug: 1,
		LvlInfo:  2,
		LvlWarn:  3,
		LvlError: 4,
		LvlFatal: 5,
	}
	return priority[level] >= priority[s.threshold]
}

func (s *StructuredLogger) write(ctx context.Context, level Level, msg string, details map[string]interface{}) {
	if !s.shouldEmit(level) {
		return
	}

	entry := _pool.Get().(map[string]interface{})
	defer func() {
		for k := range entry {
			delete(entry, k)
		}
		_pool.Put(entry)
	}()

	entry["timestamp"] = getTimestamp()
	entry["level"] = level
	entry["message"] = msg
	entry["service"] = s.identifier
	entry["env"] = s.environment

	// Contexto + Campos adicionais
	for k, v := range extractContext(ctx) {
		entry[k] = v
	}
	for k, v := range s.fields {
		entry[k] = v
	}
	for k, v := range details {
		entry[k] = v
	}
	if s.err != nil {
		entry["error"] = s.err.Error()
	}

	if payload, err := json.Marshal(entry); err == nil {
		s.writer.Println(string(payload))
	} else {
		fallback := fmt.Sprintf(`{"level":"ERROR","msg":"log serialization failed: %v"}`, err)
		s.writer.Println(fallback)
	}
}

func extractContext(ctx context.Context) map[string]interface{} {
	fields := make(map[string]interface{}, 4)
	if ctx == nil {
		return fields
	}
	var keys = []string{"trace_id", "span_id", "request_id", "user_id"}
	for _, key := range keys {
		if val, ok := ctx.Value(key).(string); ok && val != "" {
			fields[key] = val
		}
	}
	return fields
}

// Logging helpers

func Debug(msg string, kv ...map[string]interface{}) {
	ensure().write(context.Background(), LvlDebug, msg, merge(kv...))
}

func Info(msg string, kv ...map[string]interface{}) {
	ensure().write(context.Background(), LvlInfo, msg, merge(kv...))
}

func Warn(msg string, kv ...map[string]interface{}) {
	ensure().write(context.Background(), LvlWarn, msg, merge(kv...))
}

func Error(msg string, kv ...map[string]interface{}) {
	ensure().write(context.Background(), LvlError, msg, merge(kv...))
}

func Fatal(msg string, kv ...map[string]interface{}) {
	ensure().write(context.Background(), LvlFatal, msg, merge(kv...))
	os.Exit(1)
}

func Debugf(format string, args ...interface{}) {
	ensure().write(context.Background(), LvlDebug, fmt.Sprintf(format, args...), nil)
}

func Infof(format string, args ...interface{}) {
	ensure().write(context.Background(), LvlInfo, fmt.Sprintf(format, args...), nil)
}

func Warnf(format string, args ...interface{}) {
	ensure().write(context.Background(), LvlWarn, fmt.Sprintf(format, args...), nil)
}

func Errorf(format string, args ...interface{}) {
	ensure().write(context.Background(), LvlError, fmt.Sprintf(format, args...), nil)
}

func Fatalf(format string, args ...interface{}) {
	ensure().write(context.Background(), LvlFatal, fmt.Sprintf(format, args...), nil)
	os.Exit(1)
}

func merge(sets ...map[string]interface{}) map[string]interface{} {
	if len(sets) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(sets)*4)
	for _, m := range sets {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

func WithFields(fields map[string]interface{}) *StructuredLogger {
	l := *ensure()
	l.fields = fields
	return &l
}

func WithError(err error) *StructuredLogger {
	l := *ensure()
	l.err = err
	return &l
}
