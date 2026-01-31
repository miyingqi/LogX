package main

import (
	"LogX"
	"LogX/hooks"
)

func main() {
	syncLogger := LogX.NewDefaultSyncLogger("sync")
	syncLogger.AddHook(&hooks.DesensitizeHook{})

	// 添加文件写入钩子
	fileHook := hooks.NewFileWriteHook("sync.log")
	syncLogger.AddHook(fileHook)
	defer fileHook.Close()

	for i := 0; i < 100; i++ {
		syncLogger.Debug("debug %s", "13345678910")
		syncLogger.Field(map[string]interface{}{"name": "张了", "phone": "13345678980"}).Info("hello world %s", "13345678910")
		syncLogger.Field(map[string]interface{}{"name": "张三", "phone": "13345678910"}).Info("hello world %s", "13345678910")
	}
}
