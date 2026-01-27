package LogX

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

type SyncLogger struct {
	config      LoggerConfig
	model       string
	file        *os.File
	writer      *bufio.Writer
	mutex       sync.Mutex
	colors      map[LogLevel]string
	sigCh       chan interface{}
	wg          sync.WaitGroup
	stopCh      chan os.Signal
	autoCloseCh chan struct{}
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewDefaultSyncLogger(model string) (*SyncLogger, error) {
	if model == "" {
		model = "default"
	}
	ctx, cancel := context.WithCancel(context.Background())
	logger := &SyncLogger{
		config:      NewDefaultLoggerConfig(),
		model:       model,
		file:        nil,
		writer:      nil,
		colors:      levelColors,
		sigCh:       make(chan interface{}),
		stopCh:      make(chan os.Signal, 1),
		autoCloseCh: make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
	}

	runtime.AddCleanup(logger, func(l *SyncLogger) {
		l.cancel() // 触发context取消
		l.Close()  // 自动刷盘+关文件
	}, nil)

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
	ctx, cancel := context.WithCancel(context.Background())

	logger := &SyncLogger{
		config:      parseLoggerConfigFromJSON(config),
		model:       model,
		file:        nil,
		writer:      nil,
		colors:      levelColors,
		sigCh:       make(chan interface{}),
		stopCh:      make(chan os.Signal, 1),
		autoCloseCh: make(chan struct{}), // 初始化自动关闭通道
		ctx:         ctx,
		cancel:      cancel,
	}

	runtime.AddCleanup(logger, func(l *SyncLogger) {
		l.cancel()
		l.Close()
	}, nil)

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
	if !(level >= l.config.Level) {
		return
	}

	// 先写入控制台
	if l.config.OutputConsole {
		logEntry := l.formatLogColorEntry(level, format, args...)
		_, err := os.Stdout.WriteString(logEntry)
		if err != nil {
			fmt.Printf("控制台写入失败：%v\n", err)
		}
	}

	// 再写入文件
	if l.config.OutputFile {
		logEntry := l.formatLogEntry(level, format, args...)
		lnLog := len(logEntry)

		l.mutex.Lock()
		defer l.mutex.Unlock() // FIX: 统一用defer释放锁，避免锁泄漏

		if lnLog >= 2*l.config.BufferSize {
			_ = l.writer.Flush()
			_, err := l.writer.WriteString(logEntry)
			if err != nil {
				log.Printf("[LogX] 大日志写入失败：%v，执行刷盘", err)
				l.Sync()
				return
			}
		} else {
			_, err := l.writer.WriteString(logEntry)
			if err != nil {
				_ = l.writer.Flush()
				log.Printf("[LogX] 日志写入失败：%v，执行刷盘", err)
				l.Sync()
				return
			}

			// 水位刷盘（保留你的逻辑）
			buffered := l.writer.Buffered()
			threshold := int(float64(l.config.BufferSize) * 0.8)
			if buffered >= threshold {
				err = l.writer.Flush()
				if err != nil {
					log.Printf("[LogX] 水位刷盘失败：%v", err)
					return
				}
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

// 启动守护协程（处理退出信号和自动关闭）
func (l *SyncLogger) startDaemon() {
	signalChan := make(chan os.Signal, 3)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer signal.Stop(signalChan)
	defer close(signalChan)

	flushInterval := l.config.FlushInterval
	if flushInterval <= 0 {
		flushInterval = 5 * time.Second
	}
	flushTicker := time.NewTicker(flushInterval)
	defer flushTicker.Stop()

	l.wg.Add(1)
	go func() {
		defer l.wg.Done() // 协程退出释放wg
		defer func() {
			if r := recover(); r != nil {
				log.Printf("守护协程崩溃：%v，执行兜底刷盘", r)
				l.safeSync()
			}
		}()

		// ========== 核心修改：监听ctx.Done()，自动退出循环 ==========
		for {
			select {
			case <-l.ctx.Done(): // 感知到退出信号，自动退出
				log.Printf("[LogX] 守护协程自动退出（业务已结束）")
				l.Sync() // 退出前最后刷一次盘
				return

			case sig := <-signalChan: // 兼容Ctrl+C手动退出
				log.Printf("[LogX] 收到退出信号%v，刷盘后退出", sig)
				for i := 0; i < 3; i++ {
					l.Sync()
					time.Sleep(l.config.ExitSyncDelay)
					l.mutex.Lock()
					if l.writer.Buffered() == 0 {
						l.mutex.Unlock()
						break
					}
					l.mutex.Unlock()
				}
				l.Close()
				os.Exit(0)

			case <-flushTicker.C: // 定时刷盘（业务运行中正常执行）
				l.safeSync()
			}
		}
	}()
}

// 强制刷盘（原有逻辑）
func (l *SyncLogger) Sync() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.writer != nil {
		// 1. 强制刷空缓冲区
		if err := l.writer.Flush(); err != nil {
			log.Printf("刷盘失败：%v", err)
		}
		// 2. 检查缓冲区是否真的空了（核心：避免残留）
		if l.writer.Buffered() > 0 {
			_ = l.writer.Flush()
		}
	}

	if l.file != nil {
		// 3. 强制内核写入物理磁盘
		if err := l.file.Sync(); err != nil {
			log.Printf("文件同步失败：%v", err)
		}
	}
}

// 新增：安全刷盘（捕获panic）
func (l *SyncLogger) safeSync() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("安全刷盘触发panic：%v", r)
		}
	}()
	l.Sync()
}
func (l *SyncLogger) safeSyncWithErr() error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("安全刷盘触发panic：%v", r)
		}
	}()
	l.mutex.Lock()
	defer l.mutex.Unlock()

	var err error
	if l.writer != nil {
		if e := l.writer.Flush(); e != nil {
			err = e
		}
	}
	if l.file != nil {
		if e := l.file.Sync(); e != nil && err == nil {
			err = e
		}
	}
	return err
}

// 关闭日志器（手动调用或自动调用）
func (l *SyncLogger) Close() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// 刷盘+关文件（自动退出时也会执行）
	if l.writer != nil {
		_ = l.writer.Flush()
		_ = l.writer.Flush()
	}
	if l.file != nil {
		_ = l.file.Sync()
		_ = l.file.Close()
	}
	log.Printf("[LogX] 日志器自动刷盘关闭")
}
