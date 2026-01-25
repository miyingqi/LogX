package LogX

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

type SyncLogger struct {
	config LoggerConfig
	model  string
	file   *os.File
	writer *bufio.Writer
	metux  sync.Mutex
	colors map[LogLevel]string
}

func (l *SyncLogger) Info(format string, args ...interface{}) {
	l.Log(INFO, format, args...)
}
func (l *SyncLogger) Warn(format string, args ...interface{}) {
	l.Log(WARNING, format, args...)
}
func (l *SyncLogger) Error(format string, args ...interface{}) {
	l.Log(ERROR, format, args...)
}
func (l *SyncLogger) Debug(format string, args ...interface{}) {
	l.Log(DEBUG, format, args...)
}
func (l *SyncLogger) Fatal(format string, args ...interface{}) {
	l.Log(FATAL, format, args...)
}
func NewSyncLogger(model string) (*SyncLogger, error) {
	if model == "" {
		model = "default"
	}
	logger := &SyncLogger{
		config: DefaultConfig,
		model:  model,
		file:   nil,
		writer: nil,
		colors: levelColors,
	}

	err := ensureFileExists(logger.config.Dir+"/"+logger.config.FileName, 0644)
	if err != nil {
		return nil, err
	}
	file, err := os.OpenFile(logger.config.Dir+"/"+logger.config.FileName, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	// 只有在需要输出到文件时才创建文件 writer
	writer := bufio.NewWriterSize(file, logger.config.BufferSize)
	logger.file = file
	logger.writer = writer
	return logger, nil
}

func (l *SyncLogger) Log(level LogLevel, format string, args ...interface{}) {
	l.metux.Lock()
	defer l.metux.Unlock()
	l.writeToConsole(level, format, args...)
	l.writeToFile(level, format, args...)
}

func (l *SyncLogger) writeToConsole(level LogLevel, format string, args ...interface{}) {
	if !(level >= l.config.Level) {
		return
	}
	if l.config.OutputConsole {
		logEntry := l.formatLogColorEntry(level, format, args...)
		_, err := os.Stdout.WriteString(logEntry)
		if err != nil {
			return
		}

	}

}
func (l *SyncLogger) writeToFile(level LogLevel, format string, args ...interface{}) {
	if !(level >= l.config.Level) {
		return
	}
	if l.config.OutputFile {
		logEntry := l.formatLogEntry(level, format, args...)
		lnLog := len(logEntry)
		if lnLog >= 2*l.config.BufferSize {
			_ = l.writer.Flush()
			_, err := l.writer.WriteString(logEntry)
			if err != nil {
				log.Println("error flushing log buffer: %v", err)
				return
			}

		}
		_, err := l.writer.WriteString(logEntry)
		if err != nil {
			_ = l.writer.Flush()
			log.Println("error flushing log buffer: %v", err)
			return
		}
		if float64(l.writer.Buffered()/l.config.BufferSize) >= 0.8 {
			err = l.writer.Flush()
			if err != nil {
				log.Println("error flushing log buffer: %v", err)
				return
			}
		}
	}

}

func (l *SyncLogger) formatLogEntry(level LogLevel, format string, args ...interface{}) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	logLevelStr := levelStrings[level]
	programId := l.model
	logMessage := fmt.Sprintf(format, args...)

	return fmt.Sprintf("{%s} [%s] (%s)  - %s \n",
		timestamp, logLevelStr, programId, logMessage)
}

func (l *SyncLogger) formatLogColorEntry(level LogLevel, format string, args ...interface{}) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	logLevelStr := levelStrings[level]
	programId := l.model
	logMessage := fmt.Sprintf(format, args...)
	if l.config.EnableColor {
		timestamp = ColorGray + timestamp + ColorReset
		logLevelStr = l.colors[level] + logLevelStr + ColorReset
		programId = ColorCyan + programId + ColorReset
		logMessage = ColorWhite + logMessage + ColorReset
		return fmt.Sprintf("{%s} [%s] (%s)  - %s \n",
			timestamp, logLevelStr, programId, logMessage)
	}
	return fmt.Sprintf("{%s} [%s] (%s)  - %s \n",
		timestamp, logLevelStr, programId, logMessage)
}

func (l *SyncLogger) SetLevel(level LogLevel) {
	l.config.Level = level
}
func (l *SyncLogger) SetOutputFile(outputFile bool) {
	if l.config.OutputConsole == false && outputFile == false {
		panic("至少需要输出一个日志输出方式")
		return
	}
	l.config.OutputFile = outputFile
}
func (l *SyncLogger) SetOutputConsole(outputConsole bool) {
	if l.config.OutputFile == false && outputConsole == false {
		panic("至少需要输出一个日志输出方式")
		return
	}
	l.config.OutputConsole = outputConsole
}
func (l *SyncLogger) SetBufferSize(bufferSize int) {
	l.config.BufferSize = bufferSize
}
func (l *SyncLogger) SetEnableColor(enableColor bool) {
	l.config.EnableColor = enableColor
}
func (l *SyncLogger) SetMaxBackups(maxBackups int) {
	l.config.MaxBackups = maxBackups
}
func (l *SyncLogger) SetMaxFileSize(maxFileSize int64) {
	l.config.MaxFileSize = maxFileSize
}
func (l *SyncLogger) SetDir(dir string) {
	l.config.Dir = dir
}
func (l *SyncLogger) SetFileName(fileName string) {
	l.config.FileName = fileName
}
func (l *SyncLogger) SetModel(model string) {
	l.model = model
}
func (l *SyncLogger) SetConfig(config LoggerConfig) {
	l.config = config
}
func (l *SyncLogger) Close() {
	if l.writer != nil {
		err := l.writer.Flush()
		if err != nil {
			return
		}
	}
	if l.file != nil {
		err := l.file.Close()
		if err != nil {
			return
		}
	}
}
