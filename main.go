package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func setupLogging() (*os.File, error) {
	// 创建 logs 目录
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, err
	}
	
	// 生成日志文件名（按日期）
	logFileName := filepath.Join(logsDir, time.Now().Format("2006-01-02")+".log")
	
	// 打开日志文件（追加模式）
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	
	// 设置日志同时输出到控制台和文件
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
	
	// 设置日志格式
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	
	return logFile, nil
}

func main() {
	// 设置日志
	logFile, err := setupLogging()
	if err != nil {
		log.Fatalf("Failed to setup logging: %v", err)
	}
	defer logFile.Close()
	
	log.Println("📁 Logging to file:", logFile.Name())
	
	// 初始化配置
	if err := initConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	// 使用 http.HandleFunc 注册 "/invoke" 路由到 handleInvoke
	http.HandleFunc("/invoke", handleInvoke)
	http.HandleFunc("/chat", handleChat)
	
	// 打印启动日志
	log.Println("🌐 Gateway service starting on :8080")
	
	// 调用 http.ListenAndServe 启动服务器，使用 log.Fatal 包装以处理启动错误
	log.Fatal(http.ListenAndServe(":8080", nil))
}
