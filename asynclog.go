package LogX

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"time"
)

type AsyncLogger struct {
	config    LoggerConfig
	model     string
	colors    map[LogLevel]string
	file      *os.File
	writer    *bufio.Writer
	conWriter *bufio.Writer
	mutex     sync.Mutex
	conMutex  sync.Mutex
	stopCh    chan interface{}
	wg        sync.WaitGroup
	fileChan  chan string
	conChan   chan string
	stopChan  chan interface{}
}

func NewDefaultAsyncLogger(model string) (*AsyncLogger, error) {
	if model == "" {
		model = "default"
	}
	logger := &AsyncLogger{
		config:    NewDefaultLoggerConfig(),
		model:     model,
		colors:    levelColors,
		conWriter: bufio.NewWriterSize(os.Stdout, 4096),
		stopCh:    make(chan interface{}),
		fileChan:  make(chan string, 1000),
		conChan:   make(chan string, 1000),
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
	logger.wg.Add(2)
	go logger.writeToFile()
	go logger.writeToConsole()
	return logger, nil
}

func (l *AsyncLogger) Debug(format string, a ...interface{}) {
	l.log(DEBUG, format, a...)
}
func (l *AsyncLogger) Info(format string, a ...interface{}) {
	l.log(INFO, format, a...)
}
func (l *AsyncLogger) Warn(format string, a ...interface{}) {
	l.log(WARNING, format, a...)
}
func (l *AsyncLogger) Error(format string, a ...interface{}) {
	l.log(ERROR, format, a...)
}
func (l *AsyncLogger) Fatal(format string, a ...interface{}) {
	l.log(FATAL, format, a...)
}
func (l *AsyncLogger) SetLevel(level LogLevel) {
	l.config.Level = level
}
func (l *AsyncLogger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.config.Level {
		return
	}

	// 非阻塞发送到 fileChan
	select {
	case l.fileChan <- l.formatLogEntry(level, format, args...):
	default:
		// 如果 channel 满了，则丢弃日志
		fmt.Println("fileChan is full, dropping log entry")
	}

	// 非阻塞发送到 conChan
	select {
	case l.conChan <- l.formatLogColorEntry(level, format, args...):
	default:
		// 如果 channel 满了，则丢弃日志
		fmt.Println("conChan is full, dropping log entry")
	}
}

func (l *AsyncLogger) formatLogEntry(level LogLevel, format string, args ...interface{}) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	logLevelStr := levelStrings[level]
	programId := l.model
	logMessage := fmt.Sprintf(format, args...)

	return fmt.Sprintf("{%s} [%s] (%s)  - %s \n",
		timestamp, logLevelStr, programId, logMessage)
}

func (l *AsyncLogger) formatLogColorEntry(level LogLevel, format string, args ...interface{}) string {
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

// 优化 writeToFile：批量刷盘 + 修复水位判断
func (l *AsyncLogger) writeToFile() {
	defer l.wg.Done()
	for {
		select {
		case <-l.stopChan:
			// 退出前刷空缓冲区
			l.mutex.Lock()
			_ = l.writer.Flush()
			l.mutex.Unlock()
			return
		case logEntry := <-l.fileChan:
			l.mutex.Lock()
			_, err := l.writer.WriteString(logEntry)
			if err != nil {
				fmt.Printf("文件写入失败：%v\n", err)
				l.mutex.Unlock()
				continue // 错误时不终止协程，继续处理下一条
			}
			// 修复水位判断：用浮点除法，80%水位时刷盘
			if float64(l.writer.Buffered())/float64(l.config.BufferSize) >= 0.8 {
				_ = l.writer.Flush()
			}
			l.mutex.Unlock()
		}
	}
}

// 优化 writeToConsole：定时刷盘 + 修复逻辑
func (l *AsyncLogger) writeToConsole() {
	defer l.wg.Done()
	ticker := time.NewTicker(100 * time.Millisecond) // 每100ms强制刷盘
	defer ticker.Stop()

	for {
		select {
		case <-l.stopChan:
			// 退出前刷空控制台缓冲区
			l.conMutex.Lock()
			_ = l.conWriter.Flush()
			l.conMutex.Unlock()
			return
		case logEntry := <-l.conChan:
			l.conMutex.Lock()
			_, err := l.conWriter.WriteString(logEntry)
			if err != nil {
				fmt.Printf("控制台写入失败：%v\n", err)
				l.conMutex.Unlock()
				continue
			}
			// 控制台缓冲区80%水位时刷盘
			if float64(l.conWriter.Buffered())/float64(4096) >= 0.8 {
				_ = l.conWriter.Flush()
			}
			l.conMutex.Unlock()
		case <-ticker.C:
			// 定时刷盘，避免少量日志积压
			l.conMutex.Lock()
			_ = l.conWriter.Flush()
			l.conMutex.Unlock()
		}
	}
}

// 新增：优雅关闭日志器
func (l *AsyncLogger) Close() {
	// 发送退出信号
	close(l.stopChan)
	// 等待消费协程退出
	l.wg.Wait()

	// 刷空文件缓冲区并关闭文件
	l.mutex.Lock()
	_ = l.writer.Flush()
	_ = l.file.Close()
	l.mutex.Unlock()

	// 刷空控制台缓冲区
	l.conMutex.Lock()
	_ = l.conWriter.Flush()
	l.conMutex.Unlock()

	// 关闭通道（可选）
	close(l.fileChan)
	close(l.conChan)
}
