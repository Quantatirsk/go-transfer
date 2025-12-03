package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"go-transfer/internal/config"
	"go-transfer/internal/infrastructure/logger"
	"go-transfer/internal/infrastructure/system"
	"go-transfer/internal/transfer/client"
	"go-transfer/internal/transfer/server"
)



func main() {
	// 命令行参数 - 简化处理
	verbose := flag.Bool("v", false, "详细模式")
	silent := flag.Bool("s", false, "静默模式")
	debug := flag.Bool("debug", false, "调试模式")
	flag.Parse()
	
	// 设置日志级别
	if *debug {
		logger.GlobalLogger.SetLevel(logger.DEBUG)
	} else if *verbose {
		logger.GlobalLogger.SetVerbose(true)
	} else if *silent {
		logger.GlobalLogger.SetSilent(true)
	}
	
	// 创建配置管理器
	cm := config.NewConfigManager()

	// 运行交互式配置
	cfg, err := cm.LoadOrCreateConfig()
	if err != nil {
		logger.LogError("配置错误: %v", err)
		os.Exit(1)
	}

	// 根据配置的模式执行相应功能
	switch cfg.Mode {
	case "client":
		// 客户端模式 - 上传文件
		runClient(cfg)

	case "receiver", "forward":
		// 服务器模式 - 启动服务
		ft := &server.FileTransfer{
			Mode:        cfg.Mode,
			Port:        cfg.Port,
			StoragePath: cfg.StoragePath,
			TargetURL:   cfg.TargetURL,
		}
		ft.Start()

	default:
		logger.LogError("未知模式: %s", cfg.Mode)
		os.Exit(1)
	}
}

// runClient 根据配置运行客户端
func runClient(cfg *config.Config) {
	transferClient := client.NewTransferClient()
	transferClient.SetFilePath(system.ExpandPath(cfg.FilePath))
	transferClient.SetServerURL(cfg.TargetURL)
	
	// 检查文件/目录
	fileInfo, err := os.Stat(system.ExpandPath(cfg.FilePath))
	if err != nil {
		logger.LogError("路径不存在: %s", cfg.FilePath)
		os.Exit(1)
	}
	
	transferClient.SetIsDir(fileInfo.IsDir())
	
	// 验证URL
	serverURL := cfg.TargetURL
	if !strings.HasPrefix(serverURL, "http://") && !strings.HasPrefix(serverURL, "https://") {
		serverURL = "http://" + serverURL
	}
	serverURL = strings.TrimSuffix(serverURL, "/")
	transferClient.SetServerURL(serverURL)
	
	// 显示传输信息
	fmt.Println()
	system.PrintSeparator()
	fmt.Println("📁 准备传输")
	system.PrintSeparator()
	if fileInfo.IsDir() {
		fileCount, totalSize := transferClient.GetDirStats(system.ExpandPath(cfg.FilePath))
		fmt.Printf("📂 目录: %s\n", cfg.FilePath)
		fmt.Printf("   包含 %d 个文件，总大小: %s\n", fileCount, system.FormatSize(totalSize))
	} else {
		fmt.Printf("📄 文件: %s\n", cfg.FilePath)
		fmt.Printf("   大小: %s\n", system.FormatSize(fileInfo.Size()))
	}
	fmt.Printf("🎯 目标: %s\n", serverURL)
	
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
	if err := transferClient.Upload(); err != nil {
		logger.LogError("%v", err)
		os.Exit(1)
	}
}
