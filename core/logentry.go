package core

import (
	config2 "LogX/config"
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

func NewEntry() *Entry {
	return &Entry{
		Fields: make(map[string]interface{}, 8),
	}
}
func (e *Entry) SetEntry(time time.Time, level config2.LogLevel, message string, model string, skip int, fields map[string]interface{}) {
	e.Timestamp = time
	e.Level = level
	e.Message = message
	e.Fields = fields
	e.Skip = skip
	e.Model = model
}

func (e *Entry) Reset() {
	e.Timestamp = time.Time{}
	e.Level = config2.LogLevel(0)
	e.Message = ""
	for k := range e.Fields {
		delete(e.Fields, k)
	}
	e.Skip = 0
	e.Model = ""
}
