package LogX

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type LogLevel uint8

const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
	FATAL
)

var levelStrings = map[LogLevel]string{
	DEBUG:   "DEBUG",
	INFO:    "INFO",
	WARNING: "WARN",
	ERROR:   "ERROR",
	FATAL:   "FATAL"}

const (
	ColorReset   = "\033[0m"
	ColorGray    = "\033[90m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorCyan    = "\033[36m"
	ColorBoldRed = "\033[1;31m"
	ColorWhite   = "\033[37m"
)

var levelColors = map[LogLevel]string{
	DEBUG:   ColorCyan,
	INFO:    ColorGreen,
	WARNING: ColorYellow,
	ERROR:   ColorRed,
	FATAL:   ColorBoldRed,
}

const (
	DefaultDir           = "./logs"
	DefaultFileName      = "app.log"
	DefaultMaxFileSize   = 100 * 1024 * 1024 // 100MB
	DefaultMaxBackups    = 5                 // 最多保留5个备份文件
	DefaultLevel         = INFO              // 默认日志级别
	DefaultBufferSize    = 4096              // 4KB缓冲区
	DefaultEnableColor   = true              // 默认启用控制台颜色
	DefaultOutputFile    = true              // 默认输出到文件
	DefaultOutputConsole = true              // 默认输出到控制台
	DefaultFlushInterval = 5 * time.Second
	DefaultExitSyncDelay = 200 * time.Millisecond
)

// LC 是 LoggerConfig 的配置映射，支持以下字段：
// - dir: string - 日志目录路径
// - file_name: string - 日志文件名
// - max_file_size: float64 - 最大文件大小(字节)
// - max_backups: float64 - 最大备份文件数
// - level: LogLevel - 日志级别 (0=DEBUG, 1=INFO, 2=WARNING, 3=ERROR, 4=FATAL)
// - buffer_size: float64 - 缓冲区大小
// - enable_color: bool - 是否启用颜色
// - output_file: bool - 是否输出到文件
// - output_console: bool - 是否输出到控制台
type LC map[string]interface{}

// LoggerConfig 日志配置
type LoggerConfig struct {
	Dir           string        `json:"dir,omitempty"`            // 可选，默认./logs
	FileName      string        `json:"file_name,omitempty"`      // 可选，默认app.log
	MaxFileSize   int64         `json:"max_file_size,omitempty"`  // 可选，默认100MB
	MaxBackups    int           `json:"max_backups,omitempty"`    // 可选，默认5
	Level         LogLevel      `json:"level,omitempty"`          // 可选，默认INFO
	BufferSize    int           `json:"buffer_size,omitempty"`    // 可选，默认4096
	EnableColor   bool          `json:"enable_color,omitempty"`   // 可选，默认true
	OutputFile    bool          `json:"output_file,omitempty"`    // 可选，默认true
	OutputConsole bool          `json:"output_console,omitempty"` // 可选，默认true
	FlushInterval time.Duration `json:"flush_interval,omitempty"`
	ExitSyncDelay time.Duration `json:"exit_sync_delay,omitempty"`
}

// DefaultConfig 默认配置
func NewDefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Level:         DefaultLevel,
		Dir:           DefaultDir,
		FileName:      DefaultFileName,
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

// 判断文件是否存在，不存在则创建（空文件）
func ensureFileExists(filePath string, perm os.FileMode) error {
	// 获取父目录路径
	dir := filepath.Dir(filePath)

	// 确保父目录存在，如果不存在则创建
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// 检查文件是否存在
	statResult, err := os.Stat(filePath)
	if err == nil {
		// 文件存在，检查是否是目录
		if statResult.IsDir() {
			return fmt.Errorf("path exists but is a directory: %s", filePath)
		}
		return nil // 文件已存在，无需创建
	}

	// 文件不存在，创建空文件
	if os.IsNotExist(err) {
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, perm)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		defer func() {
			if closeErr := file.Close(); closeErr != nil {
				// 记录错误但不中断程序
				fmt.Fprintf(os.Stderr, "warning: failed to close file: %v\n", closeErr)
			}
		}()
		return nil
	}

	// 其他错误
	return fmt.Errorf("failed to stat file: %w", err)
}

// parseLoggerConfigFromJSON 从JSON字符串解析配置，未指定的字段使用默认值
func parseLoggerConfigFromJSON(jsonData map[string]interface{}) LoggerConfig {
	config := NewDefaultLoggerConfig()

	// 处理字符串字段
	if dir, ok := jsonData["dir"].(string); ok && dir != "" {
		config.Dir = dir
	}

	if fileName, ok := jsonData["file_name"].(string); ok && fileName != "" {
		config.FileName = fileName
	}

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

	return config
}
