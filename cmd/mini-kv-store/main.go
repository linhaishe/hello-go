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
	s.mu.Lock()         // 加互斥写锁，防止并发写入冲突
	defer s.mu.Unlock() // 函数执行完毕后自动解锁

	version := time.Now().UnixNano() // 用纳秒时间戳作为数据的版本号 不保证绝对唯一，但在大多数普通场景下非常接近唯一
	entry := entry{
		value:   value,
		version: version,
	}

	// 如果设置了 TTL（大于 0），计算出具体的过期绝对时刻 Time To Live
	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl) // 给当前时间点加上一段时间
	}

	s.items[key] = entry // 存入 map
}

func (s *KVStore) Get(key string) (string, bool) {
	s.mu.RLock()         // 加共享读锁，允许同时有多个 Goroutine 进来读
	defer s.mu.RUnlock() // 函数退出时解读锁

	item, ok := s.items[key] // value, exists := map[key]
	if !ok {
		return "", false // Key 不存在
	}

	// 惰性删除判断（Lazy Expiration）
	// expiresAt.IsZero()：判断该 key 是否设置了过期时间。如果是 0 值，说明是永久有效的 key。
	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		return "", false // 虽然 map 里还有这个 key，但它已经过期了，当作不存在处理
	}

	// 惰性过期（Lazy Expiration）：即便后台清理线程还没来得及把过期的 key 从 map 中删除，
	// 只要用户查到它时发现 time.Now().After(item.expiresAt)，就立即返回 false，保证用户绝不会查到过期数据。

	return item.value, true
}

func (s *KVStore) Delete(key string) bool {
	s.mu.Lock() // 修改 map 需要加写锁
	defer s.mu.Unlock()

	if _, ok := s.items[key]; !ok {
		return false // 如果 key 本来就不存在，返回 false 告知调用者
	}

	delete(s.items, key) // 调用原生 delete 函数从 map 中移除
	return true
}

func (s *KVStore) cleanupExpired() {
	now := time.Now()

	s.mu.Lock() // 遍历并删除 map 元素需要加写锁
	defer s.mu.Unlock()

	for key, item := range s.items {
		if item.expiresAt.IsZero() {
			continue // 永不过期的 key 直接跳过
		}
		if now.After(item.expiresAt) {
			delete(s.items, key) // 已过期，从内存中抹除，释放资源
		}
	}
}

// 如果有些 key 设置了过期时间，但之后再也没有人调用 Get 去查询它，那么“惰性过期”就永远不会触发。这些 key 就会一直残留在内存里造成内存泄漏。
// 使用方式：通常会在后台用一个 Goroutine + time.Ticker 定时（比如每隔 1 秒或 1 分钟）自动调用一次 cleanupExpired()。

// 经典的 REPL（Read-Eval-Print Loop，读取-执行-输出 循环） 模型
func main() {
	// 启动后台清理任务（Goroutine + Ticker）
	store := NewKVStore()
	// 启动一个新的 Goroutine（轻量级线程）。这让清理任务在后台默默运行，不会阻塞主程序等待用户输入。
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		// 创建一个“定时器”，就像节拍器一样，每隔 100 毫秒（0.1秒）就会触发一次。
		defer ticker.Stop()
		for range ticker.C {
			store.cleanupExpired()
		}
	}()

	log.Println("mini kv-store started")
	log.Println("commands: SET key value [ttl], GET key, DELETE key, HELP, EXIT")

	// 初始化标准输入扫描器
	scanner := bufio.NewScanner(os.Stdin) // os.Stdin：代表操作系统的标准输入（即键盘输入）。
	for {
		fmt.Print("mini-kv> ")
		// 用 scanner.Scan() 阻塞等待你敲击回车。如果你按下 Ctrl+C 或 Ctrl+D 结束输入，Scan() 会返回 false，从而 break 退出程序。
		if !scanner.Scan() {
			break
		}

		// 解析用户输入的命令
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)    // 按空格把字符串切分成数组，自动忽略中间多余的空格
		cmd := strings.ToUpper(parts[0]) // 提取第一个单词作为命令，并强制转为大写 set、Set、SET

		// 路由分发（Switch-Case 命令处理）
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
