package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

// workflowSessionMap 存储 workflow_run_id 到 session_id 的映射
var (
	workflowSessionMap = make(map[string]string)
	workflowSessionMu  sync.RWMutex
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

	// 调用 runCLI 函数执行 CLI
	log.Println("🚀 Calling CLI...")
	cliStart := time.Now()
	result, err := runCLI(req.CLI, prompt, req.System, req.Profile, "", false, nil, "")
	cliDuration := time.Since(cliStart)
	
	if err != nil {
		// 如果 runCLI 返回错误，返回 500 错误响应
		log.Printf("❌ CLI failed after %v: %v", cliDuration, err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	log.Printf("✅ CLI succeeded, response length: %d chars (took %v)", len(result), cliDuration)
	
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

// handleChat 处理 /chat 端点的简化 HTTP 请求
func handleChat(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	log.Printf("📥 Received request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
	
	// 检查 HTTP 方法是否为 POST
	if r.Method != http.MethodPost {
		log.Printf("❌ Method not allowed: %s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	// 解析请求体 JSON 到 ChatRequest 结构体
	parseStart := time.Now()
	var req ChatRequest
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
	log.Printf("📝 Request parsed - Prompt: %q, System: %q, Profile: %s (took %v)", 
		req.Prompt, req.System, profileInfo, parseDuration)
	
	// 处理 workflow_run_id：自动管理会话
	sessionID := req.SessionID
	newSession := req.NewSession
	
	if req.WorkflowRunID != "" {
		log.Printf("🔗 Workflow Run ID: %s", req.WorkflowRunID)
		
		// 检查是否已有对应的 session_id
		workflowSessionMu.RLock()
		existingSessionID, exists := workflowSessionMap[req.WorkflowRunID]
		workflowSessionMu.RUnlock()
		
		if exists {
			// 已存在，使用现有的 session_id
			sessionID = existingSessionID
			newSession = false
			log.Printf("♻️  Reusing existing session: %s", sessionID)
		} else {
			// 不存在，标记为新会话
			newSession = true
			log.Printf("🆕 New workflow run, will create new session")
		}
	}
	
	// 调用 runCLI 函数执行 CLI（传入 cli、prompt、system、profile、session_id、new_session、allowed_tools 和 permission_mode）
	log.Println("🚀 Calling CLI...")
	cliStart := time.Now()
	result, err := runCLI(req.CLI, req.Prompt, req.System, req.Profile, sessionID, newSession, req.AllowedTools, req.PermissionMode)
	cliDuration := time.Since(cliStart)
	
	if err != nil {
		// 如果 runCLI 返回错误，返回 500 错误响应
		log.Printf("❌ CLI failed after %v: %v", cliDuration, err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	log.Printf("✅ CLI succeeded, response length: %d chars (took %v)", len(result), cliDuration)
	
	// 如果有 workflow_run_id，保存映射关系
	if req.WorkflowRunID != "" && newSession {
		// 从返回的 JSON 中提取 session_id
		var codexOut CodexOutput
		if err := json.Unmarshal([]byte(result), &codexOut); err == nil && codexOut.SessionID != "" {
			workflowSessionMu.Lock()
			workflowSessionMap[req.WorkflowRunID] = codexOut.SessionID
			workflowSessionMu.Unlock()
			log.Printf("💾 Saved mapping: workflow_run_id=%s → session_id=%s", req.WorkflowRunID, codexOut.SessionID)
		}
	}
	
	// 如果成功，构建 InvokeResponse 并返回 200 响应
	// 设置响应头 Content-Type 为 application/json
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(InvokeResponse{Answer: result})
	
	totalDuration := time.Since(startTime)
	log.Printf("📤 Response sent successfully")
	log.Printf("⏱️  Total request time: %v (parse: %v, CLI: %v)", 
		totalDuration, parseDuration, cliDuration)
}
