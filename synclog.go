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

// SyncLogger 同步日志器（线程安全，支持控制台/文件输出、扩展字段、调用方、颜色）
type SyncLogger struct {
	config    config.LoggerConfig    // 日志配置（级别、输出方式、文件路径等）
	model     string                 // 服务/模型名
	mutex     sync.Mutex             // 全局锁（保证多协程安全写入）
	formatter core.Formatter         // 格式化器（接口类型，支持动态切换）
	fields    map[string]interface{} // 临时存储扩展字段（Field方法设置）
	skip      int                    // 临时存储调用栈跳过层级（Caller方法设置）
	hook      *hooks.HookManager     // 钩子管理器
}

// NewDefaultSyncLogger 创建默认同步日志器
// 参数：model - 服务/模型名（如"HTTP"、"DB"）
func NewDefaultSyncLogger(model string) *SyncLogger {
	if model == "" {
		model = "default"
	}

	// 1. 创建默认配置
	defaultConfig := config.NewDefaultLoggerConfig()

	// 2. 默认使用文本格式化器（带颜色）
	defaultFormatter := core.TextFormatter{
		EnableColor:     config.DefaultEnableColor,
		TimestampFormat: time.DateTime,
		ShowCaller:      false,
	}

	// 3. 初始化日志器
	logger := &SyncLogger{
		config:    defaultConfig,
		model:     model,
		formatter: &defaultFormatter,
		fields:    make(map[string]interface{}), // 初始化扩展字段
		skip:      4,                            // 默认跳过4层（适配日志器封装层级）
		hook:      hooks.NewHookManager(),
	}
	return logger
}

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
		fields: make(map[string]interface{}),
		skip:   4,
		hook:   hooks.NewHookManager(),
	}
	return logger
}
func (l *SyncLogger) Trace(format string, args ...interface{}) {
	l.log(config.TRACE, fmt.Sprintf(format, args...))
}

func (l *SyncLogger) Debug(format string, args ...interface{}) {
	l.log(config.DEBUG, fmt.Sprintf(format, args...))
}

func (l *SyncLogger) Info(format string, args ...interface{}) {
	l.log(config.INFO, fmt.Sprintf(format, args...))
}

func (l *SyncLogger) Warn(format string, args ...interface{}) {
	l.log(config.WARNING, fmt.Sprintf(format, args...))
}

func (l *SyncLogger) Error(format string, args ...interface{}) {
	l.log(config.ERROR, fmt.Sprintf(format, args...))
}

func (l *SyncLogger) Fatal(format string, args ...interface{}) {
	l.log(config.FATAL, fmt.Sprintf(format, args...))
	os.Exit(1) // Fatal级别：输出日志后退出程序
}

func (l *SyncLogger) Panic(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.log(config.PANIC, msg)
	panic(msg) // Panic级别：输出日志后抛出panic
}
func (l *SyncLogger) Field(fields map[string]interface{}) *SyncLogger {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// 合并扩展字段（覆盖已有字段）
	for k, v := range fields {
		l.fields[k] = v
	}
	return l
}

// Caller 设置调用栈跳过层级（支持链式调用）
func (l *SyncLogger) Caller(skip int) *SyncLogger {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.skip = skip
	return l
}

// SetLevel 设置日志级别
func (l *SyncLogger) SetLevel(level config.LogLevel) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.config.Level = level
}

func (l *SyncLogger) SetShowCaller(show bool) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	format := core.TextFormatter{
		EnableColor:     config.DefaultEnableColor,
		TimestampFormat: time.DateTime,
		ShowCaller:      show,
	}
	l.formatter = &format
}

// SetFormatter 设置格式化器（支持动态切换，如JSONFormatter）
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
func (l *SyncLogger) log(level config.LogLevel, message string) {
	if level < l.config.Level {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	entry := core.Entry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Fields:    l.fields, // 传递扩展字段
		Skip:      l.skip,   // 传递调用栈跳过层级
		Model:     l.model,  // 传递模型名
	}

	l.fields = make(map[string]interface{})
	l.skip = 4 // 重置为默认skip

	skip, errs := l.hook.RunHooks(hooks.StageBeforeFormat, level, &entry)
	if errs != nil {
		fmt.Printf("日志钩子执行失败：%v\n", errs)
		return
	}
	if skip {
		return // 跳过后续所有流程
	}

	logBytes, err := l.formatter.Format(&entry)
	if err != nil {
		fmt.Printf("日志格式化失败：%v\n", err)
		return
	}
	skip, errs = l.hook.RunHooks(hooks.StageAfterFormat, level, &entry)
	if len(errs) > 0 {
		fmt.Printf("【钩子错误】格式化后：%v\n", errs)
	}
	if skip {
		return
	}

	skip, errs = l.hook.RunHooks(hooks.StageBeforeWrite, level, &entry)
	if len(errs) > 0 {
		fmt.Printf("【钩子错误】写入前：%v\n", errs)
	}
	if skip {
		return
	}
	// 5. 输出到控制台
	if l.config.OutputConsole {
		// 错误日志输出到stderr，其他到stdout
		if level >= config.ERROR {
			_, _ = os.Stderr.Write(logBytes)
		} else {
			_, _ = os.Stdout.Write(logBytes)
		}
	}
	skip, errs = l.hook.RunHooks(hooks.StageAfterWrite, level, &entry)
	if len(errs) > 0 {
		fmt.Printf("【钩子错误】写入后：%v\n", errs)
	}
}

// Close 关闭日志器（刷盘+关闭文件句柄）
func (l *SyncLogger) Close() {

}
