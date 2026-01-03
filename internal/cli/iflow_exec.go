package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// IflowExecCLI 执行本地 iflow 命令
type IflowExecCLI struct{}

func NewIflowExecCLI() *IflowExecCLI {
	return &IflowExecCLI{}
}

func (i *IflowExecCLI) Name() string {
	return "iflow-exec"
}

func (i *IflowExecCLI) Run(opts *RunOptions) (string, error) {
	var args []string

	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
		log.Printf("🤖 [iFlow] Using model: %s", opts.Model)
	}

	if opts.PermissionMode == "bypassPermissions" {
		args = append(args, "--yolo")
		log.Printf("🔐 [iFlow] YOLO mode enabled")
	} else if opts.PermissionMode == "plan" {
		args = append(args, "--plan")
		log.Printf("🧭 [iFlow] Plan mode enabled")
	}

	if opts.SessionID != "" {
		args = append(args, "--resume", opts.SessionID)
		log.Printf("🔄 [iFlow] Resuming session: %s", opts.SessionID)
	}

	for _, dir := range opts.Skills {
		args = append(args, "--add-dir", dir)
	}
	if len(opts.Skills) > 0 {
		log.Printf("📚 [iFlow] Using %d skill(s): %v", len(opts.Skills), opts.Skills)
	}

	if opts.SystemPrompt != "" {
		log.Printf("⚠️  [iFlow] System prompt is not supported by CLI flags")
	}
	if len(opts.AllowedTools) > 0 {
		log.Printf("⚠️  [iFlow] Allowed tools are not supported by CLI flags")
	}

	if opts.Prompt != "" {
		args = append(args, "-p", opts.Prompt)
	}

	log.Printf("⚙️  [iFlow] Executing: iflow %s", strings.Join(args, " "))

	cmd := exec.Command("iflow", args...)
	cmd.Env = buildEnv(opts.Env)

	output, err := cmd.CombinedOutput()
	log.Printf("📊 [iFlow] Output length: %d bytes", len(output))
	log.Printf("🧾 [iFlow] Raw output:\n%s", string(output))

	if err != nil {
		log.Printf("❌ [iFlow] Execution error: %v", err)
		return "", fmt.Errorf("iflow CLI execution failed: %v, output: %s", err, string(output))
	}

	return i.parseOutput(string(output), opts.Prompt)
}

func (i *IflowExecCLI) parseOutput(output string, prompt string) (string, error) {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return "", fmt.Errorf("empty output from iflow CLI")
	}

	if response, sessionID, ok := parseExecutionInfoOutput(trimmed); ok {
		return i.wrapOutput(prompt, sessionID, response), nil
	}

	jsonStart := strings.Index(trimmed, "{")
	if jsonStart == -1 {
		return i.wrapOutput(prompt, "", trimmed), nil
	}

	jsonOutput := trimmed[jsonStart:]
	var payload map[string]interface{}
	decoder := json.NewDecoder(strings.NewReader(jsonOutput))
	if err := decoder.Decode(&payload); err != nil {
		log.Printf("❌ [iFlow] JSON parse error: %v", err)
		return i.wrapOutput(prompt, "", trimmed), nil
	}

	response := pickFirstString(payload, "response", "result", "message", "output", "text")
	sessionID := pickFirstString(payload, "session_id", "sessionId")

	if response == "" {
		response = trimmed
	}

	return i.wrapOutput(prompt, sessionID, response), nil
}

func (i *IflowExecCLI) wrapOutput(prompt string, sessionID string, response string) string {
	result := CLIOutput{
		SessionID: sessionID,
		User:      prompt,
		Response:  response,
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return response
	}

	return string(jsonBytes)
}

func pickFirstString(payload map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			if text, ok := value.(string); ok {
				return text
			}
		}
	}
	return ""
}

type iflowExecutionInfo struct {
	SessionID string `json:"session-id"`
}

func parseExecutionInfoOutput(output string) (string, string, bool) {
	const startTag = "<Execution Info>"
	const endTag = "</Execution Info>"

	startIdx := strings.Index(output, startTag)
	endIdx := strings.Index(output, endTag)
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return "", "", false
	}

	responsePart := strings.TrimSpace(output[:startIdx])
	infoRaw := strings.TrimSpace(output[startIdx+len(startTag) : endIdx])
	if infoRaw == "" {
		return responsePart, "", true
	}

	var info iflowExecutionInfo
	if err := json.Unmarshal([]byte(infoRaw), &info); err != nil {
		log.Printf("❌ [iFlow] Execution info parse error: %v", err)
		return responsePart, "", true
	}

	return responsePart, info.SessionID, true
}
