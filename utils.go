package main

import (
	"fmt"
	"os/user"
	"path/filepath"
	"strings"
)

// formatSize 格式化文件大小为人类可读格式
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

// expandPath 展开路径中的 ~ 符号为用户主目录
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

// printSeparator 打印分隔线
func printSeparator() {
	fmt.Println(SeparatorLine)
}

// clearLine 清除当前行并输出固定宽度的内容（避免进度条残影）
func clearLine(content string) {
	fmt.Printf("\r%-*s", ProgressLineWidth, content)
}

// buildProgressBar 构建进度条字符串
func buildProgressBar(current, total int64) string {
	if total == 0 {
		return strings.Repeat("░", ProgressBarLength)
	}
	
	filled := int(float64(ProgressBarLength) * float64(current) / float64(total))
	if filled > ProgressBarLength {
		filled = ProgressBarLength
	}
	if filled < 0 {
		filled = 0
	}
	
	return strings.Repeat("█", filled) + strings.Repeat("░", ProgressBarLength-filled)
}

// calculateSpeed 计算传输速度
func calculateSpeed(bytes int64, elapsedSeconds float64) float64 {
	if elapsedSeconds <= 0 {
		return 0
	}
	return float64(bytes) / elapsedSeconds
}

// calculateETA 计算剩余时间（秒）
func calculateETA(current, total int64, elapsedSeconds float64) float64 {
	if current <= 0 || total <= 0 || elapsedSeconds <= 0 {
		return 0
	}
	
	speed := float64(current) / elapsedSeconds
	if speed <= 0 {
		return 0
	}
	
	return float64(total-current) / speed
}