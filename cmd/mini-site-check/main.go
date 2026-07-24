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

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		checkAll(targets)
	}
}

func checkAll(urls []string) {
	var wg sync.WaitGroup

	for _, target := range urls {
		wg.Add(1)
		go func(site string) {
			defer wg.Done()
			checkSite(site)
		}(target)
	}

	wg.Wait()
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
