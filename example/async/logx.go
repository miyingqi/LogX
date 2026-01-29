package main

import (
	"LogX"
	"sync"
)

func main() {
	asyncLogger := LogX.NewDefaultAsyncLogger("logx")

	var wg sync.WaitGroup
	wg.Add(1000) // 关键1：标记要等待100个协程

	for i := 0; i < 1000; i++ {
		go func(sync *LogX.AsyncLogger, i int) {
			defer wg.Done() // 添加这行来通知waitgroup任务完成
			sync.Field(map[string]any{
				"id": i,
			}).Info("debug %d", i)

			sync.Warn("warning %d", i)

		}(asyncLogger, i)

	}
	wg.Wait()
	defer asyncLogger.Close()
}
