package LogX

import (
	config2 "LogX/config"
	"LogX/core"
	"LogX/hooks"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

type AsyncLogger struct {
	config        config2.LoggerConfig
	model         string
	logChan       chan *core.Entry   // 日志通道（核心，必须显式关闭）
	entryPool     sync.Pool          // Entry对象池，避免频繁GC
	formatter     core.Formatter     // 日志格式化器
	hook          *hooks.HookManager // 日志钩子
	mutex         sync.RWMutex       // 配置读写锁（读多写少）
	wg            sync.WaitGroup     // 消费者协程等待组
	consumers     int                // 当前消费者数量
	maxConsumers  int                // 最大消费者数量
	consumerMutex sync.RWMutex       // 消费者数量读写锁（严格遵循调用规范）
	isClosed      bool               // 日志器关闭标记
	consoleMu     sync.Mutex         // 控制台全局写锁，避免多消费者IO竞争
	errorOut      *os.File           // 错误日志输出
}

type asyncContext struct {
	logger *AsyncLogger
	fields map[string]interface{}
	skip   int
}

func NewDefaultAsyncLogger(model string) *AsyncLogger {
	if model == "" {
		model = "default"
	}
	defaultConfig := config2.NewDefaultLoggerConfig()
	defaultFormatter := core.TextFormatter{
		EnableColor:     config2.DefaultEnableColor,
		TimestampFormat: time.DateTime,
		ShowCaller:      false,
	}
	l := &AsyncLogger{
		config:  defaultConfig,
		model:   model,
		logChan: make(chan *core.Entry, 1024), // 1024缓冲区，可根据业务调整
		entryPool: sync.Pool{
			New: func() interface{} {
				return core.NewEntry()
			},
		},
		formatter:    &defaultFormatter,
		hook:         hooks.NewHookManager(),
		maxConsumers: 16, // 最大8个消费者，可根据CPU核心数调整
		consumers:    3,
		isClosed:     false,
		consoleMu:    sync.Mutex{}, // 初始化控制台锁，避免panic
		errorOut:     os.Stderr,    // 默认错误日志输出到标准错误
	}
	// 启动3个初始消费者协程
	for i := 0; i < 3; i++ {
		l.startConsumer()
	}

	return l
}
func NewAsyncLogger(model string, conf config2.LC) *AsyncLogger {
	con := config2.ParseLoggerConfigFromJSON(conf)
	logger := &AsyncLogger{
		config:  con,
		model:   model,
		logChan: make(chan *core.Entry, 1024), // 1024缓冲区，可根据业务调整
		entryPool: sync.Pool{
			New: func() interface{} {
				return core.NewEntry()
			},
		},
		formatter: &core.TextFormatter{
			EnableColor:     config2.DefaultEnableColor,
			TimestampFormat: time.DateTime,
			ShowCaller:      con.ShowCaller,
		},
		hook:         hooks.NewHookManager(),
		maxConsumers: 16, // 最大8个消费者，可根据CPU核心数调整
		consumers:    3,
		isClosed:     false,
		consoleMu:    sync.Mutex{}, // 初始化控制台锁，避免panic
		errorOut:     os.Stderr,    // 默认错误日志输出到标准错误
	}
	// 启动3个初始消费者协程
	for i := 0; i < 3; i++ {
		logger.startConsumer()
	}
	return logger

}

func (l *asyncContext) Caller(skip int) core.LoggerContext {
	l.skip = skip
	return l
}

func (l *asyncContext) Trace(format string, args ...interface{}) {
	l.logger.output(config2.TRACE, fmt.Sprintf(format, args...), l.fields, l.skip)
}
func (l *asyncContext) Debug(format string, args ...interface{}) {
	l.logger.output(config2.DEBUG, fmt.Sprintf(format, args...), l.fields, l.skip)
}
func (l *asyncContext) Info(format string, args ...interface{}) {
	l.logger.output(config2.INFO, fmt.Sprintf(format, args...), l.fields, l.skip)
}
func (l *asyncContext) Warn(format string, args ...interface{}) {
	l.logger.output(config2.WARNING, fmt.Sprintf(format, args...), l.fields, l.skip)
}
func (l *asyncContext) Error(format string, args ...interface{}) error {
	l.logger.output(config2.ERROR, fmt.Sprintf(format, args...), l.fields, l.skip)
	return errors.New(fmt.Sprintf(format, args...))
}
func (l *asyncContext) Fatal(format string, args ...interface{}) {
	l.logger.output(config2.FATAL, fmt.Sprintf(format, args...), l.fields, l.skip)
	os.Exit(1)
}
func (l *asyncContext) Panic(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.logger.output(config2.PANIC, msg, l.fields, l.skip)
	panic(msg)
}

// 基础日志方法：保留原有逻辑
func (l *AsyncLogger) Trace(format string, args ...interface{}) { l.Field(nil).Trace(format, args...) }
func (l *AsyncLogger) Debug(format string, args ...interface{}) { l.Field(nil).Debug(format, args...) }
func (l *AsyncLogger) Info(format string, args ...interface{})  { l.Field(nil).Info(format, args...) }
func (l *AsyncLogger) Warn(format string, args ...interface{})  { l.Field(nil).Warn(format, args...) }
func (l *AsyncLogger) Error(format string, args ...interface{}) error {
	return l.Field(nil).Error(format, args...)
}
func (l *AsyncLogger) Fatal(format string, args ...interface{}) { l.Field(nil).Fatal(format, args...) }
func (l *AsyncLogger) Panic(format string, args ...interface{}) { l.Field(nil).Panic(format, args...) }

func (l *AsyncLogger) Field(fields map[string]any) core.LoggerContext {
	ctx := &asyncContext{
		logger: l,
		fields: make(map[string]any, len(fields)),
		skip:   4,
	}
	for k, v := range fields {
		ctx.fields[k] = v
	}
	return ctx
}

// SetLevel 设置日志级别（写锁保护，线程安全）
func (l *AsyncLogger) SetLevel(level config2.LogLevel) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.config.Level = level
}

// SetFormatter 设置格式化器（写锁保护，线程安全）
func (l *AsyncLogger) SetFormatter(formatter core.Formatter) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.formatter = formatter
}

// AddHook 添加日志钩子（写锁保护，线程安全）
func (l *AsyncLogger) AddHook(hook hooks.HookBase) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.hook.AddHook(hook)
}

// output 日志生产核心方法（读锁保护，避免无效生产，资源泄漏防护）
func (l *AsyncLogger) output(level config2.LogLevel, message string, fields map[string]interface{}, skip int) {
	// 读锁检查：关闭状态/日志级别，快速返回，避免无效操作
	l.mutex.RLock()
	closed := l.isClosed
	configLevel := l.config.Level
	l.mutex.RUnlock()

	if closed || level < configLevel {
		return
	}

	// 从对象池获取Entry，兜底创建
	entry, ok := l.entryPool.Get().(*core.Entry)
	if !ok {
		entry = core.NewEntry()
	}
	entry.SetEntry(time.Now(), level, message, l.model, skip, fields)

	// 发送成功标记，延迟归还Entry，避免资源泄漏
	sentSuccess := false
	defer func() {
		if !sentSuccess {
			l.entryPool.Put(entry)
		}
	}()

	// 带超时的通道发送，避免生产协程阻塞
	select {
	case l.logChan <- entry:
		sentSuccess = true
		// 检测是否需要动态扩容消费者
		if l.shouldScaleUp() {
			l.startConsumer()
		}
	case <-time.After(50 * time.Millisecond):
		// 队列满，尝试强制扩容
		l.scaleUpIfNecessary()
		// 再次尝试发送
		select {
		case l.logChan <- entry:
			sentSuccess = true
		default:
			// 队列依然满，丢弃日志，避免阻塞
			_, _ = l.errorOut.WriteString("[丢弃日志] " + message)
		}
	}
}

// startConsumer 启动消费者协程（修复锁调用规范，防panic/防锁泄漏）
func (l *AsyncLogger) startConsumer() {
	l.consumerMutex.Lock()
	defer l.consumerMutex.Unlock() // 提前defer解锁，防止校验panic导致锁泄漏

	// 严格校验：已关闭/达到最大消费者数，直接返回
	if l.isClosed || l.consumers >= l.maxConsumers {
		return
	}

	l.consumers++
	l.wg.Add(1)

	// 启动消费者协程，传入消费者ID便于日志排查
	go func(consumerID int) {
		// 必加：panic恢复，保证锁释放和wg.Done执行
		defer func() {
			if err := recover(); err != nil {
				_, _ = l.errorOut.WriteString("[消费者协程panic] " + fmt.Sprintf("ID：%d，错误：%v\n", consumerID, err))
			}
			// 核心修复：Lock() 严格对应 Unlock()，解决RUnlock未加锁错误
			l.consumerMutex.Lock()
			l.consumers--
			l.consumerMutex.Unlock() // 替换原错误的RUnlock()
			l.wg.Done()

		}()

		// 通道关闭后，for range消费完剩余日志自动退出
		for entry := range l.logChan {
			l.processEntry(entry)
			entry.Fields = make(map[string]interface{}, 4) // 重置Entry，避免旧数据污染
			l.entryPool.Put(entry)
		}
	}(l.consumers)
}

// processEntry 日志消费核心方法（提升效率，IO竞争防护，错误处理）
func (l *AsyncLogger) processEntry(entry *core.Entry) {
	// 格式化前钩子：处理跳过标记，避免无效操作
	skipHook, errs := l.hook.RunHooks(hooks.StageBeforeFormat, entry.Level, entry)
	if len(errs) > 0 {
		_, _ = l.errorOut.WriteString(fmt.Sprintf("[%s] 【异步日志】格式化前钩子错误：%v\n", time.Now().Format("2006-01-02 15:04:05"), errs))
	}
	if skipHook {
		return
	}

	// 格式化日志：处理错误，避免卡住消费协程
	logBytes, err := l.formatter.Format(entry)
	if err != nil {
		_, _ = l.errorOut.WriteString(fmt.Sprintf("[%s] 【异步日志】格式化失败：%v\n", time.Now().Format("2006-01-02 15:04:05"), err))
		return
	}

	// 格式化后钩子：再次过滤无效操作
	skipHook, errs = l.hook.RunHooks(hooks.StageAfterFormat, entry.Level, entry)
	if len(errs) > 0 {
		_, _ = l.errorOut.WriteString(fmt.Sprintf("[%s] 【异步日志】格式化后钩子错误：%v\n", time.Now().Format("2006-01-02 15:04:05"), errs))
	}
	if skipHook {
		return
	}

	//控制台输出：全局锁防护，避免多消费者IO竞争
	if l.config.OutputConsole {
		l.consoleMu.Lock()
		defer l.consoleMu.Unlock()
		if entry.Level >= config2.ERROR {
			if _, err := os.Stderr.Write(logBytes); err != nil {
				_, _ = l.errorOut.WriteString(fmt.Sprintf("[%s] 【异步日志】标准错误输出失败：%v\n", time.Now().Format("2006-01-02 15:04:05"), err))
			}
		} else {
			if _, err := os.Stdout.Write(logBytes); err != nil {
				_, _ = l.errorOut.WriteString(fmt.Sprintf("[%s] 【异步日志】标准输出失败：%v\n", time.Now().Format("2006-01-02 15:04:05"), err))
			}
		}
	}

	// 后续钩子：保留逻辑，增加错误打印
	if _, errs := l.hook.RunHooks(hooks.StageBeforeWrite, entry.Level, entry); len(errs) > 0 {
		_, _ = l.errorOut.WriteString(fmt.Sprintf("[%s] 【异步日志】写入前钩子错误：%v\n", time.Now().Format("2006-01-02 15:04:05"), errs))
	}
	if _, errs := l.hook.RunHooks(hooks.StageAfterWrite, entry.Level, entry); len(errs) > 0 {
		_, _ = l.errorOut.WriteString(fmt.Sprintf("[%s] 【异步日志】写入后钩子错误：%v\n", time.Now().Format("2006-01-02 15:04:05"), errs))
	}
}

// shouldScaleUp 判断是否需要扩容消费者（阈值50%，提前触发，避免队列满）
func (l *AsyncLogger) shouldScaleUp() bool {
	// 读锁查询消费者数（无修改，只读）
	l.consumerMutex.RLock()
	currentConsumers := l.consumers
	l.consumerMutex.RUnlock()

	// 读锁检查关闭状态
	l.mutex.RLock()
	closed := l.isClosed
	l.mutex.RUnlock()

	// 已关闭/达到最大数，不扩容
	if closed || currentConsumers >= l.maxConsumers {
		return false
	}

	// 队列使用率超过50%，触发扩容
	queueCapacity := cap(l.logChan)
	queueLength := len(l.logChan)
	return float64(queueLength)/float64(queueCapacity) > 0.5
}

// scaleUpIfNecessary 强制扩容消费者（队列满时调用）
func (l *AsyncLogger) scaleUpIfNecessary() {
	l.consumerMutex.RLock()
	currentConsumers := l.consumers
	l.consumerMutex.RUnlock()

	if currentConsumers < l.maxConsumers {
		l.startConsumer()
	}
}

// Close 优雅关闭日志器（无死锁，消费完所有剩余日志，等待消费者退出）
func (l *AsyncLogger) Close() {
	// 第一步：加写锁，快速执行标记关闭+关闭通道，立即释放写锁
	l.mutex.Lock()
	if l.isClosed {
		l.mutex.Unlock()
		return
	}
	l.isClosed = true
	close(l.logChan)
	l.mutex.Unlock() // 立即释放，避免阻塞生产协程的读锁
	l.wg.Wait()

}
