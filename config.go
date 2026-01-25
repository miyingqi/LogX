package LogX

import (
	"fmt"
	"os"
	"path/filepath"
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

// LoggerConfig 日志配置
type LoggerConfig struct {
	Dir           string
	FileName      string
	MaxFileSize   int64
	MaxBackups    int
	Level         LogLevel
	BufferSize    int
	EnableColor   bool
	OutputFile    bool
	OutputConsole bool
}

// DefaultConfig 默认配置
var DefaultConfig = LoggerConfig{
	Level:         INFO,
	Dir:           "logs",
	FileName:      "app.log",
	BufferSize:    1000,
	MaxBackups:    1,
	MaxFileSize:   100,
	EnableColor:   true,
	OutputFile:    true,
	OutputConsole: true,
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
