## 简易 Redis/KV 内存数据库 (Mini KV-Store)

> **目标**：理解指针、Map 结构以及**并发安全锁**的使用。
>
> 实现了一个**带 TTL（过期时间）和并发安全（读写锁）的内存 KV 数据库**（功能类似于一个简易版的 Redis）。

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

# QA

**指针接收者 `(s *KVStore)`**： 所有方法都使用了指针接收者。这样在方法内部修改 `s.items`（如插入或删除 key）时，修改的是同一个数据库实例，而不是副本。

**读写锁 `sync.RWMutex` (`s.mu`)**： 在 Go 中，标准的 `map` 是**线程不安全的**（并发读写会导致程序直接 `panic` 崩溃）。

- **写锁 `s.mu.Lock()`**：完全独占。`Set`、`Delete`、`cleanupExpired` 会修改 `map`，此时不允许其他 Goroutine 读或写。
- **读锁 `s.mu.RLock()`**：多读共享。`Get` 只读取 `map`，多个 Goroutine 可以同时执行 `Get`，提高了并发性能。

---



```Go
func NewKVStore() *KVStore {\
	return &KVStore{items: make(map[string]entry)}
}
```

这个*和&是啥意思

在 Go 语言中，`*` 和 `&` 是 **指针（Pointer）** 的核心符号：

- **`*`（位于类型前）：表示“这是一个指针类型”**
- **`&`（位于变量/结构体前）：表示“取地址 / 创建指针”**

简单来说：**`&` 负责制造指针，`*` 负责标记指针类型。**

### 1. `*KVStore`（返回值类型）

在函数签名 `func NewKVStore() *KVStore` 中：

- 如果写 `KVStore`，代表函数返回的是**结构体本身（值传递）**，Go 会在返回时把整个结构体**完整复制一份**给调用者。
- 写了 `*KVStore`，代表函数返回的是**指向 `KVStore` 结构体的内存地址（指针传递）**。

> 💡 **比喻：**
>
> - 返回 `KVStore` 相当于送给你一本 **1000 页的书（实体）**；
> - 返回 `*KVStore` 相当于只在便签纸上写下 **那本书放在哪个书架（地址）** 给您。

### 2. `&KVStore{...}`（返回值内容）

在函数体 `return &KVStore{items: ...}` 中：

- 如果写 `KVStore{...}`，只是在内存中新建了一个结构体实例。
- 在前面加了 `&`，意思就是：**“请给我这个刚刚新建的结构体的内存地址”**（即生成一个指针）。

### 3. 组合在一起看这段代码

```Go
// 函数声明：我要返回一个指向 KVStore 的“指针”（内存地址）
func NewKVStore() *KVStore {
    // 函数体：创建一个 KVStore 结构体，并用 & 拿到它的“指针”返回出去
    return &KVStore{
        items: make(map[string]entry),
    }
}
```

### 💡 为什么 Go 的构造函数（Factory Function）都喜欢用 `*` 和 `&`？

1. **性能更好（零拷贝）：** 结构体内部可能包含很多字段或复杂数据，如果直接传值，每次传递都会触发一次完整内存复制；而传递指针（地址）只需要复制几字节的内存地址。
2. **支持修改原数据：** 在 Go 中，如果你想让后续的方法（如 `store.Set("key", "val")`）能够修改这个 KVStore 里的 `items`，你**必须**拿到它的指针。如果拿到的是值副本，你所有的修改都只会作用在副本上，原来的 KVStore 不会有任何变化。

---

comma-ok idiom（逗号 ok 模式）

---

```Go
func NewKVStore() *KVStore {
	return &KVStore{items: make(map[string]entry)}
}
```

简单来说，这个函数就是 KVStore（键值数据库）的“建造者”或“初始化工厂”（在 Go 语言里通常叫 **构造函数 / Factory Function**）。

它的核心作用是：**在内存中创建一个全新的、准备就绪的 `KVStore` 数据库实例，并把它的内存地址（指针）返回给你。**

### 拆解里面的每一项操作

```Go
func NewKVStore() *KVStore {
    // 实例化结构体，并初始化内部的 map，然后用 & 拿到它的指针返回
    return &KVStore{
        items: make(map[string]entry),
    }
}
```

1. **`make(map[string]entry)` —— 最关键的一步！**
   - 在 Go 语言中，`map` 的默认零值是 `nil`（空指针）。
   - **如果不调用 `make` 初始化，直接对一个 `nil` map 进行写入操作（比如 `store.items["a"] = entry`），程序会直接 `panic` 崩溃！**
   - 所以这个函数帮你把底层的 `items` 提前用 `make` 准备好（分配了内存）。
2. **`&KVStore{ ... }`**
   - 用 `&` 拿到这个新创建的数据库实例的**内存地址（指针）**。
3. **`func NewKVStore() *KVStore`**
   - 返回值类型是 `*KVStore`，表明它交给你的是一个指针。这样后续你无论在哪个函数里调用 `store.Set()` 或 `store.Get()`，操作的都是**同一个内存里的数据库**，而不是复制出来的副本。

💡 怎么使用它？

在 `main` 函数里，你只需要写一行代码：

```Go
func main() {
    // 创建并初始化一个 KVStore 实例
    store := NewKVStore()

    // 接下来就可以安全地使用了，绝对不会因为 map 没初始化而报错
    store.Set("name", "bob", 0)
}
```