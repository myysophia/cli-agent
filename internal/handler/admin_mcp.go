package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type AdminMCPMeta struct {
	Path         string `json:"path"`
	DisplayPath  string `json:"display_path"`
	Exists       bool   `json:"exists"`
	LastModified string `json:"last_modified,omitempty"`
}

type AdminMCPEnvItem struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Masked bool   `json:"masked"`
}

type AdminMCPServer struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     []AdminMCPEnvItem `json:"env,omitempty"`
}

type AdminMCPResponse struct {
	Servers []AdminMCPServer `json:"servers"`
	Meta    AdminMCPMeta     `json:"meta"`
}

type cursorMCPConfig struct {
	MCPServers map[string]cursorMCPServer `json:"mcpServers"`
}

type cursorMCPServer struct {
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

type claudeSettings struct {
	Raw        map[string]json.RawMessage
	MCPServers map[string]cursorMCPServer
	ServersKey string
}

func handleAdminMCP(w http.ResponseWriter, r *http.Request, relativePath string) {
	claudeBase := "/api/mcp/claude"
	if relativePath == claudeBase {
		switch r.Method {
		case http.MethodGet:
			handleClaudeMCPList(w)
		case http.MethodPost:
			handleClaudeMCPCreate(w, r)
		default:
			writeMethodNotAllowed(w)
		}
		return
	}
	if strings.HasPrefix(relativePath, claudeBase+"/") {
		name := strings.TrimPrefix(relativePath, claudeBase+"/")
		if name == "" || strings.Contains(name, "/") {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		switch r.Method {
		case http.MethodPut:
			handleClaudeMCPUpdate(w, r, name)
		case http.MethodDelete:
			handleClaudeMCPDelete(w, name)
		default:
			writeMethodNotAllowed(w)
		}
		return
	}

	base := "/api/mcp/cursor"
	if relativePath == base {
		switch r.Method {
		case http.MethodGet:
			handleCursorMCPList(w)
		case http.MethodPost:
			handleCursorMCPCreate(w, r)
		default:
			writeMethodNotAllowed(w)
		}
		return
	}

	if strings.HasPrefix(relativePath, base+"/") {
		name := strings.TrimPrefix(relativePath, base+"/")
		if name == "" || strings.Contains(name, "/") {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		switch r.Method {
		case http.MethodPut:
			handleCursorMCPUpdate(w, r, name)
		case http.MethodDelete:
			handleCursorMCPDelete(w, name)
		default:
			writeMethodNotAllowed(w)
		}
		return
	}

	writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
}

func handleCursorMCPList(w http.ResponseWriter) {
	cfg, meta, err := loadCursorMCPConfig()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	servers := make([]AdminMCPServer, 0)
	keys := make([]string, 0, len(cfg.MCPServers))
	for name := range cfg.MCPServers {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	for _, name := range keys {
		servers = append(servers, buildAdminMCPServer(name, cfg.MCPServers[name]))
	}

	writeJSON(w, http.StatusOK, AdminMCPResponse{
		Servers: servers,
		Meta:    meta,
	})
}

func handleClaudeMCPList(w http.ResponseWriter) {
	settings, meta, err := loadClaudeMCPConfig()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	servers := make([]AdminMCPServer, 0)
	keys := make([]string, 0, len(settings.MCPServers))
	for name := range settings.MCPServers {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	for _, name := range keys {
		servers = append(servers, buildAdminMCPServer(name, settings.MCPServers[name]))
	}

	writeJSON(w, http.StatusOK, AdminMCPResponse{
		Servers: servers,
		Meta:    meta,
	})
}

func handleCursorMCPCreate(w http.ResponseWriter, r *http.Request) {
	var payload AdminMCPServer
	if err := decodeAdminMCPPayload(w, r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if payload.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "server name is required"})
		return
	}
	if payload.Command == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "command is required"})
		return
	}

	cfg, meta, err := loadCursorMCPConfig()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if _, exists := cfg.MCPServers[payload.Name]; exists {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "server already exists"})
		return
	}

	cfg.MCPServers[payload.Name] = buildCursorMCPServer(payload, cursorMCPServer{})
	if err := writeCursorMCPConfig(cfg, meta.Path); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, buildAdminMCPServer(payload.Name, cfg.MCPServers[payload.Name]))
}

func handleClaudeMCPCreate(w http.ResponseWriter, r *http.Request) {
	var payload AdminMCPServer
	if err := decodeAdminMCPPayload(w, r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if payload.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "server name is required"})
		return
	}
	if payload.Command == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "command is required"})
		return
	}

	settings, meta, err := loadClaudeMCPConfig()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if _, exists := settings.MCPServers[payload.Name]; exists {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "server already exists"})
		return
	}

	settings.MCPServers[payload.Name] = buildCursorMCPServer(payload, cursorMCPServer{})
	if err := writeClaudeMCPConfig(settings, meta.Path); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, buildAdminMCPServer(payload.Name, settings.MCPServers[payload.Name]))
}

func handleCursorMCPUpdate(w http.ResponseWriter, r *http.Request, name string) {
	var payload AdminMCPServer
	if err := decodeAdminMCPPayload(w, r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	cfg, meta, err := loadCursorMCPConfig()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	existing, exists := cfg.MCPServers[name]
	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "server not found"})
		return
	}

	payload.Name = name
	cfg.MCPServers[name] = buildCursorMCPServer(payload, existing)
	if err := writeCursorMCPConfig(cfg, meta.Path); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, buildAdminMCPServer(name, cfg.MCPServers[name]))
}

func handleClaudeMCPUpdate(w http.ResponseWriter, r *http.Request, name string) {
	var payload AdminMCPServer
	if err := decodeAdminMCPPayload(w, r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	settings, meta, err := loadClaudeMCPConfig()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	existing, exists := settings.MCPServers[name]
	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "server not found"})
		return
	}

	payload.Name = name
	settings.MCPServers[name] = buildCursorMCPServer(payload, existing)
	if err := writeClaudeMCPConfig(settings, meta.Path); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, buildAdminMCPServer(name, settings.MCPServers[name]))
}

func handleCursorMCPDelete(w http.ResponseWriter, name string) {
	cfg, meta, err := loadCursorMCPConfig()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if _, exists := cfg.MCPServers[name]; !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "server not found"})
		return
	}
	delete(cfg.MCPServers, name)
	if err := writeCursorMCPConfig(cfg, meta.Path); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func handleClaudeMCPDelete(w http.ResponseWriter, name string) {
	settings, meta, err := loadClaudeMCPConfig()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if _, exists := settings.MCPServers[name]; !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "server not found"})
		return
	}
	delete(settings.MCPServers, name)
	if err := writeClaudeMCPConfig(settings, meta.Path); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func decodeAdminMCPPayload(w http.ResponseWriter, r *http.Request, payload *AdminMCPServer) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	if err := decoder.Decode(payload); err != nil {
		return fmt.Errorf("invalid mcp payload")
	}
	return nil
}

func loadCursorMCPConfig() (cursorMCPConfig, AdminMCPMeta, error) {
	path, display := resolveCursorMCPPath()
	meta := AdminMCPMeta{
		Path:        path,
		DisplayPath: display,
		Exists:      fileExists(path),
	}
	cfg := cursorMCPConfig{
		MCPServers: map[string]cursorMCPServer{},
	}
	if !meta.Exists {
		return cfg, meta, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, meta, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, meta, err
	}
	if cfg.MCPServers == nil {
		cfg.MCPServers = map[string]cursorMCPServer{}
	}

	if stat, err := os.Stat(path); err == nil {
		meta.LastModified = stat.ModTime().Format(time.RFC3339)
	}
	return cfg, meta, nil
}

func writeCursorMCPConfig(cfg cursorMCPConfig, path string) error {
	if path == "" {
		return fmt.Errorf("mcp path not resolved")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func loadClaudeMCPConfig() (claudeSettings, AdminMCPMeta, error) {
	path, display := resolveClaudeMCPPath()
	meta := AdminMCPMeta{
		Path:        path,
		DisplayPath: display,
		Exists:      fileExists(path),
	}
	settings := claudeSettings{
		Raw:        map[string]json.RawMessage{},
		MCPServers: map[string]cursorMCPServer{},
		ServersKey: "mcpServers",
	}
	if !meta.Exists {
		return settings, meta, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return settings, meta, err
	}
	if err := json.Unmarshal(data, &settings.Raw); err != nil {
		return settings, meta, err
	}
	for _, key := range []string{"mcpServers", "mcp_servers"} {
		if raw, ok := settings.Raw[key]; ok && len(raw) > 0 {
			if err := json.Unmarshal(raw, &settings.MCPServers); err != nil {
				return settings, meta, err
			}
			settings.ServersKey = key
			break
		}
	}
	if settings.MCPServers == nil {
		settings.MCPServers = map[string]cursorMCPServer{}
	}
	if stat, err := os.Stat(path); err == nil {
		meta.LastModified = stat.ModTime().Format(time.RFC3339)
	}
	return settings, meta, nil
}

func writeClaudeMCPConfig(settings claudeSettings, path string) error {
	if path == "" {
		return fmt.Errorf("mcp path not resolved")
	}
	if settings.Raw == nil {
		settings.Raw = map[string]json.RawMessage{}
	}
	mcpRaw, err := json.Marshal(settings.MCPServers)
	if err != nil {
		return err
	}
	key := settings.ServersKey
	if key == "" {
		key = "mcpServers"
	}
	settings.Raw[key] = mcpRaw

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings.Raw, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func resolveCursorMCPPath() (string, string) {
	if value := os.Getenv("ADMIN_MCP_CURSOR_PATH"); value != "" {
		return absPath(value)
	}
	if value := os.Getenv("CURSOR_MCP_PATH"); value != "" {
		return absPath(value)
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return absPath(".cursor/mcp.json")
	}
	return absPath(filepath.Join(home, ".cursor", "mcp.json"))
}

func resolveClaudeMCPPath() (string, string) {
	if value := os.Getenv("ADMIN_MCP_CLAUDE_PATH"); value != "" {
		return absPath(value)
	}
	if value := os.Getenv("CLAUDE_MCP_PATH"); value != "" {
		return absPath(value)
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return absPath(".claude/settings.json")
	}
	return absPath(filepath.Join(home, ".claude", "settings.json"))
}

func absPath(path string) (string, string) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return path, path
	}
	display := strings.Replace(absolute, string(filepath.Separator)+"Users"+string(filepath.Separator), string(filepath.Separator)+"Users"+string(filepath.Separator)+"~"+string(filepath.Separator), 1)
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		display = strings.Replace(absolute, home, "~", 1)
	}
	return absolute, display
}

func buildAdminMCPServer(name string, server cursorMCPServer) AdminMCPServer {
	envItems := make([]AdminMCPEnvItem, 0)
	keys := make([]string, 0, len(server.Env))
	for key := range server.Env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := server.Env[key]
		masked := isSensitiveEnvKey(key)
		if masked {
			value = maskValue(value)
		}
		envItems = append(envItems, AdminMCPEnvItem{
			Key:    key,
			Value:  value,
			Masked: masked,
		})
	}
	return AdminMCPServer{
		Name:    name,
		Command: server.Command,
		Args:    server.Args,
		Env:     envItems,
	}
}

func buildCursorMCPServer(payload AdminMCPServer, existing cursorMCPServer) cursorMCPServer {
	env := map[string]string{}
	for _, item := range payload.Env {
		if item.Key == "" {
			continue
		}
		if item.Masked {
			if existingValue, ok := existing.Env[item.Key]; ok {
				env[item.Key] = existingValue
			}
			continue
		}
		env[item.Key] = item.Value
	}

	return cursorMCPServer{
		Command: payload.Command,
		Args:    payload.Args,
		Env:     env,
	}
}
