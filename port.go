package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// checkPortInUse 检查端口是否被占用
func checkPortInUse(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return true // 端口被占用
	}
	listener.Close()
	return false // 端口可用
}

// findProcessUsingPort 查找占用端口的进程
func findProcessUsingPort(port int) (pid int, processName string, err error) {
	switch runtime.GOOS {
	case "darwin", "linux":
		return findProcessUnix(port)
	case "windows":
		return findProcessWindows(port)
	default:
		return 0, "", fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

// findProcessUnix Unix系统（macOS/Linux）查找进程
func findProcessUnix(port int) (int, string, error) {
	var cmd *exec.Cmd
	
	if runtime.GOOS == "darwin" {
		// macOS 使用 lsof
		cmd = exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port))
	} else {
		// Linux 使用 lsof 或 ss
		// 先尝试 lsof
		cmd = exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port))
		if _, err := exec.LookPath("lsof"); err != nil {
			// 如果没有 lsof，尝试使用 ss
			cmd = exec.Command("sh", "-c", fmt.Sprintf("ss -tlnp | grep ':%d' | awk '{print $NF}' | grep -o '[0-9]*'", port))
		}
	}
	
	output, err := cmd.Output()
	if err != nil {
		return 0, "", fmt.Errorf("无法找到占用端口的进程")
	}
	
	pidStr := strings.TrimSpace(string(output))
	if pidStr == "" {
		return 0, "", fmt.Errorf("未找到占用端口的进程")
	}
	
	// 可能返回多个PID，取第一个
	pids := strings.Split(pidStr, "\n")
	if len(pids) > 0 {
		pidStr = pids[0]
	}
	
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, "", fmt.Errorf("无效的PID: %s", pidStr)
	}
	
	// 获取进程名称
	nameCmd := exec.Command("ps", "-p", pidStr, "-o", "comm=")
	nameOutput, err := nameCmd.Output()
	if err != nil {
		return pid, "unknown", nil
	}
	
	processName := strings.TrimSpace(string(nameOutput))
	return pid, processName, nil
}

// findProcessWindows Windows系统查找进程
func findProcessWindows(port int) (int, string, error) {
	cmd := exec.Command("netstat", "-ano")
	output, err := cmd.Output()
	if err != nil {
		return 0, "", fmt.Errorf("执行netstat失败: %v", err)
	}
	
	lines := strings.Split(string(output), "\n")
	portStr := fmt.Sprintf(":%d", port)
	
	for _, line := range lines {
		if strings.Contains(line, portStr) && strings.Contains(line, "LISTENING") {
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				pidStr := fields[len(fields)-1]
				pid, err := strconv.Atoi(pidStr)
				if err != nil {
					continue
				}
				
				// 获取进程名称
				nameCmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH")
				nameOutput, err := nameCmd.Output()
				if err != nil {
					return pid, "unknown", nil
				}
				
				// 解析CSV输出
				parts := strings.Split(string(nameOutput), ",")
				if len(parts) > 0 {
					processName := strings.Trim(parts[0], "\"")
					return pid, processName, nil
				}
				
				return pid, "unknown", nil
			}
		}
	}
	
	return 0, "", fmt.Errorf("未找到占用端口的进程")
}

// killProcess 杀死进程
func killProcess(pid int) error {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("taskkill", "/F", "/PID", strconv.Itoa(pid))
		return cmd.Run()
	}
	
	// Unix系统
	cmd := exec.Command("kill", "-9", strconv.Itoa(pid))
	return cmd.Run()
}

// handlePortConflict 处理端口冲突
func handlePortConflict(port int) bool {
	fmt.Printf("\n⚠️  端口 %d 已被占用\n", port)
	
	// 查找占用端口的进程
	pid, processName, err := findProcessUsingPort(port)
	if err != nil {
		fmt.Printf("无法确定占用端口的进程: %v\n", err)
		fmt.Println("\n请手动释放端口或选择其他端口")
		return false
	}
	
	fmt.Printf("占用进程: %s (PID: %d)\n", processName, pid)
	
	// 询问用户是否杀死进程
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n是否杀死该进程并释放端口? [y/N]: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return false
		}
		
		answer := strings.ToLower(strings.TrimSpace(input))
		if answer == "y" || answer == "yes" {
			// 杀死进程
			if err := killProcess(pid); err != nil {
				fmt.Printf("❌ 无法杀死进程: %v\n", err)
				fmt.Println("请尝试使用管理员权限运行")
				return false
			}
			
			fmt.Printf("✅ 已杀死进程 %s (PID: %d)\n", processName, pid)
			
			// 等待一下让端口释放
			fmt.Print("等待端口释放...")
			for i := 0; i < 3; i++ {
				if !checkPortInUse(port) {
					fmt.Println(" 完成!")
					return true
				}
				fmt.Print(".")
				// 短暂等待
				cmd := exec.Command("sleep", "1")
				if runtime.GOOS == "windows" {
					cmd = exec.Command("timeout", "/t", "1", "/nobreak")
				}
				cmd.Run()
			}
			
			// 再次检查
			if !checkPortInUse(port) {
				fmt.Println(" 完成!")
				return true
			}
			
			fmt.Println("\n❌ 端口仍然被占用，可能需要更多时间释放")
			return false
			
		} else if answer == "n" || answer == "no" || answer == "" {
			fmt.Println("请选择其他端口或手动释放端口")
			return false
		}
		
		fmt.Println("请输入 y(是) 或 n(否)")
	}
}