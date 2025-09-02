package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"archive/zip"
)

// TransferClient 文件传输客户端
type TransferClient struct {
	serverURL string
	filePath  string
	isDir     bool
}

// NewTransferClient 创建新的传输客户端
func NewTransferClient() *TransferClient {
	return &TransferClient{}
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
	
	// 构建上传URL
	uploadURL := fmt.Sprintf("%s/upload?name=%s", tc.serverURL, url.QueryEscape(fileName))
	
	// 创建请求
	req, err := http.NewRequest("POST", uploadURL, reader)
	if err != nil {
		return err
	}
	
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = fileSize
	
	// 执行上传
	client := &http.Client{
		Timeout: 30 * time.Minute,
	}
	
	resp, err := client.Do(req)
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

// uploadDirectory 上传目录（打包为zip）
func (tc *TransferClient) uploadDirectory() error {
	// 创建临时zip文件
	tempFile, err := os.CreateTemp("", "transfer-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()
	
	fmt.Printf("📦 正在打包目录...\n")
	
	// 创建zip写入器
	zipWriter := zip.NewWriter(tempFile)
	
	// 遍历目录并添加到zip
	baseDir := filepath.Base(tc.filePath)
	err = filepath.Walk(tc.filePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// 跳过目录本身
		if info.IsDir() {
			return nil
		}
		
		// 计算相对路径
		relPath, err := filepath.Rel(tc.filePath, path)
		if err != nil {
			return err
		}
		
		// 在zip中创建文件路径
		zipPath := filepath.Join(baseDir, relPath)
		
		// 创建zip文件条目
		writer, err := zipWriter.Create(zipPath)
		if err != nil {
			return err
		}
		
		// 打开源文件
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		
		// 复制文件内容
		_, err = io.Copy(writer, file)
		return err
	})
	
	if err != nil {
		return err
	}
	
	// 关闭zip写入器
	if err := zipWriter.Close(); err != nil {
		return err
	}
	
	// 获取zip文件大小
	zipInfo, err := tempFile.Stat()
	if err != nil {
		return err
	}
	
	fmt.Printf("✅ 打包完成，大小: %s\n\n", formatSize(zipInfo.Size()))
	
	// 重新打开文件进行上传
	tempFile.Seek(0, 0)
	
	// 创建进度读取器
	reader := &progressReader{
		Reader:    tempFile,
		Total:     zipInfo.Size(),
		Current:   0,
		StartTime: time.Now(),
	}
	
	// 构建上传URL
	zipName := filepath.Base(tc.filePath) + ".zip"
	uploadURL := fmt.Sprintf("%s/upload?name=%s", tc.serverURL, url.QueryEscape(zipName))
	
	fmt.Printf("📤 上传中: %s\n", zipName)
	
	// 创建请求
	req, err := http.NewRequest("POST", uploadURL, reader)
	if err != nil {
		return err
	}
	
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = zipInfo.Size()
	
	// 执行上传
	client := &http.Client{
		Timeout: 30 * time.Minute,
	}
	
	resp, err := client.Do(req)
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
	
	// 清除当前行并打印进度
	fmt.Printf("\r上传进度: [%s] %.1f%% (%s/%s) 速度: %s/s",
		bar, percentage,
		formatSize(pr.Current), formatSize(pr.Total),
		formatSize(int64(speed)))
	
	if eta > 0 && pr.Current < pr.Total {
		fmt.Printf(" 剩余: %d秒", int(eta))
	}
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

