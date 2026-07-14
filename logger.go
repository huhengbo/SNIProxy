package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	logMu     sync.Mutex
	logFile   *os.File
	logPath   string
	enableDbg bool
)

func initLogger(path string, debug bool) error {
	enableDbg = debug
	logPath = path
	if path == "" {
		return nil
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}
	logFile = f
	return nil
}

func closeLogger() {
	logMu.Lock()
	defer logMu.Unlock()
	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
	}
}

// serviceLogger color: 0 默认, 31 红, 32 绿, 33 黄
func serviceLogger(log string, color int, isDebug bool) {
	if isDebug && !enableDbg {
		return
	}
	log = strings.ReplaceAll(log, "\n", "")
	line := time.Now().Format("2006/01/02 15:04:05") + " " + log

	if color == 0 {
		fmt.Printf("%s\n", line)
	} else {
		fmt.Printf("%c[1;0;%dm%s%c[0m\n", 0x1B, color, line, 0x1B)
	}

	if logFile != nil {
		logMu.Lock()
		_, _ = logFile.WriteString(line + "\n")
		logMu.Unlock()
	}
}
