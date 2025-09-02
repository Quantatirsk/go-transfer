package main

import (
	"fmt"
	"log"
	"os"
	"sync"
)

// LogLevel 日志级别
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	SILENT // 静默模式，不输出任何日志
)

// Logger 统一的日志管理器
type Logger struct {
	mu       sync.RWMutex
	level    LogLevel
	verbose  bool
	logger   *log.Logger
	disabled bool
}

// GlobalLogger 全局日志实例
var GlobalLogger = &Logger{
	level:   INFO,
	verbose: false,
	logger:  log.New(os.Stdout, "", log.LstdFlags),
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetVerbose 设置详细模式
func (l *Logger) SetVerbose(verbose bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.verbose = verbose
	if verbose {
		l.level = DEBUG
	}
}

// SetSilent 设置静默模式
func (l *Logger) SetSilent(silent bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.disabled = silent
}

// Debug 调试日志
func (l *Logger) Debug(format string, v ...interface{}) {
	l.log(DEBUG, "DEBUG", format, v...)
}

// Info 信息日志
func (l *Logger) Info(format string, v ...interface{}) {
	l.log(INFO, "", format, v...)
}

// Warn 警告日志
func (l *Logger) Warn(format string, v ...interface{}) {
	l.log(WARN, "⚠️ ", format, v...)
}

// Error 错误日志
func (l *Logger) Error(format string, v ...interface{}) {
	l.log(ERROR, "❌", format, v...)
}

// Success 成功日志（始终显示）
func (l *Logger) Success(format string, v ...interface{}) {
	l.mu.RLock()
	disabled := l.disabled
	l.mu.RUnlock()
	
	if !disabled {
		msg := fmt.Sprintf(format, v...)
		fmt.Printf("✅ %s\n", msg)
	}
}

// Progress 进度日志（始终显示）
func (l *Logger) Progress(format string, v ...interface{}) {
	l.mu.RLock()
	disabled := l.disabled
	l.mu.RUnlock()
	
	if !disabled {
		msg := fmt.Sprintf(format, v...)
		fmt.Printf("%s\n", msg)
	}
}

// Print 直接输出（用于必要的用户交互）
func (l *Logger) Print(format string, v ...interface{}) {
	fmt.Printf(format, v...)
}

// Println 直接输出带换行
func (l *Logger) Println(v ...interface{}) {
	fmt.Println(v...)
}

// 内部日志方法
func (l *Logger) log(level LogLevel, prefix string, format string, v ...interface{}) {
	l.mu.RLock()
	currentLevel := l.level
	disabled := l.disabled
	l.mu.RUnlock()
	
	if disabled || level < currentLevel {
		return
	}
	
	msg := fmt.Sprintf(format, v...)
	if prefix != "" {
		msg = prefix + " " + msg
	}
	
	if level == DEBUG {
		l.logger.Printf("[DEBUG] %s", msg)
	} else {
		fmt.Println(msg)
	}
}

// 便捷函数，使用全局Logger

func LogDebug(format string, v ...interface{}) {
	GlobalLogger.Debug(format, v...)
}

func LogInfo(format string, v ...interface{}) {
	GlobalLogger.Info(format, v...)
}

func LogWarn(format string, v ...interface{}) {
	GlobalLogger.Warn(format, v...)
}

func LogError(format string, v ...interface{}) {
	GlobalLogger.Error(format, v...)
}

func LogSuccess(format string, v ...interface{}) {
	GlobalLogger.Success(format, v...)
}

func LogProgress(format string, v ...interface{}) {
	GlobalLogger.Progress(format, v...)
}

