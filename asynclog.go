package LogX

import (
	config2 "LogX/config"
	"LogX/core"
	"LogX/hooks"
	"sync"
	"time"
)

type AsyncLogger struct {
	config    config2.LoggerConfig
	model     string
	logChan   chan *core.Entry
	entryPool sync.Pool
	formatter core.Formatter
	hook      *hooks.HookManager
	skip      int
}

func NewDefaultAAsyncLogger(model string) *AsyncLogger {
	if model == "" {
		model = "default"
	}
	logger := &AsyncLogger{
		config:  config2.NewDefaultLoggerConfig(),
		model:   model,
		logChan: make(chan *core.Entry),
		entryPool: sync.Pool{
			New: func() interface{} {
				return core.NewEntry()
			},
		},
	}

	return logger
}

func (l *AsyncLogger) Trace(format string, args ...interface{}) {

}

func (l *AsyncLogger) Debug(format string, args ...interface{}) {

}

func (l *AsyncLogger) Info(format string, args ...interface{}) {

}

func (l *AsyncLogger) Warn(format string, args ...interface{}) {

}

func (l *AsyncLogger) Error(format string, args ...interface{}) {

}

func (l *AsyncLogger) Fatal(format string, args ...interface{}) {

}

func (l *AsyncLogger) Panic(format string, args ...interface{}) {

}
func (l *AsyncLogger) Field(fields map[string]interface{}) *AsyncLogger {

	return l
}

// Caller 设置调用栈跳过层级（支持链式调用）
func (l *AsyncLogger) Caller(skip int) *AsyncLogger {

	return l
}

// SetLevel 设置日志级别
func (l *AsyncLogger) SetLevel(level config2.LogLevel) {

}

func (l *AsyncLogger) SetShowCaller(show bool) {

}

// SetFormatter 设置格式化器（支持动态切换，如JSONFormatter）
func (l *AsyncLogger) SetFormatter(formatter core.Formatter) {

}
func (l *AsyncLogger) AddHook(hook hooks.HookBase) {

}

func (l *AsyncLogger) log(level config2.LogLevel, message string) {
	if level < l.config.Level {
		return
	}
	entry, ok := l.entryPool.Get().(*core.Entry)
	if !ok {
		entry = core.NewEntry()
	}
	entry.SetEntry(time.Now(), level, message, l.model, l.skip, nil)
}
func (l *AsyncLogger) Close() {

}
