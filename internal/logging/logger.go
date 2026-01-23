package logging

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func ParseLevel(s string) Level {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return LevelDebug
	case "INFO":
		return LevelInfo
	case "WARN", "WARNING":
		return LevelWarn
	case "ERROR":
		return LevelError
	default:
		return LevelInfo
	}
}

type Logger struct {
	level  Level
	logger *log.Logger
	uuid   string
}

func New(level Level, output io.Writer, uuid string) *Logger {
	if output == nil {
		output = os.Stdout
	}
	return &Logger{
		level:  level,
		logger: log.New(output, "", 0),
		uuid:   redactUUID(uuid),
	}
}

func redactUUID(uuid string) string {
	if len(uuid) <= 8 {
		return uuid
	}
	return uuid[:4] + "..." + uuid[len(uuid)-4:]
}

func (l *Logger) log(level Level, msg string, fields map[string]interface{}) {
	if level < l.level {
		return
	}

	logEntry := map[string]interface{}{
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
		"level":         level.String(),
		"msg":           msg,
		"gateway_uuid":  l.uuid,
	}

	for k, v := range fields {
		if k == "token" || k == "authorization" {
			continue
		}
		logEntry[k] = v
	}

	jsonData, err := json.Marshal(logEntry)
	if err != nil {
		l.logger.Printf(`{"level":"ERROR","msg":"failed to marshal log entry","error":"%s"}`, err.Error())
		return
	}

	l.logger.Println(string(jsonData))
}

func (l *Logger) Debug(msg string, fields map[string]interface{}) {
	l.log(LevelDebug, msg, fields)
}

func (l *Logger) Info(msg string, fields map[string]interface{}) {
	l.log(LevelInfo, msg, fields)
}

func (l *Logger) Warn(msg string, fields map[string]interface{}) {
	l.log(LevelWarn, msg, fields)
}

func (l *Logger) Error(msg string, fields map[string]interface{}) {
	l.log(LevelError, msg, fields)
}
