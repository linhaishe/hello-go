# 网站/API 可用性健康监测器 (Site Checker)

> **目标**：理解并发控制、定时器以及 Go 极简的错误处理机制。

- **功能需求**：
  - 从配置文件或数组中读取 10+ 个网址（如 baidu.com, github.com 等）。
  - 定时（如每 10 秒）向这些网址发送 HTTP GET 请求。
  - 如果请求超时或状态码非 200，记录错误日志并输出提示。
  - 用 `time.Ticker` 实现周期调度，用 `context.WithTimeout` 实现超时控制。
- **用到的 Go 特性**：
  - `net/http` 发送 Client 请求。
  - `time` 包（定时器 `Ticker`、休眠 `Sleep`）。
  - `context` 上下文机制（控制 HTTP 请求超时）。
  - `defer` 优雅关闭 HTTP 响应体（`resp.Body.Close()`）。

## 运行方式

在仓库根目录执行：

```bash
go run ./cmd/mini-site-check
```

如果你想先检查代码是否能编译：

```bash
go build ./cmd/mini-site-check
```

## 运行效果

启动后，程序会输出：

```text
site checker started
checking 12 targets every 10s
```

随后每 10 秒会并发检查一次配置中的目标网址：

- 请求成功且返回 `200`：输出 `[OK] ...`
- 请求失败或超时：输出 `[ERR] ...`
- 返回状态码不是 `200`：输出 `[WARN] ...`

## 说明

这不是一个 Web 服务，而是一个“定时健康检查”的小型示例程序。它会持续运行，直到你手动按下 `Ctrl + C` 退出。

# QA

```go
for range ticker.C {
    checkAll(targets)
}
```

```go
// 等价于更展开的写法：
for {
    _, ok := <-ticker.C
    if !ok {
        break
    }

    checkAll(targets)
}
```

如果这里的 `range` 不关心通道里收到的具体时间值，只需要“收到一次就执行一次”。

```go
for now := range ticker.C {
    fmt.Println("触发时间：", now)
    checkAll(targets)
}
```

---

```go
ticker := time.NewTicker(checkInterval)

defer ticker.Stop() // 当前函数结束时，再执行 `ticker.Stop()`。
```

创建 ticker 后，先登记一个“收尾动作”。无论函数是正常 `return`，还是中途发生 `panic`，函数退出前都会调用：`ticker.Stop()`

这样可以停止定时器，释放它占用的资源，避免它继续产生定时事件。

```go
func main() {
    defer fmt.Println("最后执行")

    fmt.Println("先执行")
}
```

多个 `defer` 会按“后写先执行”的顺序执行：

```go
defer fmt.Println("1")
defer fmt.Println("2")
defer fmt.Println("3")
```

函数结束时输出：

```go
3
2
1
```

常见用途是“申请资源后立刻登记释放动作”：

```go
file, err := os.Open("data.txt")
if err != nil {
    return err
}
defer file.Close()
```

这样后续即使提前返回，也不用担心漏掉 `file.Close()`。

那这个defer为什么不写在for后面？

因为这个 `for range ticker.C` 通常是无限循环，代码不会走到 `for` 后面。

```go
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
}
```

```text
urls: A、B、C

启动检查 A  ─┐
启动检查 B  ─┼─ 同时进行
启动检查 C  ─┘

wg.Wait() 等待
A 完成 → Done()
B 完成 → Done()
C 完成 → Done()

全部完成 → checkAll 返回
```

```go
func checkSite(site string) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel() // 即便请求提前成功，也会尽快释放 context 关联的计时器等资源。

  // 之后把 ctx 传给支持 context 的操作：
  // 若请求超过超时时间，ctx 会自动取消，HTTP 请求也会停止，并常见地返回类似：context deadline exceeded
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, site, nil)
  // 发起 HTTP 请求，分别处理“请求创建失败”“请求发送失败”“状态码不是 200”的情况
	if err != nil {
		log.Printf("[ERR] %s: build request failed: %v", site, err)
		return
	}

  // 真正发送 HTTP 请求，接收服务器响应。
  // 如果发送过程失败，例如：网络断开,DNS 解析失败,请求超时（context deadline exceeded）,HTTPS 证书错误,就打印错误并结束这次检查。
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[ERR] %s: request failed: %v", site, err)
		return
	}
  // 请求成功得到 resp 后，要关闭响应体，释放连接等资源。用 defer 确保后面即使提前 return，也会自动关闭。
	defer resp.Body.Close()

  // 检查 HTTP 状态码是否是 200 OK
	if resp.StatusCode != http.StatusOK {
		log.Printf("[WARN] %s: unexpected status code: %s", site, resp.Status)
		return
	}

	fmt.Printf("[OK] %s -> %s\n", site, resp.Status)
}
```
`ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)`

这行创建了一个“带超时限制的上下文”。

- `ctx`：传给 HTTP 请求、数据库操作等，用来通知它们“该取消了”。
- `cancel`：手动取消这个上下文的函数。
- `requestTimeout`：最长允许执行多久，例如 `5 * time.Second`。

`context.Background()`

创建最基础的 context，通常作为程序最上层的起点。

`context.WithTimeout(parent, duration)`

基于父 context 创建一个子 context。满足任一条件时，这个 `ctx` 都会被取消：

1. 经过了 `requestTimeout`。
2. 父 context 被取消。
3. 主动调用 `cancel()`。

---

Go 常用“返回值 + `error`”报告错误：

```
结果, err := 某个函数()
```

- 没有错误时：`err == nil`
- 有错误时：`err != nil`

`nil` 可以理解为“空、没有值”。所以：

`err != nil`

就是“`err` 不是空的，说明确实发生了错误”。











