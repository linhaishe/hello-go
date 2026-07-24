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