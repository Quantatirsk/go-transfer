package main

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// ProgressTracker 统一的进度跟踪接口
type ProgressTracker interface {
	io.Writer
	io.Reader
	SetTotal(total int64)
	GetProgress() (current, total int64, percentage float64)
	GetSpeed() float64
	GetETA() time.Duration
	PrintProgress()
}

// Progress 统一的进度跟踪器实现
type Progress struct {
	mu          sync.RWMutex
	reader      io.Reader
	writer      io.Writer
	total       int64
	current     int64
	startTime   time.Time
	lastPrint   time.Time
	prefix      string
	showBar     bool
}

// NewProgressReader 创建带进度跟踪的Reader
func NewProgressReader(r io.Reader, total int64, prefix string) *Progress {
	return &Progress{
		reader:    r,
		total:     total,
		startTime: time.Now(),
		lastPrint: time.Now(),
		prefix:    prefix,
		showBar:   true,
	}
}

// NewProgressWriter 创建带进度跟踪的Writer
func NewProgressWriter(w io.Writer, total int64, prefix string) *Progress {
	return &Progress{
		writer:    w,
		total:     total,
		startTime: time.Now(),
		lastPrint: time.Now(),
		prefix:    prefix,
		showBar:   true,
	}
}

// Read 实现 io.Reader 接口
func (p *Progress) Read(b []byte) (int, error) {
	if p.reader == nil {
		return 0, io.EOF
	}
	
	n, err := p.reader.Read(b)
	p.addProgress(int64(n))
	
	// 定期更新进度显示
	if p.shouldPrint() || err == io.EOF {
		p.PrintProgress()
	}
	
	return n, err
}

// Write 实现 io.Writer 接口
func (p *Progress) Write(b []byte) (int, error) {
	if p.writer == nil {
		return 0, io.ErrClosedPipe
	}
	
	n, err := p.writer.Write(b)
	p.addProgress(int64(n))
	
	// 定期更新进度显示
	if p.shouldPrint() || err == io.EOF {
		p.PrintProgress()
	}
	
	return n, err
}

// SetTotal 设置总大小
func (p *Progress) SetTotal(total int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.total = total
}

// GetProgress 获取当前进度
func (p *Progress) GetProgress() (current, total int64, percentage float64) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	current = p.current
	total = p.total
	if total > 0 {
		percentage = float64(current) * 100 / float64(total)
	}
	return
}

// GetSpeed 获取传输速度（字节/秒）
func (p *Progress) GetSpeed() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	elapsed := time.Since(p.startTime).Seconds()
	if elapsed <= 0 {
		return 0
	}
	return float64(p.current) / elapsed
}

// GetETA 获取预计剩余时间
func (p *Progress) GetETA() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if p.total <= 0 || p.current <= 0 {
		return 0
	}
	
	speed := p.GetSpeed()
	if speed <= 0 {
		return 0
	}
	
	remaining := p.total - p.current
	seconds := float64(remaining) / speed
	return time.Duration(seconds) * time.Second
}

// PrintProgress 打印进度信息
func (p *Progress) PrintProgress() {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if !p.showBar {
		return
	}
	
	current, total, percentage := p.GetProgress()
	speed := p.GetSpeed()
	eta := p.GetETA()
	
	var output string
	
	if total > 0 {
		// 构建进度条
		bar := p.buildProgressBar(current, total)
		
		// 格式化输出
		percentStr := fmt.Sprintf("%5.1f%%", percentage)
		sizeStr := fmt.Sprintf("%s/%s", formatSize(current), formatSize(total))
		speedStr := fmt.Sprintf("%s/s", formatSize(int64(speed)))
		
		output = fmt.Sprintf("%s: [%s] %s %-20s 速度: %-12s",
			p.prefix, bar, percentStr, sizeStr, speedStr)
		
		if eta > 0 && current < total {
			etaStr := fmt.Sprintf("剩余: %d秒", int(eta.Seconds()))
			output = fmt.Sprintf("%s %-15s", output, etaStr)
		}
	} else {
		// 未知大小时的进度显示
		sizeStr := formatSize(current)
		speedStr := fmt.Sprintf("%s/s", formatSize(int64(speed)))
		output = fmt.Sprintf("%s: %-15s 速度: %-12s", p.prefix, sizeStr, speedStr)
	}
	
	// 使用固定宽度输出，避免残影
	clearLine(output)
	p.lastPrint = time.Now()
}

// 内部方法

func (p *Progress) addProgress(n int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current += n
}

func (p *Progress) shouldPrint() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return time.Since(p.lastPrint) >= ProgressUpdateInterval
}

func (p *Progress) buildProgressBar(current, total int64) string {
	if total <= 0 {
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

// ProgressConfig 进度显示配置
type ProgressConfig struct {
	ShowBar      bool          // 是否显示进度条
	UpdateInterval time.Duration // 更新间隔
	Prefix       string        // 前缀文本
}

// DefaultProgressConfig 默认配置
var DefaultProgressConfig = ProgressConfig{
	ShowBar:        true,
	UpdateInterval: ProgressUpdateInterval,
	Prefix:         "进度",
}