package LogX

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type SyncLogger struct {
	config LoggerConfig
	model  string
	file   *os.File
	writer *bufio.Writer
	mutex  sync.Mutex
	colors map[LogLevel]string
	sigCh  chan interface{}
	wg     sync.WaitGroup
	stopCh chan os.Signal
}

func NewDefaultSyncLogger(model string) (*SyncLogger, error) {
	if model == "" {
		model = "default"
	}
	logger := &SyncLogger{
		config: NewDefaultLoggerConfig(),
		model:  model,
		file:   nil,
		writer: nil,
		colors: levelColors,
		sigCh:  make(chan interface{}),
		stopCh: make(chan os.Signal, 1),
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
	go logger.startDaemon()
	return logger, nil
}
func NewSyncLogger(model string, config map[string]interface{}) (*SyncLogger, error) {
	if model == "" {
		model = "default"
	}
	logger := &SyncLogger{
		config: parseLoggerConfigFromJSON(config),
		model:  model,
		file:   nil,
		writer: nil,
		colors: levelColors,
		sigCh:  make(chan interface{}),
		stopCh: make(chan os.Signal, 1),
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
	go logger.startDaemon()
	return logger, nil
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
func (l *SyncLogger) Log(level LogLevel, format string, args ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
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
	l.mutex.Lock()
	defer l.mutex.Unlock()
	_ = l.writer.Flush()
	l.writer = bufio.NewWriterSize(l.file, bufferSize)
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

// 启动守护协程（仅处理退出信号）
func (l *SyncLogger) startDaemon() {
	// 注册系统退出信号（SIGINT/Ctrl+C、SIGTERM/kill）
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// 启动唯一的守护协程（仅处理退出信号）
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		defer func() {
			// Panic兜底：捕获崩溃，执行最后一次刷盘
			if r := recover(); r != nil {
				fmt.Printf("[LogX] 守护协程崩溃：%v，执行兜底刷盘\n", r)
				l.Sync()
			}
		}()

		// 仅监听退出信号，无定时刷盘
		select {
		case sig := <-signalChan:
			// 收到系统退出信号：强制刷盘 + 关闭通道
			fmt.Printf("\n[LogX] 收到退出信号：%v，执行兜底刷盘...\n", sig)
			l.Sync()
			l.stopCh <- sig // 通知Close方法退出
		case <-l.stopCh:
			// 收到手动关闭信号：强制刷盘
			l.Sync()
		}
	}()
}

// 强制刷盘（原有逻辑）
func (l *SyncLogger) Sync() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.writer != nil {
		if err := l.writer.Flush(); err != nil {
			log.Println("[LogX] 刷盘失败：%v", err)
		}
	}

	if l.file != nil {
		if err := l.file.Sync(); err != nil {
			log.Println("[LogX] 文件同步失败：%v", err)
		}
	}
}

// 关闭日志器（手动调用）
func (l *SyncLogger) Close() {
	close(l.stopCh) // 通知守护协程退出
	l.wg.Wait()     // 等待守护协程完全退出

	// 最终刷盘 + 关闭文件
	l.Sync()
	if l.file != nil {
		l.file.Close()
	}
}
