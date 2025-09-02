package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
	// 创建优化的 HTTP 客户端
	transport := &http.Transport{
		// 关键设置：限制连接数为1，强制串行和连接复用
		MaxConnsPerHost:     MaxConnsPerHost,
		MaxIdleConnsPerHost: MaxIdleConnsPerHost,
		MaxIdleConns:        MaxIdleConns,
		IdleConnTimeout:     IdleConnTimeout,
		DisableKeepAlives:   false, // 必须启用 Keep-Alive 来复用连接
		// 关键：强制 HTTP/1.1，避免 HTTP/2 的多路复用问题
		ForceAttemptHTTP2: false,
		// 增加响应头超时，避免慢速服务器导致的问题
		ResponseHeaderTimeout: ResponseTimeout,
		// 启用 TCP Keep-Alive 保持连接活跃
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second, // TCP Keep-Alive
		}).DialContext,
	}
	
	client := &http.Client{
		Timeout:   DefaultTimeout,
		Transport: transport,
	}
	
	return &TransferClient{
		httpClient: client,
	}
}


// Upload 执行上传
func (tc *TransferClient) Upload() error {
	fmt.Println()
	printSeparator()
	fmt.Println("⏳ 开始传输...")
	printSeparator()
	
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
	reader := NewProgressReader(file, fileSize, "上传进度")
	
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
	
	
	// 逐个上传文件（严格串行，一次只上传一个）
	for i, fileInfo := range files {
		fmt.Printf("[%d/%d] 上传: %s (%s)\n", i+1, len(files), fileInfo.relPath, formatSize(fileInfo.size))
		
		// 上传单个文件
		err := tc.uploadSingleFile(fileInfo.path, fileInfo.relPath, fileInfo.size)
		if err != nil {
			return fmt.Errorf("上传失败 %s: %v", fileInfo.relPath, err)
		}
		
		fmt.Println() // 进度条后换行
	}
	
	return nil
}

// uploadSingleFile 上传单个文件（内部方法）
func (tc *TransferClient) uploadSingleFile(filePath, uploadName string, fileSize int64) error {
	// 重试机制
	maxRetries := MaxRetries
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
				time.Sleep(PortExhaustWait)
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
	reader := NewProgressReader(file, fileSize, "上传进度")
	
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

// 注意：进度跟踪功能已移至 progress.go 统一管理
// 使用 NewProgressReader 创建进度跟踪器

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
	fmt.Println()
	printSeparator()
	fmt.Println("📁 准备传输")
	printSeparator()
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

