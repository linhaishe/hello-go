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