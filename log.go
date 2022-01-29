package daoongorm

import (
	"context"
	"encoding/json"
	"fmt"
)

const (
	LevelDebug = "DEBUG"
	LevelInfo  = "INFO"
	LevelWarn  = "WARN"
	LevelError = "ERROR"
)

type LogData struct {
	Level   string
	Content string
}

type Logger interface {
	Errorc(ctx context.Context, format string, args ...interface{})
	Infoc(ctx context.Context, format string, args ...interface{})
}

var logger Logger = &DefaultLogger{}

func SetLogger(newLogger Logger) {
	logger = newLogger
}

type DefaultLogger struct{}

func (l *DefaultLogger) Errorc(ctx context.Context, format string, args ...interface{}) {
	content := fmt.Sprintf(format, args...)
	logStr, _ := json.Marshal(LogData{Level: LevelError, Content: content})
	fmt.Println(string(logStr))
}

func (l *DefaultLogger) Infoc(ctx context.Context, format string, args ...interface{}) {
	content := fmt.Sprintf(format, args...)
	logStr, _ := json.Marshal(LogData{Level: LevelInfo, Content: content})
	fmt.Println(string(logStr))
}
