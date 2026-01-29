package hooks

import (
	config2 "LogX/config"
	"LogX/core"
)

type HookFunc func(entry *core.Entry) error
type HookBase interface {
	Fire(entry *core.Entry, stage HookStage) (bool, error)
	Stages() []HookStage
	Levels() []config2.LogLevel
}
type HookStage string

const (
	StageBeforeFormat HookStage = "before_format" // 格式化前：修改Entry/脱敏/补全字段
	StageAfterFormat  HookStage = "after_format"  // 格式化后：修改格式化后的字节
	StageBeforeWrite  HookStage = "before_write"  // 写入前：最终过滤/校验日志
	StageAfterWrite   HookStage = "after_write"   // 写入后：异步告警/推第三方/审计
)
