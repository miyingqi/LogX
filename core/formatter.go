package core

import (
	config2 "LogX/config"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Entry struct {
	Timestamp time.Time              `json:"time,omitempty"`    // 日志时间（修正拼写）
	Level     config2.LogLevel       `json:"level,omitempty"`   // 日志级别
	Message   string                 `json:"message,omitempty"` // 核心消息
	Fields    map[string]interface{} `json:"fields,omitempty"`  // 扩展字段（如trace_id、caller）
	Skip      int                    `json:"skip,omitempty"`    // 调用栈跳过层级（用于获取caller）
	Model     string                 `json:"model,omitempty"`   // 服务/模型名
}
type Formatter interface {
	Format(entry *Entry) ([]byte, error)
}

type TextFormatter struct {
	EnableColor     bool   // 核心开关：是否启用颜色
	TimestampFormat string // 时间格式（可配置，默认2006-01-02 15:04:05.000）
	ShowCaller      bool   // 是否显示调用方（文件:行号）
}

func (f *TextFormatter) Format(e *Entry) ([]byte, error) {
	// 1. 性能优化：复用Builder，预设初始容量
	var buf strings.Builder
	buf.Grow(256) // 扩容阈值提高，适配扩展字段

	// 2. 处理时间戳（带/不带颜色通用逻辑）
	timestampStr := e.Timestamp.Format(f.TimestampFormat)
	if f.EnableColor {
		timestampStr = config2.ColorGray + timestampStr + config2.ColorReset
	}
	buf.WriteString("{")
	buf.WriteString(timestampStr)
	buf.WriteString("} [")

	// 3. 处理日志级别（带/不带颜色通用逻辑）
	levelStr := config2.LevelStrings[e.Level]
	if f.EnableColor {
		levelStr = config2.LevelColors[e.Level] + levelStr + config2.ColorReset
	}
	buf.WriteString(levelStr)
	buf.WriteString("] (")

	// 4. 处理模型名（带/不带颜色通用逻辑）
	modelStr := e.Model
	if f.EnableColor {
		modelStr = config2.ColorCyan + modelStr + config2.ColorReset
	}
	buf.WriteString(modelStr)
	buf.WriteString(")")

	// 5. 处理调用方（利用Skip字段，可选显示）
	if f.ShowCaller && e.Skip >= 0 {
		file, line := getCaller(e.Skip)
		callerStr := fmt.Sprintf(" [%s:%d]", file, line)
		buf.WriteString(callerStr)
	}

	// 6. 处理核心消息（带/不带颜色通用逻辑）
	buf.WriteString(" - ")
	msgStr := e.Message
	if f.EnableColor {
		msgStr = config2.ColorWhite + msgStr + config2.ColorReset
	}
	buf.WriteString(msgStr)

	// 7. 处理扩展字段（Fields）：补充trace_id、client_ip等
	if len(e.Fields) > 0 {
		buf.WriteString(" | ")
		fieldStrs := make([]string, 0, len(e.Fields))
		for k, v := range e.Fields {
			fieldStrs = append(fieldStrs, fmt.Sprintf("%s=%v", k, v))
		}
		buf.WriteString(strings.Join(fieldStrs, ", "))
	}

	// 8. 换行符（统一格式）
	buf.WriteString("\n")

	// 9. 转换为[]byte返回（匹配接口要求）
	return []byte(buf.String()), nil
}

// ========== 辅助函数：利用Skip获取真实调用方（核心） ==========
func getCaller(skip int) (string, int) {
	// runtime.Caller(skip)：跳过指定层级，获取业务代码的调用位置
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown", 0
	}

	// 优化：只保留文件名（去掉全路径，如 /app/handler/user.go → user.go）
	file = filepath.Base(file)

	// 可选：获取函数名（如 main.userLoginHandler）
	// funcName := runtime.FuncForPC(pc).Name()
	// return fmt.Sprintf("%s:%s:%d", funcName, file, line), line

	return file, line
}
