package main

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)


// extractFileName 从请求中提取文件名
func extractFileName(r *http.Request) string {
	// 从URL参数获取文件名
	if name := r.URL.Query().Get("name"); name != "" {
		return name
	}

	// 默认值：使用时间戳
	return fmt.Sprintf("upload_%d.bin", time.Now().Unix())
}

// StreamUploadHandler 纯流式上传处理器（支持二进制流和FormData）
func StreamUploadHandler(ft *FileTransfer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "仅支持POST方法", http.StatusMethodNotAllowed)
			return
		}


		// 检查Content-Type
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
	switch ft.mode {
	case "receiver":
		handleReceiveFile(ft, w, file, fileName, header.Size)
	case "forward":
		handleForwardFile(ft, w, file, fileName, header.Size)
	default:
		http.Error(w, "未知服务模式", http.StatusInternalServerError)
	}
}

// handleBinaryUpload 处理二进制流上传（命令行友好）
func handleBinaryUpload(ft *FileTransfer, w http.ResponseWriter, r *http.Request) {
	fileName := extractFileName(r)

	switch ft.mode {
	case "receiver":
		handleStreamReceive(ft, w, r, fileName)
	case "forward":
		handleStreamForward(ft, w, r, fileName)
	default:
		http.Error(w, "未知服务模式", http.StatusInternalServerError)
	}
}

// handleReceiveFile 接收并保存文件（FormData）
func handleReceiveFile(ft *FileTransfer, w http.ResponseWriter, file multipart.File, fileName string, size int64) {
	expandedPath := expandPath(ft.storagePath)
	
	// 处理带路径的文件名
	// 将斜杠路径分隔符转换为系统路径分隔符
	systemFileName := filepath.FromSlash(fileName)
	finalPath := filepath.Join(expandedPath, systemFileName)
	
	// 如果文件名包含路径，创建目录
	finalDir := filepath.Dir(finalPath)
	if finalDir != expandedPath {
		if err := os.MkdirAll(finalDir, 0755); err != nil {
			http.Error(w, fmt.Sprintf("创建目录失败: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// 立即显示开始接收文件
	if size > 0 {
		sizeMB := float64(size) / 1024 / 1024
		log.Printf("⬇️  开始接收: %s (预计 %.2f MB) [FormData]", fileName, sizeMB)
	} else {
		log.Printf("⬇️  开始接收: %s [FormData]", fileName)
	}

	// 检查文件是否已存在
	if _, err := os.Stat(finalPath); err == nil {
		log.Printf("⚠️  文件已存在，将被覆盖: %s", fileName)
	}

	// 创建目标文件（如果存在则覆盖）
	outFile, err := os.Create(finalPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("创建文件失败: %v", err), http.StatusInternalServerError)
		return
	}
	defer outFile.Close()

	// 创建进度跟踪的Writer
	progressWriter := &ProgressWriter{
		Writer:    outFile,
		Total:     size,
		FileName:  fileName,
		StartTime: time.Now(),
	}

	// 流式复制 - 带进度跟踪
	written, err := io.Copy(progressWriter, file)
	if err != nil {
		os.Remove(finalPath)
		http.Error(w, fmt.Sprintf("写入文件失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 完成进度条显示
	progressWriter.printProgress()
	fmt.Println() // 换行
	
	// 计算传输时间
	duration := time.Since(progressWriter.StartTime)
	speed := float64(written) / duration.Seconds() / 1024 / 1024

	writtenMB := float64(written) / 1024 / 1024
	log.Printf("✅ 文件已保存: %s (%.2f MB, %.2f MB/s, 耗时 %.1fs)",
		fileName, writtenMB, speed, duration.Seconds())
	fmt.Fprintf(w, "文件上传成功: %s (%d bytes)", fileName, written)
}

// handleForwardFile 转发文件（FormData）
func handleForwardFile(ft *FileTransfer, w http.ResponseWriter, file multipart.File, fileName string, size int64) {
	targetURL := ft.targetURL

	// 立即显示开始转发
	if size > 0 {
		sizeMB := float64(size) / 1024 / 1024
		log.Printf("🔄 开始转发: %s (预计 %.2f MB) → %s [FormData]", fileName, sizeMB, targetURL)
	} else {
		log.Printf("🔄 开始转发: %s → %s [FormData]", fileName, targetURL)
	}

	startTime := time.Now()
	transferredBytes := int64(0)

	// 创建管道
	pipeReader, pipeWriter := io.Pipe()
	errChan := make(chan error, 1)

	// 协程1: 从文件读取到管道（带进度跟踪，使用大缓冲区）
	go func() {
		defer pipeWriter.Close()

		// 创建进度跟踪的Writer
		progressPipe := &ProgressPipeWriter{
			Writer:      pipeWriter,
			Total:       size,
			FileName:    fileName,
			StartTime:   startTime,
			Transferred: &transferredBytes,
			LogPrefix:   "上传",
		}

		// 使用 4MB 缓冲区提高传输效率
		buffer := make([]byte, 4*1024*1024)
		_, err := io.CopyBuffer(progressPipe, file, buffer)
		errChan <- err
	}()

	// 协程2: 从管道转发到目标服务器
	go func() {
		req, err := http.NewRequest("POST", targetURL+"/upload?name="+fileName, pipeReader)
		if err != nil {
			errChan <- fmt.Errorf("创建转发请求失败: %v", err)
			return
		}

		if size > 0 {
			req.ContentLength = size
			req.Header.Set("Content-Length", fmt.Sprintf("%d", size))
		}
		req.Header.Set("Content-Type", "application/octet-stream")

		// 优化的 HTTP 客户端配置
		client := &http.Client{
			Timeout: 2 * time.Hour, // 2小时超时
			Transport: &http.Transport{
				DisableCompression: true,
				DisableKeepAlives:  false,
				IdleConnTimeout:    90 * time.Second,
				WriteBufferSize:    4 * 1024 * 1024,
				ReadBufferSize:     4 * 1024 * 1024,
				MaxIdleConns:       10,
				MaxConnsPerHost:    10,
			},
		}

		resp, err := client.Do(req)
		if err != nil {
			errChan <- fmt.Errorf("转发失败: %v", err)
			return
		}
		defer resp.Body.Close()

		w.WriteHeader(resp.StatusCode)
		// 使用缓冲复制响应
		buffer := make([]byte, 4*1024*1024)
		io.CopyBuffer(w, resp.Body, buffer)
		errChan <- nil
	}()

	// 等待完成
	err1 := <-errChan
	err2 := <-errChan

	// 换行结束进度条
	fmt.Println()
	
	duration := time.Since(startTime)
	speed := float64(transferredBytes) / duration.Seconds() / 1024 / 1024

	if err1 != nil {
		log.Printf("❌ 转发失败: %v", err1)
		if err2 == nil {
			http.Error(w, err1.Error(), http.StatusBadGateway)
		}
	} else if err2 != nil {
		log.Printf("❌ 转发失败: %v", err2)
	} else {
		transferredMB := float64(transferredBytes) / 1024 / 1024
		log.Printf("✅ 成功转发: %s (%.2f MB, %.2f MB/s, 耗时 %.1fs)",
			fileName, transferredMB, speed, duration.Seconds())
	}
}

// handleStreamReceive 流式接收（二进制流）
func handleStreamReceive(ft *FileTransfer, w http.ResponseWriter, r *http.Request, fileName string) {
	expandedPath := expandPath(ft.storagePath)
	
	// 处理带路径的文件名
	// 将斜杠路径分隔符转换为系统路径分隔符
	systemFileName := filepath.FromSlash(fileName)
	finalPath := filepath.Join(expandedPath, systemFileName)
	
	// 如果文件名包含路径，创建目录
	finalDir := filepath.Dir(finalPath)
	if finalDir != expandedPath {
		if err := os.MkdirAll(finalDir, 0755); err != nil {
			http.Error(w, fmt.Sprintf("创建目录失败: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("📁 创建目录: %s", filepath.Dir(systemFileName))
	}

	// 立即显示开始接收文件
	contentLength := r.ContentLength
	if contentLength > 0 {
		sizeMB := float64(contentLength) / 1024 / 1024
		log.Printf("⬇️  开始接收: %s (预计 %.2f MB)", fileName, sizeMB)
	} else {
		log.Printf("⬇️  开始接收: %s", fileName)
	}

	// 检查文件是否已存在
	if _, err := os.Stat(finalPath); err == nil {
		log.Printf("⚠️  文件已存在，将被覆盖: %s", fileName)
	}

	// 创建目标文件（如果存在则覆盖）
	outFile, err := os.Create(finalPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("创建文件失败: %v", err), http.StatusInternalServerError)
		return
	}
	defer outFile.Close()

	// 创建进度跟踪的Writer
	progressWriter := &ProgressWriter{
		Writer:    outFile,
		Total:     contentLength,
		FileName:  fileName,
		StartTime: time.Now(),
	}

	// 流式复制 - 带进度跟踪
	written, err := io.Copy(progressWriter, r.Body)
	if err != nil {
		os.Remove(finalPath) // 失败时清理
		http.Error(w, fmt.Sprintf("写入文件失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 完成进度条显示
	progressWriter.printProgress()
	fmt.Println() // 换行
	
	// 计算传输时间
	duration := time.Since(progressWriter.StartTime)
	speed := float64(written) / duration.Seconds() / 1024 / 1024

	writtenMB := float64(written) / 1024 / 1024
	log.Printf("✅ 文件已保存: %s (%.2f MB, %.2f MB/s, 耗时 %.1fs)",
		fileName, writtenMB, speed, duration.Seconds())
	fmt.Fprintf(w, "文件上传成功: %s (%d bytes)", fileName, written)
}

// handleStreamForward 流式转发（二进制流，零缓存）
func handleStreamForward(ft *FileTransfer, w http.ResponseWriter, r *http.Request, fileName string) {
	targetURL := ft.targetURL

	// 获取Content-Length用于转发
	contentLength := r.ContentLength

	// 立即显示开始转发
	if contentLength > 0 {
		sizeMB := float64(contentLength) / 1024 / 1024
		log.Printf("🔄 开始转发: %s (预计 %.2f MB) → %s", fileName, sizeMB, targetURL)
	} else {
		log.Printf("🔄 开始转发: %s → %s", fileName, targetURL)
	}

	startTime := time.Now()

	// 创建管道，实现零缓存流式转发
	pipeReader, pipeWriter := io.Pipe()

	// 错误通道
	errChan := make(chan error, 1)
	transferredBytes := int64(0)

	// 协程1: 从客户端读取，写入管道（带进度跟踪，使用自适应缓冲区）
	go func() {
		defer pipeWriter.Close()

		// 创建进度跟踪的Writer
		progressPipe := &ProgressPipeWriter{
			Writer:      pipeWriter,
			Total:       contentLength,
			FileName:    fileName,
			StartTime:   startTime,
			Transferred: &transferredBytes,
			LogPrefix:   "上传",
		}

		// 使用较小的缓冲区（256KB），避免过度缓冲导致的背压问题
		// 较小的缓冲区能更快地感知下游压力，实现更平滑的流控
		buffer := make([]byte, 256*1024)
		_, err := io.CopyBuffer(progressPipe, r.Body, buffer)
		if err != nil {
			errChan <- fmt.Errorf("读取上传数据失败: %v", err)
			return
		}
		errChan <- nil
	}()

	// 协程2: 从管道读取，转发到目标服务器
	go func() {
		// 创建转发请求
		req, err := http.NewRequest("POST", targetURL+"/upload?name="+fileName, pipeReader)
		if err != nil {
			errChan <- fmt.Errorf("创建转发请求失败: %v", err)
			return
		}

		// 复制原始请求的相关header
		if contentLength > 0 {
			req.ContentLength = contentLength
			req.Header.Set("Content-Length", fmt.Sprintf("%d", contentLength))
		}
		req.Header.Set("X-File-Name", fileName)
		req.Header.Set("Content-Type", "application/octet-stream")

		// 优化的 HTTP 客户端配置，平衡速度和背压
		client := &http.Client{
			Timeout: 2 * time.Hour, // 2小时超时，支持超大文件
			Transport: &http.Transport{
				// 禁用请求体缓冲，实现真正的流式传输
				DisableCompression: true,
				DisableKeepAlives:  false,
				// 增加空闲连接超时，支持长时间传输
				IdleConnTimeout: 90 * time.Second,
				// 使用适中的缓冲区（512KB），避免过度缓冲
				// 过大的缓冲区会延迟背压信号的传递
				WriteBufferSize: 512 * 1024,
				ReadBufferSize:  512 * 1024,
				// 保持适量连接
				MaxIdleConns:    10,
				MaxConnsPerHost: 10,
			},
		}

		resp, err := client.Do(req)
		if err != nil {
			errChan <- fmt.Errorf("转发失败: %v", err)
			return
		}
		defer resp.Body.Close()

		// 将目标服务器的响应流式返回给客户端（使用缓冲）
		w.WriteHeader(resp.StatusCode)
		buffer := make([]byte, 4*1024*1024)
		io.CopyBuffer(w, resp.Body, buffer)

		errChan <- nil
	}()

	// 等待两个协程完成
	err1 := <-errChan
	err2 := <-errChan

	duration := time.Since(startTime)
	speed := float64(transferredBytes) / duration.Seconds() / 1024 / 1024

	if err1 != nil {
		log.Printf("❌ 转发失败: %v", err1)
		if err2 == nil {
			// 如果只有一个错误，返回错误信息
			http.Error(w, err1.Error(), http.StatusBadGateway)
		}
	} else if err2 != nil {
		log.Printf("❌ 转发失败: %v", err2)
		http.Error(w, err2.Error(), http.StatusBadGateway)
	} else {
		transferredMB := float64(transferredBytes) / 1024 / 1024
		log.Printf("✅ 成功转发: %s (%.2f MB, %.2f MB/s, 耗时 %.1fs)",
			fileName, transferredMB, speed, duration.Seconds())
	}
}

// StreamForwardWithProgress 带进度的流式转发（可选）
func StreamForwardWithProgress(ft *FileTransfer, w http.ResponseWriter, r *http.Request, fileName string) {
	targetURL := ft.targetURL

	// 创建进度跟踪的Reader
	progressReader := &ProgressReader{
		Reader:   r.Body,
		Total:    r.ContentLength,
		FileName: fileName,
	}

	// 创建转发请求
	req, err := http.NewRequest("POST", targetURL+"/upload?name="+fileName, progressReader)
	if err != nil {
		http.Error(w, "创建请求失败", http.StatusInternalServerError)
		return
	}

	req.ContentLength = r.ContentLength
	req.Header.Set("X-File-Name", fileName)

	// 使用自定义Transport实现零缓存
	client := &http.Client{
		Transport: &http.Transport{
			DisableCompression: true,
			// 禁用请求缓冲
			WriteBufferSize: 0,
			ReadBufferSize:  0,
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("转发失败: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 流式返回响应
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// ProgressReader 带进度跟踪的Reader
type ProgressReader struct {
	Reader   io.Reader
	Total    int64
	Current  int64
	FileName string
	LastLog  time.Time
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Current += int64(n)

	// 每秒打印一次进度
	now := time.Now()
	if now.Sub(pr.LastLog) > time.Second && pr.Total > 0 {
		percentage := float64(pr.Current) / float64(pr.Total) * 100
		log.Printf("转发进度 %s: %.1f%% (%d/%d bytes)",
			pr.FileName, percentage, pr.Current, pr.Total)
		pr.LastLog = now
	}

	return n, err
}

// ProgressWriter 带进度跟踪的Writer（用于接收文件）
type ProgressWriter struct {
	Writer    io.Writer
	Total     int64
	Current   int64
	FileName  string
	LastPrint time.Time
	StartTime time.Time
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.Writer.Write(p)
	pw.Current += int64(n)

	// 每100ms更新一次进度（和客户端保持一致）
	now := time.Now()
	if now.Sub(pw.LastPrint) >= 100*time.Millisecond || err == io.EOF {
		pw.printProgress()
		pw.LastPrint = now
	}

	return n, err
}

func (pw *ProgressWriter) printProgress() {
	if pw.Total == 0 && pw.Current == 0 {
		return
	}

	percentage := float64(0)
	if pw.Total > 0 {
		percentage = float64(pw.Current) * 100 / float64(pw.Total)
	}
	
	elapsed := time.Since(pw.StartTime).Seconds()
	speed := float64(0)
	eta := float64(0)
	
	if elapsed > 0 {
		speed = float64(pw.Current) / elapsed
		if speed > 0 && pw.Total > 0 {
			eta = float64(pw.Total-pw.Current) / speed
		}
	}

	// 构建进度条（和客户端保持一致的风格）
	barLength := 40
	filled := 0
	if pw.Total > 0 {
		filled = int(float64(barLength) * float64(pw.Current) / float64(pw.Total))
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barLength-filled)

	// 构建固定长度的输出字符串，避免残影
	const lineWidth = 120 // 固定行宽
	var output string
	
	if pw.Total > 0 {
		// 格式化各个部分，确保固定宽度
		percentStr := fmt.Sprintf("%5.1f%%", percentage) // 固定5字符宽
		sizeStr := fmt.Sprintf("%s/%s", formatSize(pw.Current), formatSize(pw.Total))
		speedStr := fmt.Sprintf("%s/s", formatSize(int64(speed)))
		
		output = fmt.Sprintf("接收进度: [%s] %s %-20s 速度: %-12s",
			bar, percentStr, sizeStr, speedStr)
		
		if eta > 0 && pw.Current < pw.Total {
			etaStr := fmt.Sprintf("剩余: %d秒", int(eta))
			output = fmt.Sprintf("%s %-15s", output, etaStr)
		}
	} else {
		// 未知大小时的进度显示
		sizeStr := formatSize(pw.Current)
		speedStr := fmt.Sprintf("%s/s", formatSize(int64(speed)))
		output = fmt.Sprintf("接收进度: %-15s 速度: %-12s", sizeStr, speedStr)
	}
	
	// 使用固定宽度输出，多余部分用空格填充，避免残影
	fmt.Printf("\r%-*s", lineWidth, output)
}

// ProgressPipeWriter 带进度跟踪的管道Writer（用于转发）
type ProgressPipeWriter struct {
	Writer      io.Writer
	Total       int64
	FileName    string
	LastPrint   time.Time
	StartTime   time.Time
	Transferred *int64
	LogPrefix   string
}

func (ppw *ProgressPipeWriter) Write(p []byte) (int, error) {
	n, err := ppw.Writer.Write(p)
	*ppw.Transferred += int64(n)

	// 每100ms更新一次进度（和客户端保持一致）
	now := time.Now()
	if now.Sub(ppw.LastPrint) >= 100*time.Millisecond || err == io.EOF {
		ppw.printProgress()
		ppw.LastPrint = now
	}

	return n, err
}

func (ppw *ProgressPipeWriter) printProgress() {
	current := *ppw.Transferred
	if current == 0 {
		return
	}

	percentage := float64(0)
	if ppw.Total > 0 {
		percentage = float64(current) * 100 / float64(ppw.Total)
	}
	
	elapsed := time.Since(ppw.StartTime).Seconds()
	speed := float64(0)
	eta := float64(0)
	
	if elapsed > 0 {
		speed = float64(current) / elapsed
		if speed > 0 && ppw.Total > 0 {
			eta = float64(ppw.Total-current) / speed
		}
	}

	// 构建进度条
	barLength := 40
	filled := 0
	if ppw.Total > 0 {
		filled = int(float64(barLength) * float64(current) / float64(ppw.Total))
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barLength-filled)

	// 构建固定长度的输出字符串，避免残影
	const lineWidth = 120 // 固定行宽
	var output string
	
	if ppw.Total > 0 {
		// 格式化各个部分，确保固定宽度
		percentStr := fmt.Sprintf("%5.1f%%", percentage) // 固定5字符宽
		sizeStr := fmt.Sprintf("%s/%s", formatSize(current), formatSize(ppw.Total))
		speedStr := fmt.Sprintf("%s/s", formatSize(int64(speed)))
		
		output = fmt.Sprintf("%s进度: [%s] %s %-20s 速度: %-12s",
			ppw.LogPrefix, bar, percentStr, sizeStr, speedStr)
		
		if eta > 0 && current < ppw.Total {
			etaStr := fmt.Sprintf("剩余: %d秒", int(eta))
			output = fmt.Sprintf("%s %-15s", output, etaStr)
		}
	} else {
		// 未知大小时的进度显示
		sizeStr := formatSize(current)
		speedStr := fmt.Sprintf("%s/s", formatSize(int64(speed)))
		output = fmt.Sprintf("%s进度: %-15s 速度: %-12s", 
			ppw.LogPrefix, sizeStr, speedStr)
	}
	
	// 使用固定宽度输出，多余部分用空格填充，避免残影
	fmt.Printf("\r%-*s", lineWidth, output)
}

/*
使用示例：

1. 浏览器/Swagger UI上传（FormData）:
   - 在 /docs 页面可以直接选择文件上传
   - 支持拖拽文件

2. 命令行二进制流上传:
   curl -X POST http://localhost:17002/upload?name=myfile.zip \
        --data-binary @myfile.zip

3. 命令行FormData上传:
   curl -X POST http://localhost:17002/upload \
        -F "file=@myfile.zip"

优势：
- 兼容性好：同时支持浏览器和命令行
- 零缓存：数据直接流式传输，不占用中继服务器空间
- 支持超大文件：10GB、100GB都没问题
- 简单可靠：一行命令搞定
*/
