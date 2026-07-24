# mini-image-hosting

这是一个用 Go 原生 `net/http` 实现的简易图床示例程序。

## 目标

- 体验 Go 标准库自带的 Web 服务器能力
- 不依赖第三方框架
- 用最少代码完成图片上传、图片预览、文件列表接口

## 功能

1. `POST /upload`
   - 接收上传的图片文件
   - 保存到本地 `uploads/` 目录

2. `GET /uploads/<filename>`
   - 直接访问图片路径预览

3. `GET /files`
   - 返回当前上传文件的 JSON 列表

## 用到的 Go 特性

- `net/http`：标准库 HTTP 服务与路由
- `struct`：定义响应结构体
- `encoding/json`：序列化 JSON 数据
- `http.ServeMux`：手动注册路由
- Go 原生并发：每个请求由 `net/http` 自动在独立 Goroutine 中处理

## 运行方式

```bash
cd /Users/chenruo/Documents/GitHub/hello-go
go run ./cmd/mini-image-hosting
```

打开浏览器访问：

```text
http://localhost:8080
```

## 目录结构

```text
cmd/mini-image-hosting/
├── main.go
├── README.md
└── uploads/
```

## 说明

- `main.go` 是这个示例程序的入口
- `uploads/` 是上传图片的存储目录
- 这个示例完全使用 Go 标准库，不需要额外安装 Web 框架
