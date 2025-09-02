# gt (go-transfer)

高性能流式文件传输工具，零缓存设计，支持超大文件传输。

## 特性

- 🚀 **流式传输** - 零缓存转发，内存占用极低
- 📁 **目录传输** - 自动保留目录结构
- 📊 **实时进度** - 显示速度、进度和剩余时间
- 🔧 **三种模式** - 接收/转发/客户端
- 🎯 **智能日志** - 支持静默/详细/调试模式

## 快速开始

```bash
# 编译
go build -o gt

# 运行（交互式配置）
./gt

# 带日志控制
./gt           # 默认模式
./gt -s        # 静默模式
./gt -v        # 详细模式
./gt --debug   # 调试模式
```

## 三种工作模式

### 1️⃣ Receiver（接收服务器）
接收并存储文件到本地磁盘
```yaml
端口: 17002
存储路径: ~/uploads
```

### 2️⃣ Forward（转发服务器）
零缓存转发到下一跳服务器
```yaml
端口: 17002
目标服务器: http://10.0.0.1:17002
```

### 3️⃣ Client（上传客户端）
上传文件或目录到服务器
```yaml
目标服务器: http://10.0.0.1:17002
```

## 使用示例

### 上传单文件
```bash
./gt
# 选择 3 (client模式)
# 输入文件路径: /path/to/file.zip
```

### 上传整个目录
```bash
./gt
# 选择 3 (client模式)  
# 输入目录路径: /path/to/directory
```
目录结构会完整保留在服务器端。

### 搭建转发链
```
客户端 → 转发器1 → 转发器2 → 接收器
  ↓         ↓         ↓         ↓
Client   Forward   Forward   Receiver
```

## API 接口

| 端点 | 方法 | 描述 |
|-----|------|------|
| `/upload?name=filename` | POST | 上传文件 |
| `/status` | GET | 服务状态 |
| `/docs` | GET | Swagger文档 |

## 其他上传方式

### curl 命令
```bash
curl -X POST "http://server:17002/upload?name=file.txt" \
     --data-binary @file.txt
```

### 浏览器上传
访问 `http://server:17002/docs` 使用 Swagger UI

## 高级特性

- **端口冲突处理**: 自动检测并提示释放
- **进度显示优化**: 统一的进度跟踪系统
- **日志分级**: DEBUG/INFO/WARN/ERROR/SILENT
- **连接优化**: 单连接复用避免端口耗尽

## 构建所有平台

```bash
./build.sh

# 生成文件：
# gt-linux-amd64
# gt-darwin-arm64  (macOS M1/M2/M3)
# gt-windows-amd64.exe
# ...
```

## 配置文件

配置自动保存在 `~/.config/go-transfer/config.yaml`

## 性能指标

- 传输速度: 30MB/s+ (局域网)
- 文件大小: 无限制（测试过100GB+）
- 内存占用: <50MB（零缓存设计）
- 并发连接: 支持多客户端同时上传

## 项目结构

```
├── main.go      # 主程序入口
├── stream.go    # 流式传输核心
├── client.go    # 客户端实现
├── config.go    # 配置管理
├── progress.go  # 统一进度显示
├── logger.go    # 日志系统
└── build.sh     # 多平台构建脚本
```

## License

MIT