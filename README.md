# go-transfer

高性能流式文件传输工具，专为内网环境设计，支持零缓存转发和高速传输。

## 特性

- **纯流式传输** - 无需复杂的分块参数，直接流式上传
- **零缓存转发** - 转发服务器不占用磁盘空间，使用 `io.Pipe()` 实现
- **高速传输优化** - 支持 30MB/s+ 传输速度，4MB 缓冲区优化
- **极简配置** - 只需配置端口和目标地址
- **支持超大文件** - 10GB、100GB文件都能轻松处理
- **两种模式** - receiver（接收）、forward（转发）
- **文件覆盖** - 自动覆盖同名文件，无需手动清理
- **端口冲突处理** - 自动检测端口占用，支持一键释放
- **Nginx 代理支持** - 完整的反向代理配置，支持端口复用

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

### Forward（转发文件）
```yaml
mode: forward
port: 17002
target_url: http://192.168.1.100:17002
```

## 使用方法

### 上传文件

支持多种上传方式：

#### 1. 浏览器上传（FormData）
- 访问 `http://localhost:port/docs` 
- 在 Swagger UI 界面直接选择文件上传
- 支持拖拽文件

#### 2. 命令行上传（优雅的进度显示）

**方式1: Python客户端（推荐，最优雅）**
```bash
# 无需安装依赖，直接使用
python3 transfer.py myfile.zip http://192.168.1.100:17001
```

显示效果：
```
📁 文件: myfile.zip
📊 大小: 109.61 MB
🎯 目标: http://192.168.1.100:17001

上传进度: [████████████████████░░░░░░░░░░░░░░░░░░░] 52.3% (57.36 MB/109.61 MB) 速度: 12.45 MB/s 剩余: 4秒
```

**方式2: Shell脚本（需要pv）**
```bash
# 使用 transfer.sh 脚本
./transfer.sh myfile.zip http://192.168.1.100:17001
```

**方式3: 直接使用 pv + curl**
```bash
# 安装 pv: brew install pv (macOS) 或 apt install pv (Linux)
pv myfile.zip | curl -X POST "http://localhost:8080/upload?name=myfile.zip" \
     -H "Content-Type: application/octet-stream" \
     --data-binary @-
```

**方式4: 基础上传（无客户端进度）**
```bash
curl -X POST "http://localhost:8080/upload?name=myfile.zip" \
     --data-binary @myfile.zip
```

### API文档

启动服务后访问：`http://localhost:port/docs`

## 部署架构

### 典型多层架构

```
客户端 → Forward(:8080) → Forward(:17002) → Receiver(:17002)
         [入口服务器]      [转发服务器]       [存储服务器]
```

### 部署步骤

1. **部署 Receiver**（存储服务器）
   ```bash
   # 运行配置向导，选择 receiver 模式
   ./go-transfer
   # 配置端口：17002
   # 配置存储路径：/data/uploads
   ```

2. **部署 Forward**（转发服务器，可多层）
   ```bash
   # 运行配置向导，选择 forward 模式
   ./go-transfer
   # 配置端口：17002
   # 配置目标：http://next-server:17002
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
| `/upload?name=filename` | POST | 流式上传文件（支持重名覆盖） |
| `/status` | GET | 服务状态和健康检查 |
| `/docs` | GET | Swagger API文档界面 |
| `/swagger.json` | GET | Swagger JSON文档 |

## 系统要求

- Go 1.19+
- Linux/macOS/Windows
- 内网环境（稳定网络）

## 性能优化

### 高速传输优化
- **4MB 缓冲区** - 优化 `io.CopyBuffer` 提升传输效率
- **HTTP Transport 优化** - 自定义读写缓冲区和连接池
- **2小时超时** - 支持超大文件长时间传输
- **TCP 优化** - Nginx 配置 `tcp_nodelay` 减少延迟

### 流式传输
- 使用 `io.Pipe()` 实现真正的零缓存转发
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

## Nginx 代理配置

支持通过 Nginx 反向代理访问，配置文件 `nginx.conf` 提供了完整的代理设置：

```nginx
server {
    listen 5000;
    location /sender/ {
        proxy_pass http://127.0.0.1:17002;
        proxy_buffering off;
        proxy_request_buffering off;
        # 更多配置见 nginx.conf
    }
}
```

访问示例：
- 上传：`http://localhost:5000/sender/upload`
- 文档：`http://localhost:5000/sender/docs`
- 状态：`http://localhost:5000/sender/status`

## 常见问题

**Q: 中继服务器需要多大磁盘空间？**
A: 几乎不需要，100MB足够运行程序本身。所有数据都是流式转发，不占用磁盘。

**Q: 支持多大的文件？**
A: 理论上无限制。已测试10GB+文件传输无问题，支持30MB/s+传输速度。

**Q: 网络中断怎么办？**
A: 客户端需要支持断点续传。推荐使用支持resume的工具。

**Q: 端口被占用怎么办？**
A: 程序会自动检测并显示占用进程，询问是否杀死该进程释放端口。

**Q: 文件重名怎么处理？**
A: 自动覆盖已存在的文件，并在日志中显示警告信息。

**Q: 如何支持高速传输？**
A: 已优化支持30MB/s+传输速度，使用4MB缓冲区和优化的HTTP Transport配置。

## 项目结构

```
go-transfer/
├── main.go              # 主程序入口
├── config.go            # 配置管理和交互式向导
├── stream.go            # 核心流式传输逻辑（优化版）
├── swagger.go           # Swagger API文档生成
├── port.go              # 端口检测和管理
├── build.sh             # 多平台构建脚本
├── go.mod/go.sum        # Go模块依赖
├── README.md            # 项目文档
│
├── transfer.py          # Python上传客户端（带进度条）
├── transfer.sh          # Shell上传脚本（使用pv）
├── nginx.conf           # Nginx反向代理配置（优化版）
├── nginx-usage.md       # Nginx使用说明
│
├── diagnose-transfer.sh # 传输诊断工具
├── test-nginx-proxy.sh  # Nginx代理测试脚本
│
└── go-transfer-*        # 各平台编译后的二进制文件
    ├── linux-amd64      # Linux x64
    ├── linux-arm64      # Linux ARM64
    ├── darwin-amd64     # macOS Intel
    ├── darwin-arm64     # macOS M1/M2/M3
    ├── windows-amd64    # Windows x64
    └── windows-arm64    # Windows ARM64
```

## 技术亮点

- **零缓存设计**: 使用 `io.Pipe()` 实现真正的流式转发
- **高性能优化**: 4MB缓冲区 + HTTP Transport优化
- **跨平台支持**: 支持6种主流平台架构
- **简洁架构**: 仅5个核心Go文件，代码清晰易维护
- **完整工具链**: 提供多种客户端和诊断工具

## 许可证

MIT License