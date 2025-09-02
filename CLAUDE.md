# CLAUDE.md

此文件为 Claude Code (claude.ai/code) 在此代码仓库中工作时提供指导。

## 项目概述

gt (go-transfer) 是一个高性能流式文件传输工具，采用零缓存设计。支持三种模式：接收器（存储文件）、转发器（无缓存中继）和客户端（上传文件/目录）。

## 常用命令

### 构建命令
```bash
# 构建当前平台二进制文件
go build -o gt

# 构建所有平台版本（输出到 dist/）
./build.sh

# 构建到指定输出目录
mkdir -p dist && go build -o dist/gt
```

### 运行命令
```bash
# 交互式配置运行
./gt

# 日志级别控制
./gt -s        # 静默模式（最少输出）
./gt -v        # 详细模式（详细日志）
./gt --debug   # 调试模式（所有调试信息）
```

### 测试
```bash
# 运行测试
go test ./...

# 测试文件上传
curl -X POST "http://localhost:17002/upload?name=test.txt" --data-binary @test.txt

# 访问 API 文档
open http://localhost:17002/docs
```

## 架构与核心概念

### 三种工作模式
应用程序在三种互斥模式之一下运行，首次运行时交互式配置：

1. **接收器模式**：接收文件并本地存储
   - 监听配置的端口（默认：17002）
   - 在配置路径存储文件（默认：~/uploads）
   - 保留上传目录的目录结构

2. **转发器模式**：零缓存中继到下一个服务器
   - 使用 `io.Pipe()` 实现真正的流式传输，无磁盘 I/O
   - 每次传输维护两个 goroutine（从客户端读取，写入目标）
   - 对链式部署至关重要（客户端 → 转发器 → 转发器 → 接收器）

3. **客户端模式**：上传文件/目录
   - 单文件：仅用文件名上传（无路径）
   - 目录：保留完整目录结构
   - 使用单个 HTTP 连接与 keep-alive 避免 Windows 端口耗尽

### 关键技术决策

**零缓存架构**：转发模式使用 `io.Pipe()` 将数据从传入请求直接流式传输到传出请求，无需缓冲到磁盘。这在 `stream.go` 的 `handleForwardFile()` 和 `handleStreamForward()` 函数中实现。

**统一进度系统**：所有进度跟踪都使用 `progress.go` 中的集中式 `Progress` 类型。这替换了三个独立的实现，在客户端上传、服务器接收和转发操作中提供一致的进度显示。

**结构化日志记录**：`logger.go` 中的自定义日志记录器，具有日志级别（DEBUG/INFO/WARN/ERROR/SILENT）。替换所有直接的 `fmt.Printf` 和 `log.Printf` 调用。

**配置持久化**：首次运行后配置保存到 `~/.config/go-transfer/config.yaml`。后续运行使用保存的配置，除非用户选择重新配置。

**API 文档**：内置 Swagger UI 在 `/docs` 端点提供服务。基于可用端点和当前服务器配置自动生成 OpenAPI 规范。

### 文件组织

核心流式传输逻辑分布在：
- `stream.go`：服务器端流式处理程序（接收/转发）
- `client.go`：客户端上传逻辑
- `progress.go`：统一进度跟踪
- `logger.go`：结构化日志系统

配置和工具：
- `config.go`：交互式配置和 YAML 持久化
- `constants.go`：所有硬编码值和超时设置
- `port.go`：端口可用性检查和进程管理
- `utils.go`：共享工具（formatSize、expandPath 等）
- `swagger.go`：自动生成的 API 文档和 Swagger UI

### 关键实现细节

**端口优化**（client.go）：
- 强制使用 HTTP/1.1（禁用 HTTP/2 多路复用）
- 限制为单个连接并使用 keep-alive
- 在端口耗尽错误时实施指数退避

**目录上传保留**（client.go、stream.go）：
- 客户端发送文件时保留相对路径
- 服务器根据需要创建目录
- 示例：上传 `mydir/` 保留 `mydir/sub/file.txt` 结构

**流缓冲区大小**：
- 小（256KB）：用于流式传输以避免背压
- 中（512KB）：HTTP 传输缓冲区
- 大（4MB）：本地文件操作和响应复制

## 开发说明

### 添加新功能
- 所有面向用户的消息都应使用日志函数（LogInfo、LogWarn、LogError、LogSuccess）
- 进度显示必须使用来自 progress.go 的统一 Progress 类型
- 新的配置选项应添加到 config.go 中的 Config 结构
- API 端点应在 swagger.go 中记录以实现自动 OpenAPI 生成

### 测试文件传输
```bash
# 启动接收器
./gt  # 选择选项 1（接收器）

# 在另一个终端中上传文件
./gt  # 选择选项 3（客户端）
# 在提示时输入文件路径

# 或使用 curl 快速测试
curl -X POST "http://localhost:17002/upload?name=test.txt" --data-binary @test.txt
```

### 调试传输问题
- 使用 `--debug` 标志查看所有调试日志
- 转发模式问题通常与目标服务器连接性相关