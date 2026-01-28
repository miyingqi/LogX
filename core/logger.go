package core

import (
	config2 "LogX/config"
)

type Logger interface {
	Trace(format string, args ...interface{})
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Fatal(format string, args ...interface{})
	Panic(format string, args ...interface{})

	Field(fields map[string]interface{}) *Logger

	SetLevel(level config2.LogLevel)

	Caller(skip int) *Logger
	Close() error
}
