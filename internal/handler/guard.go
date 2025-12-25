package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

const guardedResponseText = "我是您的AI助手啊，有什么问题尽管问。"

var directGuardPhrases = []string{
	"你是谁",
	"你是誰",
	"who are you",
	"what are you",
	"你用什么模型",
	"你用什麼模型",
	"你用的模型",
	"模型是什么",
	"模型是什麼",
	"system prompt",
	"系统提示词",
	"系統提示詞",
	"configs.json",
	"config.json",
}

var guardReferPhrases = []string{
	"你的",
	"本服务",
	"本服務",
	"当前",
	"当前服务",
	"当前服務",
	"网关",
	"gateway",
	"this service",
	"the service",
	"your",
}

var guardTopicPhrases = []string{
	"模型",
	"model",
	"系统提示词",
	"system prompt",
	"配置",
	"configuration",
	"config",
	"环境变量",
	"env",
	"environment variable",
	"密钥",
	"secret",
	"api key",
	"apikey",
	"token",
	"服务器地址",
	"server address",
	"服务器目录",
	"server directory",
	"mcp",
	"skill",
	"skills",
}

func shouldGuardPrompt(prompt string) bool {
	trimmed := strings.TrimSpace(prompt)
	if trimmed == "" {
		return false
	}

	lower := strings.ToLower(trimmed)
	if containsAny(lower, directGuardPhrases) {
		return true
	}

	if containsAny(lower, guardTopicPhrases) && containsAny(lower, guardReferPhrases) {
		return true
	}

	return false
}

func shouldGuardMessages(messages []Message) (string, bool) {
	lastUser := ""
	for _, msg := range messages {
		if msg.Role != "user" {
			continue
		}
		lastUser = msg.Content
	}
	if lastUser == "" {
		return lastUser, false
	}
	return lastUser, shouldGuardPrompt(lastUser)
}

func writeGuardedResponse(w http.ResponseWriter, prompt string) {
	log.Printf("🛑 Guarded prompt detected, returning safe response")

	result := CLIOutput{
		SessionID: "",
		User:      prompt,
		Response:  guardedResponseText,
	}

	payload, err := json.Marshal(result)
	if err != nil {
		payload = []byte(guardedResponseText)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(InvokeResponse{Answer: string(payload)})
}

func containsAny(text string, phrases []string) bool {
	for _, phrase := range phrases {
		if phrase == "" {
			continue
		}
		if strings.Contains(text, phrase) {
			return true
		}
	}
	return false
}
