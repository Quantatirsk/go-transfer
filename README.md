# go-transfer

纯流式文件传输工具，专为内网稳定环境设计，支持零缓存转发。

## 特性

- **纯流式传输** - 无需复杂的分块参数，直接流式上传
- **零缓存转发** - 中继服务器不占用磁盘空间，使用 `io.Pipe()` 实现
- **极简配置** - 只需配置端口和目标地址
- **支持超大文件** - 10GB、100GB文件都能轻松处理
- **三种模式** - receiver（接收）、relay（中继）、gateway（网关）
- **端口冲突处理** - 自动检测端口占用，支持一键释放

## 硬编码参数

以下参数已内置，无需配置：
- 监听地址: `0.0.0.0`
- 最大文件: `16GB`
- 日志级别: `info`

## 快速开始

### 安装

```bash
go build -o go-transfer
```

### 运行

首次运行会启动配置向导：
```bash
./go-transfer
```

## 配置示例

每种模式只需要最少的配置参数：

### Receiver（接收文件）
```yaml
mode: receiver
port: 17002
storage_path: ~/uploads
```

### Relay（中继转发）
```yaml
mode: relay
port: 17002
target_url: http://192.168.1.100:17002
```

### Gateway（网关入口）
```yaml
mode: gateway
port: 8080
target_url: http://relay-server:17002
```

## 使用方法

### 上传文件

支持两种上传方式：

#### 1. 浏览器上传（FormData）
- 访问 `http://localhost:port/docs` 
- 在 Swagger UI 界面直接选择文件上传
- 支持拖拽文件

#### 2. 命令行上传

```bash
# 方式1: 二进制流上传（推荐用于大文件）
curl -X POST "http://localhost:8080/upload?name=myfile.zip" \
     --data-binary @myfile.zip

# 方式2: FormData上传（自动获取文件名）
curl -X POST "http://localhost:8080/upload" \
     -F "file=@myfile.zip"

# 带进度显示
curl -X POST "http://localhost:8080/upload?name=huge.tar.gz" \
     --data-binary @huge.tar.gz \
     --progress-bar
```

### API文档

启动服务后访问：`http://localhost:port/docs`

## 部署架构

### 典型三层架构

```
客户端 → Gateway(:8080) → Relay(:17002) → Receiver(:17002)
         [入口服务器]      [中继服务器]     [存储服务器]
```

### 部署步骤

1. **部署 Receiver**（存储服务器）
   ```bash
   # 运行配置向导，选择 receiver 模式
   ./go-transfer
   # 配置端口：17002
   # 配置存储路径：/data/uploads
   ```

2. **部署 Relay**（中继服务器）
   ```bash
   # 运行配置向导，选择 relay 模式
   ./go-transfer
   # 配置端口：17002
   # 配置目标：http://receiver-ip:17002
   ```

3. **部署 Gateway**（入口服务器）
   ```bash
   # 运行配置向导，选择 gateway 模式
   ./go-transfer
   # 配置端口：8080
   # 配置目标：http://relay-ip:17002
   ```

## 零缓存原理

中继服务器使用 Go 的 `io.Pipe()` 实现真正的流式转发：

```go
// 创建管道
pipeReader, pipeWriter := io.Pipe()

// 协程1: 从客户端读取数据写入管道
go func() {
    io.Copy(pipeWriter, request.Body)
}()

// 协程2: 从管道读取数据转发到目标
go func() {
    http.Post(targetURL, "application/octet-stream", pipeReader)
}()
```

这种设计确保：
- 数据不落盘，直接在内存中转发
- 中继服务器可以只有很小的磁盘空间（如100MB）
- 支持任意大小的文件传输

## 实时进度显示

系统提供实时的传输进度显示：

### 特性
- **即时响应**：文件传输开始时立即显示信息
- **实时进度**：每秒更新传输进度、速度和剩余时间
- **性能指标**：显示传输速度（MB/s）和总耗时

### 显示示例

```
⬇️  开始接收: myfile.zip (预计 109.61 MB)
   接收进度 myfile.zip: 23.5% (25.78/109.61 MB, 12.34 MB/s, 剩余 7s)
   接收进度 myfile.zip: 47.2% (51.73/109.61 MB, 11.89 MB/s, 剩余 5s)
   接收进度 myfile.zip: 71.8% (78.69/109.61 MB, 11.56 MB/s, 剩余 3s)
✅ 文件已保存: myfile.zip (109.61 MB, 11.42 MB/s, 耗时 9.6s)
```

中继服务器显示：
```
🔄 开始转发: myfile.zip (预计 109.61 MB) → http://api.example.com:17002
   上传进度 myfile.zip: 15.3% (16.75/109.61 MB, 8.76 MB/s, 剩余 11s)
✅ 成功转发: myfile.zip (109.61 MB, 9.23 MB/s, 耗时 11.9s)
```

## API端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/upload?name=filename` | POST | 流式上传文件 |
| `/status` | GET | 服务状态和健康检查 |
| `/docs` | GET | API文档 |

## 系统要求

- Go 1.19+
- Linux/macOS/Windows
- 内网环境（稳定网络）

## 性能优化

- 使用纯流式传输，无需缓存
- 支持长连接，适合大文件
- 自动处理断点续传（客户端支持）

## 端口冲突处理

当端口被占用时，程序会：

1. 自动检测端口占用情况
2. 显示占用进程信息（名称和PID）
3. 询问用户是否杀死占用进程
4. 自动释放端口并启动服务

示例：
```
⚠️  端口 17002 已被占用
占用进程: nginx (PID: 1234)

是否杀死该进程并释放端口? [y/N]: y
✅ 已杀死进程 nginx (PID: 1234)
等待端口释放... 完成!
```

## 常见问题

**Q: 中继服务器需要多大磁盘空间？**
A: 几乎不需要，100MB足够运行程序本身。所有数据都是流式转发，不占用磁盘。

**Q: 支持多大的文件？**
A: 理论上无限制。已测试10GB+文件传输无问题。

**Q: 网络中断怎么办？**
A: 客户端需要支持断点续传。推荐使用支持resume的工具。

**Q: 端口被占用怎么办？**
A: 程序会自动检测并显示占用进程，询问是否杀死该进程释放端口。

## 许可证

MIT License