package main

import "LogX"

// 业务代码使用多实例
func main() {
	logger, _ := LogX.NewSyncLogger("default")
	for i := 0; i < 100; i++ {
		logger.Debug("debug")
		logger.Info("hello world")
		logger.Error("error")
		logger.Warn("warning")
		logger.Fatal("fatal")
	}
	defer logger.Close()
}
