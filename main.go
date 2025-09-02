package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

// FileTransfer 文件传输服务
type FileTransfer struct {
	mode        string
	port        int
	storagePath string // receiver模式使用
	targetURL   string // forward模式使用
}

// Start 启动服务
func (ft *FileTransfer) Start() {
	// 先检查端口是否被占用
	if checkPortInUse(ft.port) {
		if !handlePortConflict(ft.port) {
			LogError("无法启动服务，端口 %d 被占用", ft.port)
			os.Exit(1)
		}
	}

	mux := http.NewServeMux()

	// API路由 - 纯流式上传
	mux.HandleFunc("/upload", StreamUploadHandler(ft))
	mux.HandleFunc("/status", ft.handleStatus)

	// Swagger文档路由
	mux.HandleFunc("/swagger.json", handleSwaggerJSON)
	mux.HandleFunc("/swagger/", handleSwaggerUI)
	mux.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})

	addr := fmt.Sprintf("0.0.0.0:%d", ft.port)

	LogInfo("\n========================================")
	LogInfo("启动 %s 模式服务", ft.mode)
	LogInfo("监听地址: %s", addr)

	if ft.mode == "receiver" {
		expandedPath := expandPath(ft.storagePath)
		LogInfo("存储路径: %s", expandedPath)
		os.MkdirAll(expandedPath, 0755)
	} else {
		LogInfo("目标服务器: %s", ft.targetURL)
	}

	LogInfo("📚 API文档: http://%s/docs", addr)
	LogInfo("========================================\n")

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  time.Hour,
		WriteTimeout: time.Hour,
	}

	if err := server.ListenAndServe(); err != nil {
		LogError("服务启动失败: %v", err)
		os.Exit(1)
	}
}

// handleStatus 状态检查
func (ft *FileTransfer) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":    "ok",
		"mode":      ft.mode,
		"port":      ft.port,
		"timestamp": time.Now().Unix(),
		"version":   "2.0.0", // 简化版
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func main() {
	// 命令行参数
	var verbose bool
	var silent bool
	var debug bool
	
	flag.BoolVar(&verbose, "v", false, "详细模式，显示更多日志信息")
	flag.BoolVar(&verbose, "verbose", false, "详细模式，显示更多日志信息")
	flag.BoolVar(&silent, "s", false, "静默模式，仅显示必要信息")
	flag.BoolVar(&silent, "silent", false, "静默模式，仅显示必要信息")
	flag.BoolVar(&debug, "debug", false, "调试模式，显示所有调试信息")
	flag.Parse()
	
	// 设置日志级别
	if debug {
		GlobalLogger.SetLevel(DEBUG)
	} else if verbose {
		GlobalLogger.SetVerbose(true)
	} else if silent {
		GlobalLogger.SetSilent(true)
	}
	
	// 创建配置管理器
	cm := NewConfigManager()

	// 运行交互式配置
	config, err := cm.LoadOrCreateConfig()
	if err != nil {
		LogError("配置错误: %v", err)
		os.Exit(1)
	}

	// 根据配置的模式执行相应功能
	switch config.Mode {
	case "client":
		// 客户端模式 - 上传文件
		runConfiguredClient(config)

	case "receiver", "forward":
		// 服务器模式 - 启动服务
		ft := &FileTransfer{
			mode:        config.Mode,
			port:        config.Port,
			storagePath: config.StoragePath,
			targetURL:   config.TargetURL,
		}
		ft.Start()

	default:
		LogError("未知模式: %s", config.Mode)
		os.Exit(1)
	}
}
