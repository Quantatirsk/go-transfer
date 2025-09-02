# go-transfer

高性能流式文件传输工具，支持服务器和客户端模式。

## 特性

- **三种模式** - receiver（接收）、forward（转发）、client（客户端）
- **统一配置** - 一个命令，交互式选择模式
- **目录传输** - 自动保留目录结构，无需打包
- **流式传输** - 零缓存转发，支持超大文件
- **实时进度** - 显示传输进度、速度和剩余时间
- **极简配置** - 配置自动保存，下次直接使用

## 快速开始

### 下载或编译

```bash
# 从源码编译
go build -o go-transfer

# 或下载预编译版本
# Linux/macOS/Windows 版本见 releases
```

### 运行

```bash
./go-transfer
# 选择模式: 1-receiver, 2-forward, 3-client
```

## 三种模式

### 1. Receiver（接收服务器）
接收并存储文件
- 配置：端口、存储路径

### 2. Forward（中继服务器）
转发文件到下一跳，不占用磁盘
- 配置：端口、目标服务器

### 3. Client（客户端）  
上传文件或目录到服务器
- 配置：目标服务器（保存）
- 每次输入：文件/目录路径

## 目录传输特性

### 单文件上传
- 只传输文件名，不包含路径
- 例：`/home/user/doc.pdf` → `doc.pdf`

### 目录上传
- 保留完整目录结构
- 自动在接收端创建子目录
- 例：上传 `myproject/` 目录
  ```
  myproject/
  ├── src/main.go
  └── README.md
  
  接收端：
  uploads/myproject/src/main.go
  uploads/myproject/README.md
  ```

## 其他上传方式

### 使用 Go 客户端（推荐）
```bash
./go-transfer
# 选择 client 模式
```

### Python 客户端
```bash
python3 transfer.py file.zip http://server:5000
```

### 直接 curl
```bash
curl -X POST "http://server:5000/upload?name=file.txt" \
     --data-binary @file.txt
```

## 典型部署

```
Client → Forward(:8080) → Forward(:17002) → Receiver(:17002)
客户端     入口服务器        中继服务器        存储服务器
```

配置保存在 `~/.config/go-transfer/config.yaml`

## 技术特点

### 零缓存转发
中继服务器使用 `io.Pipe()` 实现流式转发，数据不落盘

### 实时进度
```
上传进度: [████████░░░░] 52.3% (57.36 MB/109.61 MB) 速度: 12.45 MB/s 剩余: 4秒
```

## API端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/upload?name=filename` | POST | 流式上传文件 |
| `/status` | GET | 服务状态 |
| `/docs` | GET | API文档 |

## 性能特性

- **高速传输**: 30MB/s+，4MB缓冲区优化
- **超大文件**: 支持10GB、100GB文件
- **端口冲突处理**: 自动检测并释放
- **文件覆盖**: 同名自动覆盖

## 编译的二进制

- Linux AMD64/ARM64
- macOS Intel/M1/M2/M3
- Windows x64/ARM64

## 项目文件

- `main.go` - 主程序入口
- `config.go` - 配置管理
- `client.go` - 客户端实现
- `stream.go` - 流式传输
- `build.sh` - 构建脚本

## 许可证

MIT License