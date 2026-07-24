package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

type entry struct {
	value     string
	expiresAt time.Time
	version   int64
}

type KVStore struct {
	mu    sync.RWMutex
	items map[string]entry
}

func NewKVStore() *KVStore {
	return &KVStore{items: make(map[string]entry)}
}

func (s *KVStore) Set(key, value string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	version := time.Now().UnixNano()
	entry := entry{
		value:   value,
		version: version,
	}

	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl)
	}

	s.items[key] = entry
}

func (s *KVStore) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.items[key]
	if !ok {
		return "", false
	}

	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		return "", false
	}

	return item.value, true
}

func (s *KVStore) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[key]; !ok {
		return false
	}

	delete(s.items, key)
	return true
}

func (s *KVStore) cleanupExpired() {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	for key, item := range s.items {
		if item.expiresAt.IsZero() {
			continue
		}
		if now.After(item.expiresAt) {
			delete(s.items, key)
		}
	}
}

func main() {
	store := NewKVStore()
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			store.cleanupExpired()
		}
	}()

	log.Println("mini kv-store started")
	log.Println("commands: SET key value [ttl], GET key, DELETE key, HELP, EXIT")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("mini-kv> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		cmd := strings.ToUpper(parts[0])

		switch cmd {
		case "SET":
			if len(parts) < 3 {
				fmt.Println("usage: SET key value [ttl]")
				continue
			}

			key := parts[1]
			value := parts[2]
			var ttl time.Duration
			if len(parts) >= 4 {
				parsedTTL, err := time.ParseDuration(parts[3])
				if err != nil {
					fmt.Printf("invalid ttl: %v\n", err)
					continue
				}
				ttl = parsedTTL
			}

			store.Set(key, value, ttl)
			if ttl > 0 {
				fmt.Printf("SET %s=%s ttl=%s\n", key, value, ttl)
			} else {
				fmt.Printf("SET %s=%s\n", key, value)
			}

		case "GET":
			if len(parts) != 2 {
				fmt.Println("usage: GET key")
				continue
			}

			value, ok := store.Get(parts[1])
			if !ok {
				fmt.Printf("GET %s -> (not found)\n", parts[1])
				continue
			}
			fmt.Printf("GET %s -> %s\n", parts[1], value)

		case "DELETE":
			if len(parts) != 2 {
				fmt.Println("usage: DELETE key")
				continue
			}

			if deleted := store.Delete(parts[1]); deleted {
				fmt.Printf("DELETE %s -> ok\n", parts[1])
			} else {
				fmt.Printf("DELETE %s -> not found\n", parts[1])
			}

		case "HELP":
			fmt.Println("SET key value [ttl] | GET key | DELETE key | EXIT")

		case "EXIT", "QUIT":
			fmt.Println("bye")
			return

		default:
			fmt.Printf("unknown command: %s\n", cmd)
		}
	}
}
