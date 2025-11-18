package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// handleInvoke 处理 /invoke 端点的 HTTP 请求
func handleInvoke(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	log.Printf("📥 Received request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
	
	// 检查 HTTP 方法是否为 POST
	if r.Method != http.MethodPost {
		log.Printf("❌ Method not allowed: %s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	// 解析请求体 JSON 到 InvokeRequest 结构体
	parseStart := time.Now()
	var req InvokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Failed to parse JSON: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON request body"})
		return
	}
	parseDuration := time.Since(parseStart)

	profileInfo := req.Profile
	if profileInfo == "" {
		profileInfo = "default"
	}
	log.Printf("📝 Request parsed - System: %q, Messages: %d, Profile: %s (took %v)", 
		req.System, len(req.Messages), profileInfo, parseDuration)
	
	// 调用 buildPrompt 函数构建 prompt
	buildStart := time.Now()
	prompt := buildPrompt(req.Messages)
	buildDuration := time.Since(buildStart)
	log.Printf("🔨 Built prompt (%d chars, took %v)", len(prompt), buildDuration)

	// 调用 runClaude 函数执行 Claude CLI
	log.Println("🚀 Calling Claude CLI...")
	cliStart := time.Now()
	result, err := runClaude(prompt, req.System, req.Profile)
	cliDuration := time.Since(cliStart)
	
	if err != nil {
		// 如果 runClaude 返回错误，返回 500 错误响应
		log.Printf("❌ Claude CLI failed after %v: %v", cliDuration, err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	log.Printf("✅ Claude CLI succeeded, response length: %d chars (took %v)", len(result), cliDuration)
	
	// 如果成功，构建 InvokeResponse 并返回 200 响应
	// 设置响应头 Content-Type 为 application/json
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(InvokeResponse{Answer: result})
	
	totalDuration := time.Since(startTime)
	log.Printf("📤 Response sent successfully")
	log.Printf("⏱️  Total request time: %v (parse: %v, build: %v, CLI: %v)", 
		totalDuration, parseDuration, buildDuration, cliDuration)
}
