## 命令行文件/文本并发搜索工具 (Mini-Grep)

> **目标**：熟悉 Go 的文件操作、命令行参数解析，以及最核心的 **并发（Goroutine + Channel）**。

- **功能需求**：
  - 接收命令行参数：目标文件夹路径、要搜索的关键词。
  - 递归遍历文件夹下的所有文件。
  - 为每个文件开启一个 Goroutine 搜索关键词。
  - 通过 Channel 汇总所有匹配到关键词的文件名和行号并打印。
- **用到的 Go 特性**：
  - `os`、`filepath` 模块进行目录遍历与文件读取。
  - `flag` 模块解析命令行参数。
  - `go func()` 启动协程、`chan` 传输结果、`sync.WaitGroup` 等待所有文件检索完成。

## 运行方式

```bash
go run ./cmd/mini-grep -dir ./ -keyword "hello"
```

如果你想搜索当前仓库里的某个关键词，可以这样做：

```bash
go run ./cmd/mini-grep -dir . -keyword "package"
```

## 示例输出

```text
./cmd/mini-kv-store/main.go:12: type KVStore struct {
./cmd/mini-site-check/main.go:27: var targets = []string{
```

## 说明

- `-dir` 指定要搜索的根目录。
- `-keyword` 指定要查找的关键词。
- 程序会递归扫描目录下的所有文件，并为每个文件启动一个 goroutine 处理搜索逻辑。
- 命中结果会按 `文件路径:行号:内容` 的格式输出。

例如文件 `test.txt`：

```
hello world
go is awesome
hello golang
python
```

调用：

```
matches, err := searchFile("test.txt", "hello")
```

得到：

```
[
    {Line: 1, Text: "hello world"},
    {Line: 3, Text: "hello golang"},
]
```

# QA

| 包              | 功能                         | 这段 mini-grep 中的用途                                |
| --------------- | ---------------------------- | ------------------------------------------------------ |
| `bufio`         | 带缓冲的输入/输出工具。      | 用 `bufio.Scanner` 按行读取文件内容。                  |
| `flag`          | 解析命令行参数。             | 读取 `-dir` 和 `-keyword`。                            |
| `fmt`           | 格式化输入输出。             | 打印匹配行、使用说明和“未找到结果”提示。               |
| `io/fs`         | 文件系统接口和类型定义。     | 使用 `fs.DirEntry` 表示遍历目录时获得的文件/目录信息。 |
| `log`           | 输出带时间信息的日志。       | 记录文件读取失败、目录遍历失败等错误。                 |
| `os`            | 操作系统相关能力。           | 例如用 `os.Open` 打开文件。                            |
| `path/filepath` | 处理本地文件路径、遍历目录。 | 使用 `filepath.WalkDir` 递归扫描目标目录。             |
| `strings`       | 字符串处理。                 | 用 `strings.Contains` 判断一行文本是否包含关键词。     |
| `sync`          | 并发同步工具。               | 用 `sync.WaitGroup` 等待所有文件搜索 goroutine 完成。  |

Flag: https://pkg.go.dev/flag#hdr-Command_line_flag_syntax

```
var svar string
flag.StringVar(&svar, "svar", "bar", "a string var")
```

| 部分              | 含义                                       |
| ----------------- | ------------------------------------------ |
| `var svar string` | 创建一个字符串变量，初始值是 `""`          |
| `&svar`           | 取得变量 `svar` 的地址，让 `flag` 能修改它 |
| `"svar"`          | 定义命令行参数名：`-svar`                  |
| `"bar"`           | 默认值                                     |
| `"a string var"`  | 参数说明，显示在 `-help` 中                |

```
解析参数
  ↓
递归遍历目录
  ↓
每个文件启动一个 goroutine 搜索
  ↓                    ↓
主 goroutine 持续读取并打印 results
  ↓                    ↑
全部文件搜索完 → 关闭 results
  ↓
没有结果则提示
```

```go
package main

import (
    "flag"
    "fmt"
)

func main() {
    var svar string
    flag.StringVar(&svar, "svar", "bar", "a string var")

    flag.Parse()

    fmt.Println(svar)
}
```

```go
`go run . -svar hello`

hello // with parms
bar // no pram
```

```go
// String：返回 *string，需要解引用
keyword := flag.String("keyword", "", "keyword to search")
fmt.Println(*keyword)

// StringVar：你自己提供 string 变量，直接使用
var keyword string
flag.StringVar(&keyword, "keyword", "", "keyword to search")
fmt.Println(keyword)
```

```go
dir := flag.String("dir", ".", "target directory")
keyword := flag.String("keyword", "", "keyword to search")
flag.Parse()
```

| 代码                                | 含义                                   |
| ----------------------------------- | -------------------------------------- |
| `flag.String("dir", ".", "...")`    | 定义 `-dir` 参数，默认值是 `"."`       |
| `flag.String("keyword", "", "...")` | 定义 `-keyword` 参数，默认值是空字符串 |
| `flag.Parse()`                      | 读取用户启动程序时传入的参数           |

`go run ./cmd/mini-grep -dir ./demo -keyword hello`

```go
*dir     // "./demo"
*keyword // "hello"
```

如果用户没有传参数：

```
go run ./cmd/mini-grep
```

则使用默认值：

```
*dir     // "."
*keyword // ""
```

还有其他常用类型：

```go
port := flag.Int("port", 8080, "server port")
debug := flag.Bool("debug", false, "enable debug mode")
timeout := flag.Duration("timeout", 5*time.Second, "request timeout")

flag.Parse() // 必须

fmt.Println(*port, *debug, *timeout)
```

---

这个函数在哪里被调用了？

```go
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
```

它是在定义完的同时，立刻被调用了：

```
go func(filePath string) {
    // 函数内容
}(path)
```

最后的：

```
(path)
```

就是调用这个匿名函数，并把当前的 `path` 传给参数 `filePath`。

等价于先给函数取名，再调用：

```go
func searchOneFile(filePath string) {
    defer wg.Done()

    matches, err := searchFile(filePath, *keyword)
    if err != nil {
        log.Printf("failed to read %s: %v", filePath, err)
        return
    }

    for _, item := range matches {
        results <- matchResult{
            File: filePath,
            Line: item.Line,
            Text: item.Text,
        }
    }
}

go searchOneFile(path)
```

---

那这个path是在哪里传进来的，我看是从`filepath.WalkDir`，但`filepath.WalkDir`是什么时候被调用的

`filepath.WalkDir` 就是在这行被直接调用的：

```
filepath.WalkDir(...)
```

这是在调用 `WalkDir`。因为它就一函数

```go
if err := filepath.WalkDir(*dir, func(path string, d fs.DirEntry, err error) error {
    // ...
}); err != nil {
    log.Fatalf("walk directory failed: %v", err)
}
```

可以拆成这样理解：

```go
err := filepath.WalkDir(*dir, 回调函数)

if err != nil {
    log.Fatalf("walk directory failed: %v", err)
}
```

当程序运行到这一行时，`filepath.WalkDir` 立即开始递归遍历 `*dir` 目录。

因为它是一个普通的函数调用表达式：函数名后面有括号，就表示“现在调用它”。

```go
filepath.WalkDir(*dir, func(...) error {
    // 回调函数内容
})
```

和下面是同一个规律：

```go
fmt.Println("hello") // 立刻调用 Println
checkAll(targets)    // 立刻调用 checkAll
```

---

我一般啥时候需要`close(results)`

当你需要告诉接收方：**不会再有新的数据发送到这个 channel 了**，就需要 `close(results)`。

你的场景中：

```
for result := range results {
    fmt.Println(result)
}
```

`for range results` 只有在 `results` 被关闭后才会结束。

所以必须在所有文件搜索任务结束后关闭：

```
go func() {
    wg.Wait()      // 所有发送方都结束
    close(results) // 宣布：后面不会再发送结果
}()
```

否则主 goroutine 会一直卡在这里：

```
for result := range results {
    // 没有新结果时，会继续等下一条
}
```

常见判断方式：

| 场景                                             | 是否需要 `close(channel)` |
| ------------------------------------------------ | ------------------------- |
| 接收方使用 `for range ch`，需要自然结束          | 需要                      |
| 需要向接收方传达“数据发送完毕”                   | 需要                      |
| channel 只是持续运行的事件流，程序退出时直接结束 | 通常不需要                |
| 只有一次接收：`value := <-ch`                    | 通常不需要                |
| 不确定谁负责发送完毕                             | 不要随便关闭              |

原则：**由发送方关闭 channel，不是接收方。**

更准确地说，应该由“能确定不会再有任何发送操作的人”关闭。你的例子中有多个文件 goroutine 都在发送 `results`，所以不能让任意一个文件 goroutine 关闭；要由一个协调者等待所有发送方结束后再关闭。

```
文件 goroutine A ─┐
文件 goroutine B ─┼─→ results
文件 goroutine C ─┘
                   ↓
        全部完成后统一 close(results)
```

---

```go
found := false
// 因为 results 是一个 channel，for range results 本身就在不断“接收”它的数据。
// 等价于反复做：result := <-results
for result := range results {
  found = true
  fmt.Printf("%s:%d: %s\n", result.File, result.Line, result.Text)
}
```

```go
// 最容易理解的写法是不用 range，显式接收：
for {
  result, ok := <-results

  if !ok {
      // results 已被 close，说明不会再有结果了
      break
  }

  found = true
  fmt.Printf("%s:%d: %s\n", result.File, result.Line, result.Text)
}
```

----

我看很多时候都有close,我一般啥时候能用close

`Close()` 通常用于释放“打开后一直占着资源”的东西，例如文件、网络连接、HTTP 响应体。

```
file, err := os.Open("a.txt")
if err != nil {
    return err
}
defer file.Close()
```

原则很简单：

> 只要你“打开、创建、获取”了一个需要关闭的资源，并且它有 `Close()` 方法，通常就应在成功后立刻写 `defer xxx.Close()`。

常见场景：

| 获取资源                                  | 关闭方式                  | 原因                   |
| ----------------------------------------- | ------------------------- | ---------------------- |
| `os.Open(...)` 打开文件                   | `defer file.Close()`      | 释放文件描述符         |
| `os.Create(...)` 创建文件                 | `defer file.Close()`      | 刷新/释放文件资源      |
| `http.Client.Do(...)` 得到响应            | `defer resp.Body.Close()` | 释放响应体和连接       |
| `net.Listen(...)` 创建监听器              | `defer listener.Close()`  | 停止监听端口           |
| `net.Dial(...)` 创建网络连接              | `defer conn.Close()`      | 断开连接               |
| `sql.Open(...)` 创建数据库连接池          | `defer db.Close()`        | 释放连接池资源         |
| `os.Open` 后配合 `bufio.NewScanner(file)` | 仍关闭 `file`             | Scanner 不负责关闭文件 |

典型模式：

```
resource, err := getResource()
if err != nil {
    return err
}
defer resource.Close()

// 后续安全使用 resource
```

注意：这里的 `Close()` 与 channel 的 `close(ch)` 不一样：

```
defer file.Close() // 调用对象的方法，关闭文件资源
close(results)     // Go 内置函数，关闭 channel
```

一般不要对这些随便 `Close()`：

- `fmt.Println`：不需要关闭。
- `sync.WaitGroup`：没有 `Close()`。
- `context.Context`：没有 `Close()`，而是调用 `cancel()`。
- `http.DefaultClient`：通常不关闭。
- channel：只有发送方确认再也不会发送时才 `close(ch)`。

---

`scanner := bufio.NewScanner(file)`

`bufio.NewScanner()` 是 Go 标准库 **`bufio`（Buffered I/O，带缓冲的输入输出）** 中最常用的一个函数。

它的作用可以一句话概括：

> **把一个数据流（文件、网络、字符串等）包装成一个可以"一点一点读取"的扫描器（Scanner）。**

对于文件来说，它默认是**按行读取**。

在 Go 中，它最常见的用途就是**按行读取文本文件**：

          Reader（文件/网络/字符串）
                    │
                    ▼
          bufio.NewScanner()
                    │
                    ▼
          Scanner（扫描器）
                    │
        ┌───────────┴───────────┐
        ▼           ▼           ▼
     Scan()      Text()      Err()
     读取下一段    获取当前段    检查读取错误
`bufio` 是 Go 标准库中专门做**带缓冲 I/O（Buffered I/O）** 的包。

它主要提供了 **3 大类对象**：

| 类型      | 用途            | 最常用指数 |
| --------- | --------------- | ---------- |
| `Reader`  | 高效读取数据    | ⭐⭐⭐⭐⭐      |
| `Writer`  | 高效写入数据    | ⭐⭐⭐⭐       |
| `Scanner` | 按行/按单词扫描 | ⭐⭐⭐⭐⭐      |

为你将这份 Go `bufio` 标准库的学习笔记梳理整理为了 **核心组件对照表** 与 **API 详细索引表**，方便随查随用：

## 一、 Scanner / Reader / Writer 核心组件对比

| **组件分类**  | **适用场景**                                                 | **优点**                                                     | **缺点 / 常见坑点**                                        | **企业使用频率** |
| ------------- | ------------------------------------------------------------ | ------------------------------------------------------------ | ---------------------------------------------------------- | ---------------- |
| **`Scanner`** | 文本解析、按行读取日志、搜索关键词、按词分割文本             | 接口设计极佳，通过 `for scanner.Scan()` 遍历最直观           | 单行数据有默认缓冲区大小限制（64KB），超长行易溢出         | ★★★★★            |
| **`Reader`**  | 网络协议解析（TCP/HTTP Body）、二进制文件处理、单行超长文本读取 | 灵活度高，支持 `Peek`（偷看）、`Discard`（跳过）及按任意分隔符读取 | 无 `Scan()` 式的简洁循环结构，需要手动处理 `io.EOF`        | ★★★★★            |
| **`Writer`**  | 批量写入文件、构建网络响应数据流                             | 避免频繁写入磁盘/网络 IO，大大提升写入吞吐量                 | **必须显式调用 `Flush()`**，否则数据滞留缓冲区无法写入磁盘 | ★★★★★            |

## 二、 `bufio` 核心 API 快速检索表

### 1. Scanner 扫描器（文本处理首选）

| **API 方法 / 函数**            | **返回值类型**   | **核心作用与使用说明**                                       |
| ------------------------------ | ---------------- | ------------------------------------------------------------ |
| **`bufio.NewScanner(r)`**      | `*bufio.Scanner` | 创建一个以 `r` (实现 `io.Reader` 接口) 为源的扫描器。        |
| **`scanner.Scan()`**           | `bool`           | 尝试获取下一段 Token（默认是一行）。有数据返回 `true`，读完或报错返回 `false`。 |
| **`scanner.Text()`**           | `string`         | 获取刚刚扫描到的文本内容（已自动剥离换行符）。               |
| **`scanner.Bytes()`**          | `[]byte`         | 获取刚刚扫描到的字节切片（零拷贝，用于计算哈希或网络发送效率更高）。 |
| **`scanner.Err()`**            | `error`          | 扫描循环结束后检查是否出错（需要排除 `io.EOF`）。            |
| **`scanner.Buffer(buf, max)`** | 无               | 调整扫描时的初始缓冲区和最大 Token 限制（解决默认 64KB 超长行限制）。 |
| **`scanner.Split(splitFunc)`** | 无               | 自定义分割规则：如 `bufio.ScanLines`（按行）、`bufio.ScanWords`（按词）、`bufio.ScanBytes`（按字节）。 |

### 2. Reader 带缓冲读取器

| **API 方法 / 函数**            | **返回值类型**          | **核心作用与使用说明**                                       |
| ------------------------------ | ----------------------- | ------------------------------------------------------------ |
| **`bufio.NewReader(r)`**       | `*bufio.Reader`         | 创建一个带缓冲区的读取器。                                   |
| **`reader.ReadString(delim)`** | `(string, error)`       | 读取数据直到首次遇到分隔符 `delim`（如 `'\n'`），**结果保留分隔符**。 |
| **`reader.ReadBytes(delim)`**  | `([]byte, error)`       | 与 `ReadString` 功能一致，仅返回格式为字节切片。             |
| **`reader.Read(p)`**           | `(n int, error)`        | 读取最多 `len(p)` 字节的数据填充到 `p` 中（适合二进制处理）。 |
| **`reader.Peek(n)`**           | `([]byte, error)`       | **偷看前 `n` 字节数据**，但不会移动底层读取指针（后续读取操作仍能读到）。 |
| **`reader.Discard(n)`**        | `(discarded int, err)`  | 直接丢弃跳过接下来的 `n` 字节数据。                          |
| **`reader.ReadLine()`**        | `(line, isPrefix, err)` | 底层按行读取（企业级开发中更推荐 `ReadString('\n')` 或 `Scanner`）。 |

### 3. Writer 带缓冲写入器 & ReadWriter

| **API 方法 / 函数**             | **返回值类型**      | **核心作用与使用说明**                                       |
| ------------------------------- | ------------------- | ------------------------------------------------------------ |
| **`bufio.NewWriter(w)`**        | `*bufio.Writer`     | 创建一个带缓冲区的写入器。                                   |
| **`writer.WriteString(s)`**     | `(int, error)`      | 写入一个字符串到缓冲区（常用）。                             |
| **`writer.Write(b)`**           | `(int, error)`      | 写入一个字节切片到缓冲区。                                   |
| **`writer.Flush()`**            | `error`             | **最核心方法！** 将缓冲区中的残余数据真正刷入磁盘文件或网络连接（常用 `defer writer.Flush()`）。 |
| **`bufio.NewReadWriter(r, w)`** | `*bufio.ReadWriter` | 将一个 `Reader` 和一个 `Writer` 组合为一个同时具备读写能力的对象（常见于双向 TCP 网络编程）。 |

💡 必背核心清单（Go 后端高频 9 式）

```Go
// 1. 文本读取
scanner := bufio.NewScanner(file)
for scanner.Scan() {
    line := scanner.Text()
}
if err := scanner.Err(); err != nil { ... }

// 2. 网络/流式读取
reader := bufio.NewReader(conn)
line, err := reader.ReadString('\n')

// 3. 高效写入
writer := bufio.NewWriter(file)
defer writer.Flush() // 必写！
writer.WriteString("hello world\n")
```