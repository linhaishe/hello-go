package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type matchResult struct {
	File string
	Line int
	Text string
}

// 递归扫描目录内的文件，并发搜索关键词，打印每个匹配的位置。
func main() {
	dir := flag.String("dir", ".", "target directory")
	keyword := flag.String("keyword", "", "keyword to search")
	flag.Parse()

	// 没有传 -keyword 就提示用法并结束，避免搜索空字符串（空字符串会匹配几乎所有内容）
	if *keyword == "" {
		fmt.Println("usage: go run ./cmd/mini-grep -dir ./path -keyword hello")
		return
	}

	results := make(chan matchResult, 128)
	var wg sync.WaitGroup

	// 递归遍历，会递归遍历 *dir 目录下所有目录和文件。每发现一个路径，就调用一次这个回调函数
	// path：当前路径，如 ./path/a.txt
	// d：当前路径的信息，如是否为目录
	// err：访问这个路径时产生的错误
	if err := filepath.WalkDir(*dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		wg.Add(1)
		go func(filePath string) { // 启动一个 goroutine，独立搜索这个文件。
			defer wg.Done()
			// goroutine 即将结束时执行 wg.Done()，告诉 WaitGroup：“这个文件搜索完成了”。
			matches, err := searchFile(filePath, *keyword)
			// searchFile 应该会读取该文件，找出包含关键词的行，并返回匹配结果。

			if err != nil {
				// 若文件无法读取，例如没有权限或它是特殊/二进制文件，就记日志并跳过该文件，不影响其他文件继续搜索。
				log.Printf("failed to read %s: %v", filePath, err)
				return
			}
			// 该文件找到的每一项匹配都发到 results channel。发送时若 channel 已满，会在这里等待，直到有人读取结果。
			for _, item := range matches {
				results <- matchResult{File: filePath, Line: item.Line, Text: item.Text}
			}
		}(path)

		return nil
	}); err != nil {
		log.Fatalf("walk directory failed: %v", err)
	}

	// 遍历目录本身如果失败，例如目录不存在，打印错误并直接结束程序。log.Fatal 会调用 os.Exit(1)。
	go func() {
		wg.Wait()      // 等待所有文件搜索 goroutine 全部结束；
		close(results) // 确认不会再有任何人向 results 发送结果后，关闭 channel。
	}()

	found := false
	// 因为 results 是一个 channel，for range results 本身就在不断“接收”它的数据。
	// 等价于反复做：result := <-results
	for result := range results {
		found = true
		fmt.Printf("%s:%d: %s\n", result.File, result.Line, result.Text)
	}

	if !found {
		fmt.Printf("no matches found for keyword: %s\n", *keyword)
	}
}

// 在一个文件中搜索包含指定关键字的所有行，并返回这些行。
func searchFile(filePath, keyword string) ([]matchResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close() // 现在登记一下，等整个函数结束以后，自动执行

	scanner := bufio.NewScanner(file)
	// 按行读取文件
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	// 把 Scanner 默认能读取的一行长度调大。默认 Scanner 一行只能：64KB，如果：一行 300KB，Scanner 会报错：token too long` 所以这里修改成：最大支持`1MB`
	lineNo := 0
	// 记录当前行号
	var matches []matchResult

	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		if strings.Contains(line, keyword) {
			matches = append(matches, matchResult{
				Line: lineNo,
				Text: strings.TrimSpace(line),
				// 去掉首尾空格
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return matches, nil
}
