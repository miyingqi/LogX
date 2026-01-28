package config

import (
	"time"
)

type LogLevel uint8

const (
	TRACE LogLevel = iota
	DEBUG
	INFO
	WARNING
	ERROR
	PANIC
	FATAL
)

var LevelStrings = map[LogLevel]string{
	TRACE:   "TRACE",
	DEBUG:   "DEBUG",
	INFO:    "INFO",
	WARNING: "WARN",
	ERROR:   "ERROR",
	PANIC:   "PANIC",
	FATAL:   "FATAL",
}

const (
	ColorReset   = "\033[0m"    // 重置颜色
	ColorWhite   = "\033[37m"   // 白色
	ColorGray    = "\033[90m"   // 灰色
	ColorRed     = "\033[31m"   // 红色
	ColorGreen   = "\033[32m"   // 绿色
	ColorYellow  = "\033[33m"   // 黄色
	ColorCyan    = "\033[36m"   // 青色
	ColorBoldRed = "\033[1;31m" // 加粗红色
)

var LevelColors = map[LogLevel]string{
	TRACE:   ColorGray,    // 灰色 - 最详细的追踪信息
	DEBUG:   ColorCyan,    // 青色 - 调试信息
	INFO:    ColorGreen,   // 绿色 - 一般信息
	WARNING: ColorYellow,  // 黄色 - 警告信息
	ERROR:   ColorRed,     // 红色 - 错误信息
	FATAL:   ColorBoldRed, // 加粗红色 - 致命错误
}

const (
	DefaultMaxFileSize   = 100 * 1024 * 1024 // 100MB
	DefaultMaxBackups    = 5                 // 最多保留5个备份文件
	DefaultLevel         = INFO              // 默认日志级别
	DefaultBufferSize    = 4096              // 4KB缓冲区
	DefaultEnableColor   = true              // 默认启用控制台颜色
	DefaultOutputFile    = false             // 默认输出到文件
	DefaultOutputConsole = true              // 默认输出到控制台
	DefaultFlushInterval = 5 * time.Second
	DefaultExitSyncDelay = 200 * time.Millisecond
)

// LC 是 LoggerConfig 的配置映射，支持以下字段：
// - max_file_size: float64 - 最大文件大小(字节)
// - max_backups: float64 - 最大备份文件数
// - level: LogLevel - 日志级别 (0=DEBUG, 1=INFO, 2=WARNING, 3=ERROR, 4=FATAL)
// - buffer_size: float64 - 缓冲区大小
// - enable_color: bool - 是否启用颜色
// - output_file: bool - 是否输出到文件
// - output_console: bool - 是否输出到控制台
// - show_caller: bool - 是否显示调用方
type LC map[string]interface{}

// LoggerConfig 日志配置
type LoggerConfig struct {
	MaxFileSize   int64         `json:"max_file_size,omitempty"`  // 可选，默认100MB
	MaxBackups    int           `json:"max_backups,omitempty"`    // 可选，默认5
	Level         LogLevel      `json:"level,omitempty"`          // 可选，默认INFO
	BufferSize    int           `json:"buffer_size,omitempty"`    // 可选，默认4096
	EnableColor   bool          `json:"enable_color,omitempty"`   // 可选，默认true
	OutputFile    bool          `json:"output_file,omitempty"`    // 可选，默认true
	OutputConsole bool          `json:"output_console,omitempty"` // 可选，默认true
	ShowCaller    bool          `json:"show_caller,omitempty"`
	FlushInterval time.Duration `json:"flush_interval,omitempty"`
	ExitSyncDelay time.Duration `json:"exit_sync_delay,omitempty"`
}

// DefaultConfig 默认配置
func NewDefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Level:         DefaultLevel,
		BufferSize:    DefaultBufferSize,
		MaxBackups:    DefaultMaxBackups,
		MaxFileSize:   DefaultMaxFileSize,
		EnableColor:   DefaultEnableColor,
		OutputFile:    DefaultOutputFile,
		OutputConsole: DefaultOutputConsole,
		FlushInterval: DefaultFlushInterval,
		ExitSyncDelay: DefaultExitSyncDelay,
	}
}

// parseLoggerConfigFromJSON 从JSON字符串解析配置，未指定的字段使用默认值
func ParseLoggerConfigFromJSON(jsonData map[string]interface{}) LoggerConfig {
	config := NewDefaultLoggerConfig()
	// 处理数值字段
	if maxFileSize, ok := jsonData["max_file_size"].(float64); ok {
		config.MaxFileSize = int64(maxFileSize)
	}

	if maxBackups, ok := jsonData["max_backups"].(float64); ok {
		config.MaxBackups = int(maxBackups)
	}

	if level, ok := jsonData["level"].(LogLevel); ok {
		config.Level = level
	}

	if bufferSize, ok := jsonData["buffer_size"].(float64); ok {
		config.BufferSize = int(bufferSize)
	}

	// 处理布尔字段
	if enableColor, ok := jsonData["enable_color"].(bool); ok {
		config.EnableColor = enableColor
	}

	if outputFile, ok := jsonData["output_file"].(bool); ok {
		config.OutputFile = outputFile
	}

	if outputConsole, ok := jsonData["output_console"].(bool); ok {
		config.OutputConsole = outputConsole
	}
	if showCaller, ok := jsonData["show_caller"].(bool); ok {
		config.ShowCaller = showCaller
	}

	return config
}
