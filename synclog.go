package LogX

import (
	"LogX/config"
	"LogX/core"
	"LogX/hooks"
	"fmt"
	"os"
	"sync"
	"time"
)

type SyncLogger struct {
	config    config.LoggerConfig // 全局不变配置
	model     string              // 全局不变模型名
	mutex     sync.Mutex          // 全局锁：仅保护「配置修改」和「日志最终输出」
	formatter core.Formatter      // 全局共享格式化器
	hook      *hooks.HookManager  // 全局共享钩子管理器
}

type logContext struct {
	logger *SyncLogger    // 关联无状态日志器
	fields map[string]any // 本次调用私有字段（无共享）
	skip   int            // 本次调用私有skip（无共享）
}

// NewDefaultSyncLogger 创建默认同步日志器（比原版本少2行代码，无fields/skip初始化）
func NewDefaultSyncLogger(model string) *SyncLogger {
	if model == "" {
		model = "default"
	}
	defaultConfig := config.NewDefaultLoggerConfig()
	defaultFormatter := core.TextFormatter{
		EnableColor:     config.DefaultEnableColor,
		TimestampFormat: time.DateTime,
		ShowCaller:      false,
	}
	// 无fields/skip，初始化代码更简单
	logger := &SyncLogger{
		config:    defaultConfig,
		model:     model,
		formatter: &defaultFormatter,
		hook:      hooks.NewHookManager(),
	}
	return logger
}

// NewSyncLogger 自定义配置创建日志器（同理，删除fields/skip初始化，代码更简单）
func NewSyncLogger(model string, conf config.LC) *SyncLogger {
	con := config.ParseLoggerConfigFromJSON(conf)
	logger := &SyncLogger{
		config: con,
		model:  model,
		formatter: &core.TextFormatter{
			EnableColor:     config.DefaultEnableColor,
			TimestampFormat: time.DateTime,
			ShowCaller:      con.ShowCaller,
		},
		hook: hooks.NewHookManager(),
	}
	return logger
}

func (l *SyncLogger) Field(fields map[string]any) *logContext {
	// 初始化本次调用的私有上下文，预分配字段空间
	ctx := &logContext{
		logger: l,
		fields: make(map[string]any, len(fields)),
		skip:   4, // 默认skip，和原版本一致
	}
	for k, v := range fields {
		ctx.fields[k] = v
	}
	return ctx
}

func (l *logContext) Caller(skip int) *logContext {
	l.skip = skip
	return l
}

func (l *logContext) Trace(format string, args ...interface{}) {
	l.logger.output(config.TRACE, fmt.Sprintf(format, args...), l.fields, l.skip)
}
func (l *logContext) Debug(format string, args ...interface{}) {
	l.logger.output(config.DEBUG, fmt.Sprintf(format, args...), l.fields, l.skip)
}
func (l *logContext) Info(format string, args ...interface{}) {
	l.logger.output(config.INFO, fmt.Sprintf(format, args...), l.fields, l.skip)
}
func (l *logContext) Warn(format string, args ...interface{}) {
	l.logger.output(config.WARNING, fmt.Sprintf(format, args...), l.fields, l.skip)
}
func (l *logContext) Error(format string, args ...interface{}) {
	l.logger.output(config.ERROR, fmt.Sprintf(format, args...), l.fields, l.skip)
}
func (l *logContext) Fatal(format string, args ...interface{}) {
	l.logger.output(config.FATAL, fmt.Sprintf(format, args...), l.fields, l.skip)
	os.Exit(1)
}
func (l *logContext) Panic(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.logger.output(config.PANIC, msg, l.fields, l.skip)
	panic(msg)
}

func (l *SyncLogger) Trace(format string, args ...interface{}) {
	l.Field(nil).Trace(format, args...)
}
func (l *SyncLogger) Debug(format string, args ...interface{}) {
	l.Field(nil).Debug(format, args...)
}
func (l *SyncLogger) Info(format string, args ...interface{}) {
	l.Field(nil).Info(format, args...)
}
func (l *SyncLogger) Warn(format string, args ...interface{}) {
	l.Field(nil).Warn(format, args...)
}
func (l *SyncLogger) Error(format string, args ...interface{}) {
	l.Field(nil).Error(format, args...)
}
func (l *SyncLogger) Fatal(format string, args ...interface{}) {
	l.Field(nil).Fatal(format, args...)
}
func (l *SyncLogger) Panic(format string, args ...interface{}) {
	l.Field(nil).Panic(format, args...)
}

func (l *SyncLogger) output(level config.LogLevel, message string, fields map[string]any, skip int) {
	if level < l.config.Level {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	entry := core.Entry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Fields:    fields,
		Skip:      skip,
		Model:     l.model,
	}

	// 执行钩子（原有逻辑不变）
	skipHook, errs := l.hook.RunHooks(hooks.StageBeforeFormat, level, &entry)
	if errs != nil {
		fmt.Printf("日志钩子执行失败：%v\n", errs)
		return
	}
	if skipHook {
		return
	}

	// 格式化日志（原有逻辑不变）
	logBytes, err := l.formatter.Format(&entry)
	if err != nil {
		fmt.Printf("日志格式化失败：%v\n", err)
		return
	}

	// 后续钩子+控制台输出（原有逻辑不变，无任何状态重置）
	skipHook, errs = l.hook.RunHooks(hooks.StageAfterFormat, level, &entry)
	if len(errs) > 0 {
		fmt.Printf("【钩子错误】格式化后：%v\n", errs)
	}
	if skipHook {
		return
	}
	if l.config.OutputConsole {
		if level >= config.ERROR {
			_, _ = os.Stderr.Write(logBytes)
		} else {
			_, _ = os.Stdout.Write(logBytes)
		}
	}

}

func (l *SyncLogger) SetLevel(level config.LogLevel) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.config.Level = level
}
func (l *SyncLogger) SetShowCaller(show bool) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if tf, ok := l.formatter.(*core.TextFormatter); ok {
		tf.ShowCaller = show
	}
}
func (l *SyncLogger) SetFormatter(formatter core.Formatter) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.formatter = formatter
}
func (l *SyncLogger) AddHook(hook hooks.HookBase) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.hook.AddHook(hook)
}

// Close 关闭日志器（原有逻辑不变）
func (l *SyncLogger) Close() {
}
