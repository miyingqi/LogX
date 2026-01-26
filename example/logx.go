package main

import (
	"LogX"
	"sync"
)

func main() {
	syncLogger, err := LogX.NewDefaultSyncLogger("logx")
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	wg.Add(100) // 关键1：标记要等待100个协程

	for i := 0; i < 100; i++ {

		syncLogger.Debug("debug %d", i)
		syncLogger.Info("hello world %d", i)
		syncLogger.Error("error %d", i)
		syncLogger.Warn("warning %d", i)
		syncLogger.Fatal("fatal %d", i)
	}
	defer syncLogger.Close()
}
