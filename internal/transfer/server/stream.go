package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-transfer/internal/constants"
	"go-transfer/internal/infrastructure/logger"
	"go-transfer/internal/infrastructure/progress"
	"go-transfer/internal/infrastructure/system"
	"go-transfer/internal/infrastructure/web"
)

// FileTransfer 文件传输服务
type FileTransfer struct {
	Mode        string
	Port        int
	StoragePath string // receiver模式使用
	TargetURL   string // forward模式使用
}

// Start 启动服务
func (ft *FileTransfer) Start() {
	// 先检查端口是否被占用
	if system.CheckPortInUse(ft.Port) {
		if !system.HandlePortConflict(ft.Port) {
			logger.LogError("无法启动服务，端口 %d 被占用", ft.Port)
			return
		}
	}

	mux := http.NewServeMux()

	// API路由 - 纯流式上传
	mux.HandleFunc("/upload", StreamUploadHandler(ft))
	mux.HandleFunc("/status", ft.handleStatus)

	// Swagger文档路由
	mux.HandleFunc("/swagger.json", web.HandleSwaggerJSON)
	mux.HandleFunc("/swagger/", web.HandleSwaggerUI)
	mux.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})

	addr := fmt.Sprintf("0.0.0.0:%d", ft.Port)

	logger.LogInfo("\n========================================")
	logger.LogInfo("启动 %s 模式服务", ft.Mode)
	logger.LogInfo("监听地址: %s", addr)

	if ft.Mode == "receiver" {
		expandedPath := system.ExpandPath(ft.StoragePath)
		logger.LogInfo("存储路径: %s", expandedPath)
		os.MkdirAll(expandedPath, 0755)
	} else {
		logger.LogInfo("目标服务器: %s", ft.TargetURL)
	}

	logger.LogInfo("📚 API文档: http://%s/docs", addr)
	logger.LogInfo("========================================\n")

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  time.Hour,
		WriteTimeout: time.Hour,
	}

	if err := server.ListenAndServe(); err != nil {
		logger.LogError("服务启动失败: %v", err)
	}
}

// handleStatus 状态检查
func (ft *FileTransfer) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":    "ok",
		"mode":      ft.Mode,
		"port":      ft.Port,
		"timestamp": time.Now().Unix(),
		"version":   "2.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// extractFileName 从请求中提取文件名
func extractFileName(r *http.Request) string {
	if name := r.URL.Query().Get("name"); name != "" {
		return name
	}
	return fmt.Sprintf("upload_%d.bin", time.Now().Unix())
}

// StreamUploadHandler 纯流式上传处理器（支持二进制流和FormData）
func StreamUploadHandler(ft *FileTransfer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "仅支持POST方法", http.StatusMethodNotAllowed)
			return
		}

		contentType := r.Header.Get("Content-Type")
		
		// 如果是multipart/form-data（浏览器文件上传）
		if strings.HasPrefix(contentType, "multipart/form-data") {
			handleMultipartUpload(ft, w, r)
			return
		}

		// 否则按二进制流处理（curl --data-binary）
		handleBinaryUpload(ft, w, r)
	}
}

// handleMultipartUpload 处理FormData上传（浏览器友好）
func handleMultipartUpload(ft *FileTransfer, w http.ResponseWriter, r *http.Request) {
	// 解析multipart表单，限制32MB内存
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, fmt.Sprintf("解析表单失败: %v", err), http.StatusBadRequest)
		return
	}
	defer r.MultipartForm.RemoveAll()

	// 获取文件
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, fmt.Sprintf("获取文件失败: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 获取文件名（优先使用URL参数，否则使用上传的文件名）
	fileName := extractFileName(r)
	if fileName == fmt.Sprintf("upload_%d.bin", time.Now().Unix()) {
		fileName = header.Filename
	}

	// 根据模式处理
	switch ft.Mode {
	case "receiver":
		handleReceive(ft, w, file, fileName, header.Size, true)
	case "forward":
		handleForward(ft, w, file, fileName, header.Size, true)
	default:
		http.Error(w, "未知服务模式", http.StatusInternalServerError)
	}
}

// handleBinaryUpload 处理二进制流上传（命令行友好）
func handleBinaryUpload(ft *FileTransfer, w http.ResponseWriter, r *http.Request) {
	fileName := extractFileName(r)

	switch ft.Mode {
	case "receiver":
		handleReceive(ft, w, r.Body, fileName, r.ContentLength, false)
	case "forward":
		handleForward(ft, w, r.Body, fileName, r.ContentLength, false)
	default:
		http.Error(w, "未知服务模式", http.StatusInternalServerError)
	}
}

// handleReceive 统一的接收处理函数
func handleReceive(ft *FileTransfer, w http.ResponseWriter, reader io.Reader, fileName string, size int64, isFormData bool) {
	expandedPath := system.ExpandPath(ft.StoragePath)

	// 处理带路径的文件名
	systemFileName := filepath.FromSlash(fileName)
	finalPath := filepath.Join(expandedPath, systemFileName)

	// 如果文件名包含路径，创建目录
	finalDir := filepath.Dir(finalPath)
	if finalDir != expandedPath {
		if err := os.MkdirAll(finalDir, constants.DirPermission); err != nil {
			http.Error(w, fmt.Sprintf("创建目录失败: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// 立即显示开始接收文件
	sourceType := ""
	if isFormData {
		sourceType = " [FormData]"
	}
	
	if size > 0 {
		sizeMB := float64(size) / 1024 / 1024
		logger.LogInfo("⬇️  开始接收: %s (%.2f MB)%s", fileName, sizeMB, sourceType)
	} else {
		logger.LogInfo("⬇️  开始接收: %s%s", fileName, sourceType)
	}

	// 检查文件是否已存在
	if _, err := os.Stat(finalPath); err == nil {
		logger.LogWarn("文件已存在，将被覆盖: %s", fileName)
	}

	// 创建目标文件（如果存在则覆盖）
	outFile, err := os.Create(finalPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("创建文件失败: %v", err), http.StatusInternalServerError)
		return
	}
	defer outFile.Close()

	// 创建进度跟踪的Writer
	progressWriter := progress.NewProgressWriter(outFile, size, "接收进度")

	// 流式复制 - 带进度跟踪
	written, err := io.Copy(progressWriter, reader)
	if err != nil {
		os.Remove(finalPath)
		http.Error(w, fmt.Sprintf("写入文件失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 完成进度条显示
	progressWriter.PrintProgress()
	fmt.Println() // 换行

	// 计算传输速度
	speed := progressWriter.GetSpeed()
	speedMB := speed / 1024 / 1024
	writtenMB := float64(written) / 1024 / 1024
	
	logger.LogSuccess("文件已保存: %s (%.2f MB, %.2f MB/s)", fileName, writtenMB, speedMB)
	fmt.Fprintf(w, "文件上传成功: %s (%d bytes)", fileName, written)
}

// handleForward 统一的转发处理函数
func handleForward(ft *FileTransfer, w http.ResponseWriter, reader io.Reader, fileName string, size int64, isFormData bool) {
	targetURL := ft.TargetURL

	// 立即显示开始转发
	sourceType := ""
	if isFormData {
		sourceType = " [FormData]"
	}
	
	if size > 0 {
		sizeMB := float64(size) / 1024 / 1024
		logger.LogInfo("🔄 开始转发: %s (%.2f MB) → %s%s", fileName, sizeMB, targetURL, sourceType)
	} else {
		logger.LogInfo("🔄 开始转发: %s → %s%s", fileName, targetURL, sourceType)
	}

	startTime := time.Now()

	// 创建管道，实现零缓存流式转发
	pipeReader, pipeWriter := io.Pipe()
	errChan := make(chan error, 2)
	transferredBytes := int64(0)

	// 协程1: 从客户端读取，写入管道（带进度跟踪）
	go func() {
		defer pipeWriter.Close()

		// 创建进度跟踪的Writer
		progressPipe := progress.NewProgressWriter(pipeWriter, size, "上传进度")
		defer func() {
			current, _, _ := progressPipe.GetProgress()
			transferredBytes = current
		}()

		// 选择合适的缓冲区大小
		bufferSize := constants.SmallBufferSize  // 256KB for streaming
		if isFormData {
			bufferSize = constants.LargeBufferSize  // 4MB for FormData
		}
		
		buffer := make([]byte, bufferSize)
		_, err := io.CopyBuffer(progressPipe, reader, buffer)
		errChan <- err
	}()

	// 协程2: 从管道读取，转发到目标服务器
	go func() {
		// 创建转发请求
		req, err := http.NewRequest("POST", targetURL+"/upload?name="+fileName, pipeReader)
		if err != nil {
			errChan <- fmt.Errorf("创建转发请求失败: %v", err)
			return
		}

		// 设置请求头
		if size > 0 {
			req.ContentLength = size
			req.Header.Set("Content-Length", fmt.Sprintf("%d", size))
		}
		req.Header.Set("X-File-Name", fileName)
		req.Header.Set("Content-Type", "application/octet-stream")

		// 使用统一的HTTP客户端
		client := web.CreateForwardClient()
		resp, err := client.Do(req)
		if err != nil {
			errChan <- fmt.Errorf("转发失败: %v", err)
			return
		}
		defer resp.Body.Close()

		// 将目标服务器的响应流式返回给客户端
		w.WriteHeader(resp.StatusCode)
		buffer := make([]byte, constants.LargeBufferSize)
		io.CopyBuffer(w, resp.Body, buffer)
		errChan <- nil
	}()

	// 等待两个协程完成
	err1 := <-errChan
	err2 := <-errChan

	// 换行结束进度条
	fmt.Println()

	duration := time.Since(startTime)
	speed := float64(transferredBytes) / duration.Seconds() / 1024 / 1024

	if err1 != nil {
		logger.LogError("转发失败: %v", err1)
		if err2 == nil {
			http.Error(w, err1.Error(), http.StatusBadGateway)
		}
	} else if err2 != nil {
		logger.LogError("转发失败: %v", err2)
	} else {
		transferredMB := float64(transferredBytes) / 1024 / 1024
		logger.LogSuccess("成功转发: %s (%.2f MB, %.2f MB/s, 耗时 %.1fs)",
			fileName, transferredMB, speed, duration.Seconds())
	}
}

