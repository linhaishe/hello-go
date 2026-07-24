# 静态文件 HTTP 服务器 / 简易图床 API

> **目标**：体验 Go 原生强大的网络库，体会**几行代码搭一个高性能 Web 服务**的感觉。

- **功能需求**：
  - **接口 1**：文件上传接口（接收图片并保存到本地文件夹）。
  - **接口 2**：图片/静态资源预览接口（直接在浏览器访问图片路径即可查看）。
  - **接口 3**：文件列表查看（返回当前已上传文件的 JSON 列表）。

- **用到的 Go 特性**：
  - `net/http` 标准库（**完全不需要依赖外部 Web 框架**）。
  - `struct` 结构体定义与 `encoding/json` 序列化。
  - `Go` 原生并发处理 `HTTP` 请求（每个 `Request` 自动跑在独立 `Goroutine` 中）。

# run project

`go run ./cmd/mini-image-hosting`

# QA

## Goroutine 在哪里

`main.go` 里没有显式写 `go ...`，但你仍然在用 `goroutine`。

`http.ListenAndServe(addr, nil)` 启动服务器。
`net/http` 的默认 `ServeMux` 在内部为每个客户端请求创建一个独立的 `goroutine`，然后调用对应的 `handler`。
也就是说：
`uploadHandler`
`filesHandler`
`rootHandler`
这三个函数每次被请求时，都是在单独的 `goroutine` 中执行的。
所以`goroutine` 在哪里：

不是你显式写的，而是 `net/http` 帮你自动做的。\

## package

| **标准库包 (package)** | **核心能力描述**                                             | **在你的项目/脚本中典型的用途**                              |
| ---------------------- | ------------------------------------------------------------ | ------------------------------------------------------------ |
| **`encoding/json`**    | **JSON 数据序列化与反序列化** 提供 `Marshal`/`Unmarshal` 以及 `Encoder`/`Decoder`。 | 解析 API 返回的 JSON 格式数据，或将本地 Go 结构体转为 JSON 格式输出。 |
| **`fmt`**              | **格式化 I/O（输入/输出）** 包含 `Printf`、`Println`、`Sprintf` 等文本格式化与打印函数。 | 向终端控制台打印调试日志、输出运行结果，或拼接格式化字符串。 |
| **`io`**               | **基础 I/O 原语** 提供 `Reader`、`Writer` 接口以及 `ReadAll`、`Copy` 等流处理工具。 | 读取 HTTP 响应的 Body 数据流、复制文件内容或处理字节流。     |
| **`log`**              | **简易日志记录** 提供带有标准时间戳的日志输出工具（如 `log.Println`、`log.Fatal`）。 | 记录带有系统时间的运行日志，或在遇到不可恢复错误时打印日志并立即终止程序。 |
| **`net/http`**         | **HTTP 客户端与服务端** 提供 HTTP 请求发送（`http.Get` / `Post`）和 Web HTTP 服务器构建能力。 | 调用外部网络 API（比如请求图床接口）、下载文件或搭建 Web 服务。 |
| **`os`**               | **操作系统底层交互** 提供文件/目录创建与读写（`os.Open` / `Create`）、环境变量读取、命令行参数获取等。 | 打开/创建本地文件、读取写入图片数据、获取命令行参数或环境变量。 |
| **`path/filepath`**    | **跨平台的路径操作** 自动适配 Windows（`\`）和 Unix-like（`/`）的路径分隔符，提供 `Join`、`Ext`、`Base` 等函数。 | 提取文件的扩展名（如 `.png`）、安全地拼接文件保存路径（如 `filepath.Join(dir, filename)`）。 |
| **`time`**             | **时间与时区处理** 提供时间测量、格式化（`Format`）、定时器以及休眠（`time.Sleep`）功能。 | 记录请求耗时、为下载的文件生成带时间戳的文件名，或者做请求频控限制。 |

（通过 `net/http` 发请求，`encoding/json` 解析响应，`io` + `os` + `path/filepath` 把文件落盘到本地，并用 `log` / `fmt` 输出日志，用 `time` 做记录或控制延迟）。

File mode: https://pkg.go.dev/github.com/richelieu42/chimera/src/consts/fileMode

为什么目录通常是 0755，而文件通常是 0644

**目录**需要 `x`（执行）权限，才能进入目录，所以通常是 `0755`。

**普通文件**一般不需要执行权限，因此通常是 `0644`。

```go
// 创建目录
os.MkdirAll("uploads", 0755)

// 创建普通文件
os.WriteFile("config.json", data, 0644)
```

`log.Fatalf`: 如果创建目录失败，打印错误日志，然后**立即退出程序**。

---

**Go 原生标准库（`net/http`）的基于 Handler 模式的路由注册方式**。

如果从不同的技术维度来看，它可以细分为以下几个概念：

1. 架构与设计模式维度：**命令/ Handler 模式（Multiplexer / ServeMux）**

- **核心思想：** Go 的标准库提供了一个叫 `http.ServeMux`（HTTP 请求多路复用器）的对象。它的作用像是一个“交通指挥官”，将传入的 HTTP 请求路径（URL Path）映射匹配到对应的“处理器（Handler）”上。
- **分流逻辑：** 请求来了后，`ServeMux` 会根据最长前缀匹配原则，把请求分发给 `uploadHandler`、`filesHandler` 或静态文件服务器。

2. 编程接口维度：**标准库函数注册（Standard Library Routing）**

在 Go 生态中，这被称为 **"Stdlib / Native approach"（原生标准库方式）**，与使用第三方框架相对比：

- **与第三方框架对比：**
  - **你当前用的原生方式：** 依赖 Go 自带的 `net/http`，不需要安装任何第三方依赖。
  - **第三方路由框架方式：** 如 **Gin** (`router.POST("/upload", ...)`), **Fiber**, **Echo**, 或 **Chi**。第三方框架通常支持更复杂的路由（如路径参数 `/user/:id`、中间件链 `Use()`、按 HTTP 方法 `GET/POST` 区分等）。

| **函数**              | **使用的代码示例**                          | **接受的参数类型**                                           | **适用场景**                                                 |
| --------------------- | ------------------------------------------- | ------------------------------------------------------------ | ------------------------------------------------------------ |
| **`http.HandleFunc`** | `http.HandleFunc("/upload", uploadHandler)` | 接收一个**普通函数** `func(w http.ResponseWriter, r *http.Request)` | 处理纯业务逻辑（如表单提交、JSON 接口）。                    |
| **`http.Handle`**     | `http.Handle("/uploads/", ...)`             | 接收一个实现了 **`http.Handler` 接口的对象**（必须包含 `ServeHTTP` 方法） | 绑定结构体或标准库封装好的复杂组件（如 `http.FileServer` 静态文件服务、中间件等）。 |

`http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir)))`

把本地文件夹 `uploadDir` 里的文件，通过 `/uploads/` 这个网页 URL 安全地开放给别人访问。

1. **`http.Dir(uploadDir)`**

- **作用：** 相当于给你的本地磁盘路径（比如 `"./my_images"`）套了一个 Go 语言识别的“文件夹外壳”。

2. **`http.FileServer(...)`**

- **作用：** 生成一个**静态文件 HTTP 处理器**。
- **工作逻辑：** 当它收到一个请求路径（例如 `a.jpg`），它就会去前面指定的文件夹里找 `a.jpg` 并返还给浏览器。

3. **`http.StripPrefix("/uploads/", ...)`**

- **作用：** **剥离（裁剪）URL 前缀**。
- **工作逻辑：** 当收到请求时，先把 URL 开头的 `"/uploads/"` 砍掉，再把剩下的路径交给后面的 `FileServer` 处理。

`http.StripPrefix` 就像是一个**“路径转换器”**，负责把对外暴露的网址路径（`/uploads/cat.jpg`）裁剪适配成硬盘上真实的物理路径（`cat.jpg`），解决路由前缀与本地目录层级不匹配的问题。

假设你的本地文件夹 `uploadDir` 叫做 `./my_images`，里面有一张图片 `cat.jpg`。 你在代码里注册的路由是：`http.Handle("/uploads/", ...)`

当用户在浏览器里输入：`http://localhost:8080/uploads/cat.jpg` 时，后台会发生什么？

#### ❌ 错误写法（如果不加 StripPrefix）：

Go

```
// 错误示范：没有 StripPrefix
http.Handle("/uploads/", http.FileServer(http.Dir("./my_images")))
```

1. 浏览器请求路径：`/uploads/cat.jpg`
2. `FileServer` 拿到完整路径 `/uploads/cat.jpg`。
3. `FileServer` 拿着这个路径去本地目录 `./my_images` 里找。
4. **结果：** 它寻找的本地实际路径变成了 `./my_images/uploads/cat.jpg`！
5. 你的本地根本没有 `uploads` 这个子文件夹，**直接报 404 找不到文件**。

#### ✅ 正确写法（加了 StripPrefix）：

Go

```
// 正确示范
http.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./my_images"))))
```

1. 浏览器请求路径：`/uploads/cat.jpg`
2. `StripPrefix` 拦截到请求，把开头的 `/uploads/` **切掉**。
3. 剩余的请求路径只剩下：`cat.jpg`。
4. `StripPrefix` 把 `cat.jpg` 交给内部的 `FileServer`。
5. `FileServer` 拿着 `cat.jpg` 去本地 `./my_images` 找。
6. **结果：** 成功找到 `./my_images/cat.jpg`，图片顺利显示！

### http.ListenAndServe

```
func ListenAndServe(addr string, handler Handler) error
```

```go
	log.Fatal(http.ListenAndServe(":8080", nil))
```

看有这些参数，为啥第二个参数是nil

**传 `nil`** ＝ “我懒得自己创建路由对象了，请直接使用我刚才通过 `http.HandleFunc` 注册好的全局默认路由 `http.DefaultServeMux`。”

**传自定义对象** ＝ “使用我自己创建的路由对象（或者 Gin/Echo 等框架路由），不要用全局路由。”

虽然传 `nil` 写起来很省事（非常适合写 Demo 或小工具），但在**大型生产项目**或**使用第三方框架**时，大家通常**不会**传 `nil`：

为了更严谨，推荐显式实例化一个全新的 `ServeMux`：

```Go
// 1. 自己创建一个独立的路由匹配器
mux := http.NewServeMux()

// 2. 绑定路由到你自己的 mux 上
mux.HandleFunc("/upload", uploadHandler)

// 3. 明确传给 ListenAndServe（或 ListenAndServeTLS），不使用全局 nil
http.ListenAndServe(":8080", mux)
```

```Go
func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Mini Image Hosting</title>
</head>
<body>
<h1>Mini Image Hosting</h1>
<form action="/upload" method="post" enctype="multipart/form-data">
  <label for="file">Select image:</label>
  <input type="file" id="file" name="file" accept="image/*" required>
  <button type="submit">Upload</button>
</form>
<p>Uploaded files: <a href="/files">/files</a></p>
<p>Preview images directly at <code>/uploads/&lt;filename&gt;</code>.</p>
</body>
</html>`)
}
```

这段代码是 Go 语言中一个典型的 **HTTP 根路径处理函数（Handler）**。它的主要作用是：当用户访问首页 `http://localhost:8080/` 时，给用户的浏览器返回一个带有**文件上传表单**的 HTML 网页。

下面将从 **函数参数**、**安全校验逻辑** 以及 **返回的 HTML 网页内容** 三个部分为你详细解释。

1. 函数参数解释

Go 的 HTTP 处理函数都必须遵循固定的签名：`func(w http.ResponseWriter, r *http.Request)`。

| **参数** | **类型**              | **含义与作用**                                               |
| -------- | --------------------- | ------------------------------------------------------------ |
| **`w`**  | `http.ResponseWriter` | **服务端响应的“出口”**（接口类型）。 用它来向浏览器写回数据，比如设置响应头（`Header`）、状态码（如 `200` 或 `404`）、以及具体的网页 HTML 内容。 |
| **`r`**  | `*http.Request`       | **客户端请求的“入口”**（指针类型）。 包含浏览器发来的所有信息，比如请求方法（`GET`/`POST`）、请求 URL 路径、请求头、用户提交的表单数据以及 Cookie 等。 |

2. 代码逻辑解释

拦截非法路径（防止 404 误判）

Go

```
if r.URL.Path != "/" {
    http.NotFound(w, r)
    return
}
```

- **为什么要写这段？**

  在 Go 原生的 `http.HandleFunc("/", rootHandler)` 中，`"/"` 是一个**前缀通配符**。如果用户在地址栏随意输入了一个不存在的路径（例如 `http://localhost:8080/abc`），Go 的路由引擎也会默认匹配到这个 `rootHandler` 来处理。

- **处理逻辑：**

  判断 `r.URL.Path` 是否严格等于 `"/"`。如果不等于（说明用户访问了未定义的路径），直接调用 `http.NotFound(w, r)` 给浏览器返回标准的 **404 Not Found** 页面，并立即 `return` 结束，不再展示首页。

设置响应头与输出网页

Go

```
w.Header().Set("Content-Type", "text/html; charset=utf-8")
fmt.Fprint(w, `...`)
```

- **`w.Header().Set(...)`**：告诉浏览器“我接下来发给你的是 **UTF-8 编码的 HTML 网页**，请按网页渲染，不要把它当纯文本或文件下载”。
- **`fmt.Fprint(w, ...)`**：把后面的 HTML 字符串**写入到 `w`（响应流）** 中，发送给用户的浏览器。

```Go
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { 
    // HTTP 动词校验（只允许 POST）如果用户尝试通过 GET 或其他请求方式访问 /upload，直接拒绝。
		w.Header().Set("Allow", http.MethodPost) 
    // 按照 HTTP 规范，当返回 405 错误时，需要在 Header 中告诉客户端允许的请求方式。
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed) 
    // 向客户端返回 405 Method Not Allowed 状态码和提示文本。
		return
	}
	// 解析上传的表单数据
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
    // maxUploadSize（如 10 << 20 即 10MB）：限制存入内存的最大字节数。如果文件大小超过这个限制，多余的数据会被临时存放在磁盘的临时文件中，防止恶意用户上传超大文件直接把内存撑爆（OOM）。
    // r.ParseMultipartForm(maxUploadSize)：解析前端通过 multipart/form-data 格式发来的二进制数据。
		log.Printf("upload parse error: %v", err)
		http.Error(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
  // r.FormFile("file")：这里的 "file" 必须与 HTML 表单中 <input name="file"> 的 name 属性保持一致。
	if err != nil {
		log.Printf("upload form file error: %v", err)
		http.Error(w, "missing file field", http.StatusBadRequest)
		return
	}
	defer file.Close() // 延迟关闭打开的文件流，保证函数退出时释放资源。

	filename := fmt.Sprintf("%d-%s", time.Now().UnixNano(), filepath.Base(header.Filename))
  // time.Now().UnixNano() 唯一化前缀：在原始文件名前加上纳秒级时间戳（例如 1718000000000-cat.jpg），防止不同用户上传同名文件（如 avatar.png）时互相覆盖。
  // filepath.Base(...) 防路径穿越攻击：黑客可能会构造恶意文件名如 ../../etc/passwd。filepath.Base() 会强行裁剪只保留最后的纯文件名（即 passwd），消除安全隐患。
	dstPath := filepath.Join(uploadDir, filename)
  // 跨平台路径拼接：安全拼接保存目录与文件名（例如把 uploads 和 123-cat.jpg 拼接为 uploads/123-cat.jpg）。

  // 在本地磁盘创建目标文件
	dst, err := os.Create(dstPath) // 在服务器磁盘上创建一个新的空文件（如果已存在则会被清空）。
	if err != nil {
		http.Error(w, "failed to create file", http.StatusInternalServerError)
		return
	}
	defer dst.Close() // 确保落盘完成后，及时关闭磁盘文件句柄。

  // 数据流复制（文件落盘）
	if _, err := io.Copy(dst, file); err != nil { // Go 极其高效的流式传输操作。
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}
  // 它把用户上传的 file 数据流（Reader）一块一块地写入本地磁盘 dst 文件（Writer）。这种方式不需要把整个文件一次性读进内存，即便上传上百兆的大文件，内存占用也极低。

  // 返回 JSON 响应给前端
	response := struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}{
		Name: filename,
		URL:  "/uploads/" + filename,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response) // 设置 HTTP 响应头为 JSON 格式，并将结构体序列化为 JSON 字符串写回客户端。
}
```

`%d-%s` 是 Go 语言（以及 C、Python 等很多编程语言）中 **`fmt.Sprintf` 格式化字符串的占位符组合**。

在你的代码片段中：

Go

```
filename := fmt.Sprintf("%d-%s", time.Now().UnixNano(), filepath.Base(header.Filename))
```

它的作用是：**把后面的两个变量按顺序替换掉这两个占位符，拼接成一个新的字符串。**

### 1. 拆解这两个占位符

- **`%d`**（Decimal）：代表一个**十进制整数**。
  - 对应传入的第 1 个参数：`time.Now().UnixNano()`（当前的纳秒级时间戳，比如 `1718000000000000000`）。
- **`-`**：就是普通的中划线字符，原样输出，作为分隔符。
- **`%s`**（String）：代表一个**字符串**。
  - 对应传入的第 2 个参数：`filepath.Base(header.Filename)`（用户上传的原始文件名，比如 `cat.jpg`）。

### 2. 实际拼接效果示例

假设用户在 **14:30:00** 上传了一张名为 **`cat.jpg`** 的图片：

1. `time.Now().UnixNano()` 算出的时间戳是：`1718000000000000000`
2. `filepath.Base(...)` 拿到的文件名是：`cat.jpg`
3. 经过 `%d-%s` 替换拼接后，最终得到的 `filename` 就是：

`1718000000000000000-cat.jpg`

```Go
// “查询已上传文件列表” 接口
// 读取存储目录 ➔ 遍历里面的所有文件 ➔ 过滤掉文件夹并提取文件信息（文件名、URL、大小） ➔ 组装成 JSON 数组返回给前端。
func filesHandler(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		http.Error(w, "failed to read upload directory", http.StatusInternalServerError)
		return
	}
// os.ReadDir(uploadDir)：Go 1.16+ 推荐的目录读取函数。它会打开 uploadDir（比如 ./uploads），并返回该目录下所有文件和子文件夹的列表（[]os.DirEntry）。
// 错误处理：如果读取失败（比如文件夹不存在或没有读取权限），向客户端返回 500 Internal Server Error 错误并退出。

	var files []FileInfo
  // 遍历目录条目并提取元数据
	for _, entry := range entries {
		if entry.IsDir() {
			continue // 过滤掉子文件夹
		}
		info, err := entry.Info()
		if err != nil {
			continue // 获取单个文件属性失败时直接跳过
		}
		files = append(files, FileInfo{
			Name: entry.Name(),
			URL:  "/uploads/" + entry.Name(),
			Size: info.Size(),
		})
	}

  // 将结果序列化为 JSON 返回给客户端
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}
```

**`if entry.IsDir() { continue }`**：我们只想列出上传的图片文件。如果目录里不小心创建了子文件夹，直接跳过。

**`entry.Info()`**：获取该文件的详细属性对象（`os.FileInfo`），里面包含文件大小、修改时间、权限等。

**构建 `FileInfo` 结构体**：

- **`entry.Name()`**：文件名（如 `1718000000000-cat.jpg`）。
- **`"/uploads/" + entry.Name()`**：拼接出可以直接在浏览器打开或展示图片的绝对路径 URL。
- **`info.Size()`**：文件的实际物理大小（单位是字节 Byte）。

**`append(files, ...)`**：把构建好的文件信息追加到切片（列表）中。