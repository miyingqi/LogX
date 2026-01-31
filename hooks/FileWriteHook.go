package hooks

import (
	"LogX/config"
	"LogX/core"
	"os"
	"sync"
)

// FileWriteHook 文件写入钩子
type FileWriteHook struct {
	filePath string
	file     *os.File
	mu       sync.Mutex
}

// NewFileWriteHook 创建一个新的文件写入钩子
func NewFileWriteHook(filePath string) *FileWriteHook {
	return &FileWriteHook{
		filePath: filePath,
	}
}

// Fire 实现钩子核心逻辑：将日志写入文件
func (f *FileWriteHook) Fire(entry *core.Entry, stage HookStage) (bool, error) {
	// 只在写入后阶段执行
	if stage != StageAfterWrite {
		return false, nil
	}

	// 确保文件已打开
	if err := f.ensureFile(); err != nil {
		return false, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// 将日志格式化为字符串
	logEntry := entry.Timestamp.Format("2006-01-02 15:04:05.000") + " [" +
		config.LevelStrings[entry.Level] + "] " +
		"(" + entry.Model + ") - " +
		entry.Message

	// 添加扩展字段
	if len(entry.Fields) > 0 {
		for k, v := range entry.Fields {
			logEntry += " | " + k + "=" + formatValue(v)
		}
	}

	// 写入文件并换行
	_, err := f.file.WriteString(logEntry + "\n")
	return false, err
}

// Levels 支持所有日志级别
func (f *FileWriteHook) Levels() []config.LogLevel {
	return []config.LogLevel{
		config.TRACE, config.DEBUG, config.INFO,
		config.WARNING, config.ERROR, config.PANIC, config.FATAL,
	}
}

// Stages 绑定到「写入后」阶段
func (f *FileWriteHook) Stages() []HookStage {
	return []HookStage{StageAfterWrite}
}

// Close 关闭文件句柄
func (f *FileWriteHook) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.file != nil {
		return f.file.Close()
	}
	return nil
}

// ensureFile 确保文件已打开
func (f *FileWriteHook) ensureFile() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.file == nil {
		file, err := os.OpenFile(f.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
		f.file = file
	}
	return nil
}

// formatValue 格式化字段值
func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case error:
		return val.Error()
	default:
		return ""
	}
}
