package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	checkInterval  = 10 * time.Second
	requestTimeout = 3 * time.Second
)

var targets = []string{
	"https://www.baidu.com",
	"https://www.github.com",
	"https://www.google.com",
	"https://www.bing.com",
	"https://www.qq.com",
	"https://www.sina.com.cn",
	"https://www.apple.com",
	"https://www.microsoft.com",
	"https://www.amazon.com",
	"https://www.netflix.com",
	"https://www.zhihu.com",
	"https://www.example.com",
}

func main() {
	log.Println("site checker started")
	log.Printf("checking %d targets every %s\n", len(targets), checkInterval)

	// 创建一个定时器 ticker，每隔 checkInterval 时间发送一次信号。
	ticker := time.NewTicker(checkInterval)

	// 在当前函数结束时停止定时器，避免资源泄漏。
	defer ticker.Stop()

	// ticker.C 是一个 channel。每到一次设定的时间间隔，它就会接收到一个值；for range 会持续等待并接收这些值。
	for range ticker.C {
		// 因此每收到一次信号，就执行
		checkAll(targets)
	}
}

// 同时检查多个网址，并等所有检查完成后才返回。
func checkAll(urls []string) {
	// 创建 WaitGroup，它用来记录“还有多少个并发任务没完成”。
	var wg sync.WaitGroup

	for _, target := range urls {
		wg.Add(1)
		go func(site string) {
			defer wg.Done()
			checkSite(site)
		}(target)
	}

	wg.Wait()
	// 主 goroutine 在这里等待，直到所有 goroutine 都调用过 wg.Done()，任务数变成 0，checkAll 才结束。
	// 如果没有 wg.Wait()，checkAll 会在刚启动所有 goroutine 后立刻返回，不会等检查结果完成。
}

func checkSite(site string) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, site, nil)
	if err != nil {
		log.Printf("[ERR] %s: build request failed: %v", site, err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[ERR] %s: request failed: %v", site, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[WARN] %s: unexpected status code: %s", site, resp.Status)
		return
	}

	fmt.Printf("[OK] %s -> %s\n", site, resp.Status)
}
