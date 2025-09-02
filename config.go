package main

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 简化配置结构
type Config struct {
	Mode        string `yaml:"mode"`         // receiver, forward, client
	Port        int    `yaml:"port"`         // 监听端口（服务器模式）
	StoragePath string `yaml:"storage_path"` // receiver模式的存储路径
	TargetURL   string `yaml:"target_url"`   // forward模式的目标URL 或 client模式的服务器地址
	FilePath    string `yaml:"-"`            // client模式的文件/目录路径（不保存到配置文件）
}

// ConfigManager 配置管理器
type ConfigManager struct {
	configFile string
}

// NewConfigManager 创建配置管理器
func NewConfigManager() *ConfigManager {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "go-transfer")
	os.MkdirAll(configDir, 0755)

	return &ConfigManager{
		configFile: filepath.Join(configDir, "config.yaml"),
	}
}

// LoadOrCreateConfig 加载或创建配置
func (cm *ConfigManager) LoadOrCreateConfig() (*Config, error) {
	reader := bufio.NewReader(os.Stdin)
	
	// 尝试加载现有配置
	if config, err := cm.loadConfig(); err == nil {
		fmt.Printf("✅ 使用配置文件: %s\n", cm.configFile)
		cm.displayConfig(config)

		// 对于服务器模式，检查端口是否被占用
		if config.Mode != "client" && checkPortInUse(config.Port) {
			fmt.Printf("\n⚠️  检测到端口 %d 被占用\n", config.Port)
		}

		// 询问是否使用现有配置
		fmt.Print("\n使用现有配置？[Y/n]: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		
		if input == "" || input == "y" || input == "yes" {
			// 如果是客户端模式，先确认服务器地址，再输入文件路径
			if config.Mode == "client" {
				// 先询问是否更改服务器地址
				fmt.Printf("\n当前服务器: %s\n", config.TargetURL)
				fmt.Print("是否更改服务器地址？[y/N]: ")
				changeServer, _ := reader.ReadString('\n')
				changeServer = strings.TrimSpace(strings.ToLower(changeServer))
				
				if changeServer == "y" || changeServer == "yes" {
					fmt.Print("新的服务器地址: ")
					serverURL, _ := reader.ReadString('\n')
					serverURL = strings.TrimSpace(serverURL)
					if serverURL != "" {
						config.TargetURL = serverURL
						// 保存更新后的配置（只保存服务器地址）
						cm.saveConfig(config)
						fmt.Println("✅ 服务器地址已更新")
					}
				}
				
				// 然后输入文件路径
				fmt.Print("\n要传输的文件或目录路径: ")
				filePath, _ := reader.ReadString('\n')
				filePath = strings.TrimSpace(filePath)
				if filePath == "" {
					return nil, fmt.Errorf("文件路径不能为空")
				}
				
				// 验证文件/目录是否存在
				expandedPath := expandPath(filePath)
				if _, err := os.Stat(expandedPath); err != nil {
					return nil, fmt.Errorf("文件或目录不存在: %s", expandedPath)
				}
				config.FilePath = filePath
			}
			return config, nil
		}
	}

	// 创建新配置
	fmt.Println("\n🚀 go-transfer 配置向导")
	fmt.Println("=" + strings.Repeat("=", 40))

	config, err := cm.createConfig()
	if err != nil {
		return nil, err
	}

	// 保存配置
	if err := cm.saveConfig(config); err != nil {
		return nil, fmt.Errorf("保存配置失败: %v", err)
	}

	fmt.Printf("\n✅ 配置已保存到: %s\n", cm.configFile)
	return config, nil
}

// loadConfig 加载配置文件
func (cm *ConfigManager) loadConfig() (*Config, error) {
	data, err := os.ReadFile(cm.configFile)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// saveConfig 保存配置文件
func (cm *ConfigManager) saveConfig(config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(cm.configFile, data, 0644)
}

// createConfig 交互式创建配置
func (cm *ConfigManager) createConfig() (*Config, error) {
	reader := bufio.NewReader(os.Stdin)
	config := &Config{}

	// 选择模式
	fmt.Println("\n请选择运行模式:")
	fmt.Println("  1) receiver - 接收并存储文件（服务器模式）")
	fmt.Println("  2) forward  - 转发文件到下一跳（中继模式）")
	fmt.Println("  3) client   - 发送文件到服务器（客户端模式）")

	for {
		fmt.Print("\n请选择 [1-3]: ")
		input, _ := reader.ReadString('\n')
		trimmedInput := strings.TrimSpace(input)
		switch trimmedInput {
		case "1":
			config.Mode = "receiver"
		case "2":
			config.Mode = "forward"
		case "3":
			config.Mode = "client"
		default:
			fmt.Println("无效选择")
			continue
		}
		break
	}

	// 输入端口（仅服务器模式需要）
	if config.Mode != "client" {
		for {
			fmt.Print("\n监听端口 [17002]: ")
			portStr, _ := reader.ReadString('\n')
			portStr = strings.TrimSpace(portStr)
			if portStr == "" {
				config.Port = 17002
			} else {
				port, err := strconv.Atoi(portStr)
				if err != nil {
					fmt.Printf("无效端口: %v\n", err)
					continue
				}
				config.Port = port
			}

			// 检查端口是否被占用
			if checkPortInUse(config.Port) {
				fmt.Printf("\n⚠️  端口 %d 已被占用\n", config.Port)

				// 查找占用端口的进程
				pid, processName, err := findProcessUsingPort(config.Port)
				if err == nil {
					fmt.Printf("占用进程: %s (PID: %d)\n", processName, pid)
				}

				fmt.Println("请选择其他端口或先释放该端口")
				continue
			}

			break
		}
	}

	// 根据模式配置特定参数
	switch config.Mode {
	case "receiver":
		fmt.Print("\n存储路径 [~/uploads]: ")
		path, _ := reader.ReadString('\n')
		path = strings.TrimSpace(path)
		if path == "" {
			config.StoragePath = "~/uploads"
		} else {
			config.StoragePath = path
		}

	case "forward":
		fmt.Print("\n目标服务器URL: ")
		url, _ := reader.ReadString('\n')
		url = strings.TrimSpace(url)
		if url == "" {
			return nil, fmt.Errorf("目标URL不能为空")
		}
		config.TargetURL = url
		
	case "client":
		// 尝试加载之前的客户端配置作为默认值
		var defaultServer string
		if oldConfig, err := cm.loadConfig(); err == nil && oldConfig.Mode == "client" {
			defaultServer = oldConfig.TargetURL
		}
		
		// 先输入目标服务器，如果有默认值则显示
		if defaultServer != "" {
			fmt.Printf("\n目标服务器地址 [%s]: ", defaultServer)
		} else {
			fmt.Print("\n目标服务器地址 (例如: http://10.193.44.211:5000): ")
		}
		serverURL, _ := reader.ReadString('\n')
		serverURL = strings.TrimSpace(serverURL)
		
		// 如果用户没有输入，使用默认值
		if serverURL == "" {
			if defaultServer != "" {
				serverURL = defaultServer
				fmt.Printf("使用默认服务器: %s\n", serverURL)
			} else {
				return nil, fmt.Errorf("服务器地址不能为空")
			}
		}
		config.TargetURL = serverURL
		
		// 然后输入文件或目录路径
		fmt.Print("\n要传输的文件或目录路径: ")
		filePath, _ := reader.ReadString('\n')
		filePath = strings.TrimSpace(filePath)
		if filePath == "" {
			return nil, fmt.Errorf("文件路径不能为空")
		}
		config.FilePath = filePath
		
		// 验证文件/目录是否存在
		expandedPath := expandPath(filePath)
		if _, err := os.Stat(expandedPath); err != nil {
			return nil, fmt.Errorf("文件或目录不存在: %s", expandedPath)
		}
	}

	return config, nil
}

// displayConfig 显示配置
func (cm *ConfigManager) displayConfig(config *Config) {
	fmt.Println("\n📋 当前配置:")
	fmt.Printf("  模式: %s\n", config.Mode)
	
	switch config.Mode {
	case "receiver":
		fmt.Printf("  端口: %d\n", config.Port)
		fmt.Printf("  存储: %s\n", expandPath(config.StoragePath))
		fmt.Println("\n硬编码参数:")
		fmt.Println("  监听地址: 0.0.0.0")
		fmt.Println("  最大文件: 16GB")
		
	case "forward":
		fmt.Printf("  端口: %d\n", config.Port)
		fmt.Printf("  目标: %s\n", config.TargetURL)
		fmt.Println("\n硬编码参数:")
		fmt.Println("  监听地址: 0.0.0.0")
		fmt.Println("  最大文件: 16GB")
		
	case "client":
		fmt.Printf("  服务器: %s\n", config.TargetURL)
	}
	
	fmt.Println()
}

// expandPath 展开路径中的~符号
func expandPath(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	usr, err := user.Current()
	if err != nil {
		return path
	}

	if path == "~" {
		return usr.HomeDir
	}

	if strings.HasPrefix(path, "~/") {
		return filepath.Join(usr.HomeDir, path[2:])
	}

	return path
}
