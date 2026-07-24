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