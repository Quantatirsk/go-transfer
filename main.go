package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// FileTransfer 文件传输服务
type FileTransfer struct {
	mode        string
	port        int
	storagePath string // receiver模式使用
	targetURL   string // relay/gateway模式使用
}

// Start 启动服务
func (ft *FileTransfer) Start() {
	// 先检查端口是否被占用
	if checkPortInUse(ft.port) {
		if !handlePortConflict(ft.port) {
			log.Fatalf("无法启动服务，端口 %d 被占用", ft.port)
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
	
	log.Printf("\n========================================")
	log.Printf("启动 %s 模式服务", ft.mode)
	log.Printf("监听地址: %s", addr)
	log.Printf("纯流式上传，零缓存，支持超大文件")
	
	if ft.mode == "receiver" {
		expandedPath := expandPath(ft.storagePath)
		log.Printf("存储路径: %s", expandedPath)
		os.MkdirAll(expandedPath, 0755)
	} else {
		log.Printf("目标服务器: %s", ft.targetURL)
	}
	
	log.Printf("📚 API文档: http://%s/docs", addr)
	log.Printf("========================================\n")
	
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  time.Hour,
		WriteTimeout: time.Hour,
	}
	
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("服务启动失败: %v", err)
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
	// 创建配置管理器
	cm := NewConfigManager()
	
	// 运行交互式配置
	config, err := cm.LoadOrCreateConfig()
	if err != nil {
		log.Fatalf("配置错误: %v", err)
	}
	
	// 创建并启动服务
	ft := &FileTransfer{
		mode:        config.Mode,
		port:        config.Port,
		storagePath: config.StoragePath,
		targetURL:   config.TargetURL,
	}
	
	ft.Start()
}