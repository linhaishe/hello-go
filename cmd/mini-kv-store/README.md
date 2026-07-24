## 简易 Redis/KV 内存数据库 (Mini KV-Store)

> **目标**：理解指针、Map 结构以及**并发安全锁**的使用。

- **功能需求**：
  - 基于内存实现一个简单的 Key-Value 存储系统。
  - 支持基本命令：`SET key value`、`GET key`、`DELETE key`。
  - 支持给 Key 设置过期时间（TTL），过期后自动清理。
  - （可选扩展）通过 TCP 暴露服务（使用 `net` 包），可以用 `telnet` 或 `nc` 连接测试。
- **用到的 Go 特性**：
  - `map[string]interface{}` 存取内存数据。
  - `sync.RWMutex`（读写锁）保证并发读写安全。
  - `time.AfterFunc` 或后台协程清理过期 Key。

## 使用方法

### 1. 运行

在仓库根目录执行：

```bash
go run ./cmd/mini-kv-store
```

### 2. 输入命令

示例命令：

```text
SET name alice
GET name
DELETE name
GET name
EXIT
```

### 3. 可支持的命令

- `SET key value [ttl]`
- `GET key`
- `DELETE key`
- `HELP`
- `EXIT`

### 4. TTL 示例

```text
SET name alice 1s
```

表示这个键会在 1 秒后自动过期，后台清理器会定时扫描并删除过期项。

### 5. 运行效果

程序启动后会打印：

```text
mini kv-store started
commands: SET key value [ttl], GET key, DELETE key, HELP, EXIT
```

然后进入交互式命令行，直到你输入 `EXIT` 为止。