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

func main() {
	dir := flag.String("dir", ".", "target directory")
	keyword := flag.String("keyword", "", "keyword to search")
	flag.Parse()

	if *keyword == "" {
		fmt.Println("usage: go run ./cmd/mini-grep -dir ./path -keyword hello")
		return
	}

	results := make(chan matchResult, 128)
	var wg sync.WaitGroup

	if err := filepath.WalkDir(*dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()
			matches, err := searchFile(filePath, *keyword)
			if err != nil {
				log.Printf("failed to read %s: %v", filePath, err)
				return
			}
			for _, item := range matches {
				results <- matchResult{File: filePath, Line: item.Line, Text: item.Text}
			}
		}(path)

		return nil
	}); err != nil {
		log.Fatalf("walk directory failed: %v", err)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	found := false
	for result := range results {
		found = true
		fmt.Printf("%s:%d: %s\n", result.File, result.Line, result.Text)
	}

	if !found {
		fmt.Printf("no matches found for keyword: %s\n", *keyword)
	}
}

func searchFile(filePath, keyword string) ([]matchResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	var matches []matchResult

	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		if strings.Contains(line, keyword) {
			matches = append(matches, matchResult{
				Line: lineNo,
				Text: strings.TrimSpace(line),
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return matches, nil
}
