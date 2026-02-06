package core

// 定义日志上下文接口
type LoggerContext interface {
	Caller(skip int) LoggerContext
	Trace(format string, args ...interface{})
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{}) error
	Fatal(format string, args ...interface{})
	Panic(format string, args ...interface{})
}
