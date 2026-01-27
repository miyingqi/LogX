package main

import (
	"LogX"
	"sync"
)

func main() {
	syncLogger, err := LogX.NewDefaultAsyncLogger("logx")
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup
	wg.Add(100) // 关键1：标记要等待100个协程

	for i := 0; i < 100; i++ {
		go func(sync *LogX.AsyncLogger, i int) {
			defer wg.Done() // 添加这行来通知waitgroup任务完成
			sync.Debug("debug %d", i)
			sync.Info("hello world %d", i)
			sync.Error("error %d", i)
			sync.Warn("warning %d", i)
			sync.Fatal("fatal %d", i)
		}(syncLogger, i)

	}
	wg.Wait()

}
