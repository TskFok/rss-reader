package logger

import (
	"log"
	"os"
	"strings"
	"sync"
)

// Level 日志等级
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

var (
	mu     sync.RWMutex
	level  = LevelInfo
	stdLog = log.New(os.Stderr, "", log.LstdFlags)
)

func parseLevel(s string) Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// Init 初始化日志等级，应在程序启动时调用
func Init(levelStr string) {
	mu.Lock()
	defer mu.Unlock()
	level = parseLevel(levelStr)
}

func shouldLog(l Level) bool {
	mu.RLock()
	defer mu.RUnlock()
	return l >= level
}

// Debug 输出 debug 日志
func Debug(format string, v ...any) {
	if shouldLog(LevelDebug) {
		stdLog.Printf("[DEBUG] "+format, v...)
	}
}

// Info 输出 info 日志
func Info(format string, v ...any) {
	if shouldLog(LevelInfo) {
		stdLog.Printf("[INFO] "+format, v...)
	}
}

// Warn 输出 warn 日志
func Warn(format string, v ...any) {
	if shouldLog(LevelWarn) {
		stdLog.Printf("[WARN] "+format, v...)
	}
}

// Error 输出 error 日志
func Error(format string, v ...any) {
	if shouldLog(LevelError) {
		stdLog.Printf("[ERROR] "+format, v...)
	}
}

// Fatal 输出 fatal 日志并退出，不受等级限制
func Fatal(v ...any) {
	stdLog.Fatal(v...)
}

// Fatalf 输出 fatal 日志并退出，不受等级限制
func Fatalf(format string, v ...any) {
	stdLog.Fatalf(format, v...)
}
