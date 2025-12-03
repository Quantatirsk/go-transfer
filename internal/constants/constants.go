package constants

import "time"

// LogLevel 日志级别
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	SILENT // 静默模式，不输出任何日志
)

const (
	// UI 显示相关
	SeparatorLine     = "========================================"
	ProgressBarLength = 40
	ProgressLineWidth = 120

	// 网络相关
	DefaultPort     = 17002
	DefaultTimeout  = 30 * time.Minute
	ReadTimeout     = time.Hour
	WriteTimeout    = time.Hour
	IdleConnTimeout = 300 * time.Second
	ResponseTimeout = 60 * time.Second

	// 缓冲区大小
	SmallBufferSize  = 256 * 1024      // 256KB - 用于流式传输
	MediumBufferSize = 512 * 1024      // 512KB - 用于HTTP传输
	LargeBufferSize  = 4 * 1024 * 1024 // 4MB - 用于本地文件操作

	// HTTP 客户端配置
	MaxIdleConns        = 1
	MaxIdleConnsPerHost = 1
	MaxConnsPerHost     = 1

	// 重试相关
	MaxRetries      = 3
	PortExhaustWait = 5 * time.Second

	// 进度更新
	ProgressUpdateInterval = 100 * time.Millisecond

	// 文件权限
	DirPermission  = 0755
	FilePermission = 0644

	// 默认路径
	DefaultStoragePath = "~/uploads"
	DefaultConfigDir   = ".config/go-transfer"
	ConfigFileName     = "config.yaml"
)
