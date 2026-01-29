package hooks

import (
	config2 "LogX/config"
	"LogX/core"
	"regexp"
)

type DesensitizeHook struct{}

// Fire 实现钩子核心逻辑：替换日志中的手机号为 138****5678 格式
func (d *DesensitizeHook) Fire(entry *core.Entry, stage HookStage) (bool, error) {
	phoneReg := regexp.MustCompile(`1[3-9]\d{9}`)
	entry.Message = phoneReg.ReplaceAllStringFunc(entry.Message, func(matched string) string {
		if len(matched) >= 7 {
			return matched[:3] + "****" + matched[7:]
		}
		return matched
	})
	return false, nil
}

// Levels 支持所有日志级别
func (d *DesensitizeHook) Levels() []config2.LogLevel {
	return []config2.LogLevel{
		config2.TRACE, config2.DEBUG, config2.INFO,
		config2.WARNING, config2.ERROR, config2.PANIC, config2.FATAL,
	}
}

// Stages 绑定到「格式化前」阶段
func (d *DesensitizeHook) Stages() []HookStage {
	return []HookStage{StageBeforeFormat}
}
