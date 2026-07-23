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

不是你显式写的，而是 `net/http` 帮你自动做的。