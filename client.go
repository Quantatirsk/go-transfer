package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// TransferClient 文件传输客户端
type TransferClient struct {
	serverURL  string
	filePath   string
	isDir      bool
	httpClient *http.Client
}

// NewTransferClient 创建新的传输客户端
func NewTransferClient() *TransferClient {
	// 创建优化的 HTTP 客户端，彻底解决 Windows 端口耗尽问题
	transport := &http.Transport{
		// 关键设置：限制连接数为1，强制串行和连接复用
		MaxConnsPerHost:     1,  // 每个主机只保持1个连接
		MaxIdleConnsPerHost: 1,  // 每个主机只保持1个空闲连接
		MaxIdleConns:        1,  // 总共只保持1个空闲连接
		IdleConnTimeout:     300 * time.Second, // 5分钟空闲超时
		DisableKeepAlives:   false, // 必须启用 Keep-Alive 来复用连接
		// 关键：强制 HTTP/1.1，避免 HTTP/2 的多路复用问题
		ForceAttemptHTTP2: false,
		// 增加响应头超时，避免慢速服务器导致的问题
		ResponseHeaderTimeout: 60 * time.Second,
		// 启用 TCP Keep-Alive 保持连接活跃
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second, // TCP Keep-Alive
		}).DialContext,
	}
	
	client := &http.Client{
		Timeout:   30 * time.Minute,
		Transport: transport,
	}
	
	return &TransferClient{
		httpClient: client,
	}
}


// Upload 执行上传
func (tc *TransferClient) Upload() error {
	fmt.Println("\n========================================")
	fmt.Println("⏳ 开始传输...")
	fmt.Println("========================================")
	
	startTime := time.Now()
	
	var err error
	if tc.isDir {
		err = tc.uploadDirectory()
	} else {
		err = tc.uploadFile()
	}
	
	if err != nil {
		return fmt.Errorf("❌ 传输失败: %v", err)
	}
	
	elapsed := time.Since(startTime)
	fmt.Printf("\n✅ 传输成功！\n")
	fmt.Printf("   总耗时: %.1f秒\n", elapsed.Seconds())
	
	return nil
}

// uploadFile 上传单个文件
func (tc *TransferClient) uploadFile() error {
	file, err := os.Open(tc.filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	fileInfo, _ := file.Stat()
	fileSize := fileInfo.Size()
	// 单个文件上传时，只使用文件名，不包含路径
	fileName := filepath.Base(tc.filePath)
	
	fmt.Printf("📁 文件: %s\n", fileName)
	fmt.Printf("📊 大小: %s\n", formatSize(fileSize))
	
	// 创建进度读取器
	reader := &progressReader{
		Reader:    file,
		Total:     fileSize,
		Current:   0,
		StartTime: time.Now(),
	}
	
	// 构建上传URL，文件名不包含路径
	uploadURL := fmt.Sprintf("%s/upload?name=%s", tc.serverURL, url.QueryEscape(fileName))
	
	// 创建请求
	req, err := http.NewRequest("POST", uploadURL, reader)
	if err != nil {
		return err
	}
	
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = fileSize
	
	// 执行上传（使用共享的客户端）
	resp, err := tc.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("服务器返回错误: %s", string(body))
	}
	
	fmt.Println() // 换行
	return nil
}

// uploadDirectory 上传目录（逐个上传文件，保留路径结构）
func (tc *TransferClient) uploadDirectory() error {
	// 获取目录名称作为路径前缀
	baseDir := filepath.Base(tc.filePath)
	
	// 收集所有文件信息
	var files []struct {
		path     string
		relPath  string
		size     int64
	}
	
	var totalSize int64
	
	// 遍历目录收集文件信息
	err := filepath.Walk(tc.filePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// 跳过目录
		if info.IsDir() {
			return nil
		}
		
		// 计算相对路径（相对于传入的目录）
		relPath, err := filepath.Rel(tc.filePath, path)
		if err != nil {
			return err
		}
		
		// 构建包含目录名的完整路径
		uploadName := filepath.Join(baseDir, relPath)
		// 将路径分隔符统一为斜杠（跨平台兼容）
		uploadName = strings.ReplaceAll(uploadName, string(filepath.Separator), "/")
		
		files = append(files, struct {
			path     string
			relPath  string
			size     int64
		}{
			path:    path,
			relPath: uploadName,
			size:    info.Size(),
		})
		
		totalSize += info.Size()
		return nil
	})
	
	if err != nil {
		return err
	}
	
	if len(files) == 0 {
		return fmt.Errorf("目录中没有文件")
	}
	
	fmt.Printf("📂 准备上传 %d 个文件，总大小: %s\n\n", len(files), formatSize(totalSize))
	
	// 在 Windows 上显示优化提示
	if runtime.GOOS == "windows" && len(files) > 50 {
		fmt.Println("💡 提示: 检测到大量文件传输，已启用 Windows 端口优化策略")
		fmt.Println("   - 使用单连接复用技术")
		fmt.Println("   - 自动重试机制")
		fmt.Println("   - 智能延迟控制")
		fmt.Println()
	}
	
	// 逐个上传文件（严格串行，一次只上传一个）
	for i, fileInfo := range files {
		fmt.Printf("[%d/%d] 上传: %s (%s)\n", i+1, len(files), fileInfo.relPath, formatSize(fileInfo.size))
		
		// 上传单个文件
		err := tc.uploadSingleFile(fileInfo.path, fileInfo.relPath, fileInfo.size)
		if err != nil {
			// 如果是端口耗尽错误，显示优化建议
			if strings.Contains(err.Error(), "Only one usage of each socket address") {
				fmt.Println("\n❌ 检测到 Windows 端口耗尽问题")
				OptimizeWindowsTCP()
			}
			return fmt.Errorf("上传失败 %s: %v", fileInfo.relPath, err)
		}
		
		fmt.Println() // 进度条后换行
	}
	
	return nil
}

// uploadSingleFile 上传单个文件（内部方法）
func (tc *TransferClient) uploadSingleFile(filePath, uploadName string, fileSize int64) error {
	// 重试机制，最多重试3次
	maxRetries := 3
	var lastErr error
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// 如果是重试，等待一段时间让系统释放端口
		if attempt > 1 {
			waitTime := time.Duration(attempt-1) * 2 * time.Second
			fmt.Printf("\n⏳ 等待 %v 后重试 (第 %d/%d 次)...\n", waitTime, attempt, maxRetries)
			time.Sleep(waitTime)
		}
		
		// 执行上传
		err := tc.doUploadSingleFile(filePath, uploadName, fileSize)
		if err == nil {
			// 上传成功后，在 Windows 上等待一小段时间
			// 让系统有时间释放端口，避免下一个文件上传时端口耗尽
			if runtime.GOOS == "windows" {
				time.Sleep(100 * time.Millisecond)
			}
			return nil
		}
		
		lastErr = err
		
		// 检查是否是端口耗尽错误
		if strings.Contains(err.Error(), "Only one usage of each socket address") ||
			strings.Contains(err.Error(), "EADDRINUSE") ||
			strings.Contains(err.Error(), "address already in use") {
			// 端口耗尽，等待更长时间
			if attempt < maxRetries {
				fmt.Printf("\n⚠️ 检测到端口耗尽，等待系统释放资源...\n")
				time.Sleep(5 * time.Second)
			}
		}
	}
	
	return fmt.Errorf("重试 %d 次后仍然失败: %v", maxRetries, lastErr)
}

// doUploadSingleFile 实际执行上传
func (tc *TransferClient) doUploadSingleFile(filePath, uploadName string, fileSize int64) error {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()
	
	// 创建进度读取器
	reader := &progressReader{
		Reader:    file,
		Total:     fileSize,
		Current:   0,
		StartTime: time.Now(),
	}
	
	// 构建上传URL
	uploadURL := fmt.Sprintf("%s/upload?name=%s", tc.serverURL, url.QueryEscape(uploadName))
	
	// 创建请求
	req, err := http.NewRequest("POST", uploadURL, reader)
	if err != nil {
		return err
	}
	
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = fileSize
	// 强制使用 HTTP/1.1 并启用 Keep-Alive
	req.Header.Set("Connection", "keep-alive")
	req.ProtoMajor = 1
	req.ProtoMinor = 1
	
	// 执行上传（使用共享的客户端）
	resp, err := tc.httpClient.Do(req)
	if err != nil {
		return err
	}
	
	// 读取响应体（确保连接可以被复用）
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close() // 立即关闭响应体
	
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}
	
	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器返回错误: %s", string(body))
	}
	
	return nil
}

// progressReader 带进度显示的读取器
type progressReader struct {
	io.Reader
	Total     int64
	Current   int64
	StartTime time.Time
	LastPrint time.Time
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Current += int64(n)
	
	// 每100ms更新一次进度
	now := time.Now()
	if now.Sub(pr.LastPrint) >= 100*time.Millisecond || err == io.EOF {
		pr.printProgress()
		pr.LastPrint = now
	}
	
	return n, err
}

func (pr *progressReader) printProgress() {
	if pr.Total == 0 {
		return
	}
	
	percentage := float64(pr.Current) * 100 / float64(pr.Total)
	elapsed := time.Since(pr.StartTime).Seconds()
	
	speed := float64(0)
	eta := float64(0)
	if elapsed > 0 {
		speed = float64(pr.Current) / elapsed
		if speed > 0 {
			eta = float64(pr.Total-pr.Current) / speed
		}
	}
	
	// 构建进度条
	barLength := 40
	filled := int(float64(barLength) * float64(pr.Current) / float64(pr.Total))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barLength-filled)
	
	// 构建固定长度的输出字符串，避免残影
	const lineWidth = 120 // 固定行宽
	
	// 格式化各个部分，确保固定宽度
	percentStr := fmt.Sprintf("%5.1f%%", percentage) // 固定5字符宽
	sizeStr := fmt.Sprintf("%s/%s", formatSize(pr.Current), formatSize(pr.Total))
	speedStr := fmt.Sprintf("%s/s", formatSize(int64(speed)))
	
	output := fmt.Sprintf("上传进度: [%s] %s %-20s 速度: %-12s",
		bar, percentStr, sizeStr, speedStr)
	
	if eta > 0 && pr.Current < pr.Total {
		etaStr := fmt.Sprintf("剩余: %d秒", int(eta))
		output = fmt.Sprintf("%s %-15s", output, etaStr)
	}
	
	// 使用固定宽度输出，多余部分用空格填充，避免残影
	fmt.Printf("\r%-*s", lineWidth, output)
}

// getDirStats 获取目录统计信息
func (tc *TransferClient) getDirStats(dirPath string) (int, int64) {
	var fileCount int
	var totalSize int64
	
	filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			fileCount++
			totalSize += info.Size()
		}
		return nil
	})
	
	return fileCount, totalSize
}

// formatSize 格式化文件大小
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}


// runConfiguredClient 根据配置运行客户端
func runConfiguredClient(config *Config) {
	client := NewTransferClient()
	client.filePath = expandPath(config.FilePath)
	client.serverURL = config.TargetURL
	
	// 检查文件/目录
	fileInfo, err := os.Stat(client.filePath)
	if err != nil {
		fmt.Printf("❌ 路径不存在: %s\n", client.filePath)
		os.Exit(1)
	}
	
	client.isDir = fileInfo.IsDir()
	
	// 验证URL
	if !strings.HasPrefix(client.serverURL, "http://") && !strings.HasPrefix(client.serverURL, "https://") {
		client.serverURL = "http://" + client.serverURL
	}
	client.serverURL = strings.TrimSuffix(client.serverURL, "/")
	
	// 显示传输信息
	fmt.Println("\n========================================")
	fmt.Println("📁 准备传输")
	fmt.Println("========================================")
	if client.isDir {
		fileCount, totalSize := client.getDirStats(client.filePath)
		fmt.Printf("📂 目录: %s\n", client.filePath)
		fmt.Printf("   包含 %d 个文件，总大小: %s\n", fileCount, formatSize(totalSize))
	} else {
		fmt.Printf("📄 文件: %s\n", client.filePath)
		fmt.Printf("   大小: %s\n", formatSize(fileInfo.Size()))
	}
	fmt.Printf("🎯 目标: %s\n", client.serverURL)
	
	// 确认上传
	fmt.Print("\n确认开始传输？[Y/n]: ")
	var confirm string
	fmt.Scanln(&confirm)
	confirm = strings.TrimSpace(strings.ToLower(confirm))
	
	// 默认为 Y，只有明确输入 n 才取消
	if confirm == "n" || confirm == "no" {
		fmt.Println("已取消传输")
		return
	}
	
	// 执行上传
	if err := client.Upload(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

