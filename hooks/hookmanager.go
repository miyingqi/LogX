package hooks

import (
	"LogX/config"
	"LogX/core"
)

type HookManager struct {
	hooks []HookBase
}

func NewHookManager() *HookManager {
	return &HookManager{
		hooks: make([]HookBase, 0), // 初始化空切片，避免nil切片遍历问题
	}
}

// AddHook 注册自定义钩子到管理器（核心：将钩子追加到切片，支持注册多个）
func (hm *HookManager) AddHook(hook HookBase) {
	if hook == nil {
		return // 防止注册nil钩子，避免后续遍历panic
	}
	hm.hooks = append(hm.hooks, hook)
}

// RunHooks 按「指定阶段+指定日志级别」执行所有匹配的钩子（核心执行方法）
// 入参：执行阶段、当前日志级别、日志Entry（指针，钩子可修改内容）
// 出参：bool(是否跳过后续日志流程)、[]error(收集所有钩子执行错误)
func (hm *HookManager) RunHooks(stage HookStage, level config.LogLevel, entry *core.Entry) (bool, []error) {
	var errs []error // 收集钩子执行错误，单个钩子失败不影响其他

	// 遍历所有注册的钩子，执行「阶段+级别」匹配的钩子
	for _, hook := range hm.hooks {
		// 第一步：判断钩子是否绑定了当前执行阶段（不匹配则跳过）
		if !hm.hookSupportStage(hook, stage) {
			continue
		}

		// 第二步：判断钩子是否支持当前日志级别（不匹配则跳过）
		if !hm.hookSupportLevel(hook, level) {
			continue
		}

		// 第三步：执行钩子核心逻辑
		skip, err := hook.Fire(entry, stage)
		// 收集错误：单个钩子失败，不终止其他钩子执行（错误隔离）
		if err != nil {
			errs = append(errs, err)
		}
		// 触发跳过：任意钩子返回skip=true，立即终止所有后续钩子+日志流程
		if skip {
			return true, errs
		}
	}

	// 所有钩子执行完成，未触发跳过，返回错误集合（无错误则为空）
	return false, errs
}

// ===== 私有辅助方法：判断钩子是否支持指定阶段/级别（解耦核心逻辑，提升可读性）=====
// hookSupportStage 判断钩子是否绑定了指定执行阶段
func (hm *HookManager) hookSupportStage(hook HookBase, stage HookStage) bool {
	for _, hStage := range hook.Stages() {
		if hStage == stage {
			return true
		}
	}
	return false
}

// hookSupportLevel 判断钩子是否支持指定日志级别
func (hm *HookManager) hookSupportLevel(hook HookBase, level config.LogLevel) bool {
	for _, hLevel := range hook.Levels() {
		if hLevel == level {
			return true
		}
	}
	return false
}
