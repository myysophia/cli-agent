# Design Document: CLI Release Notes Feature

## Overview

本设计文档描述了CLI Gateway的Release Notes功能实现方案。该功能允许用户通过HTTP API和可视化HTML界面查看各CLI工具（Claude、Codex、Cursor、Gemini、Qwen）的版本变更历史。

系统采用定时获取+缓存的架构，从多个数据源（GitHub Releases、NPM Registry、Cursor Changelog页面）获取release notes，并提供统一的JSON API和HTML可视化界面。

## Architecture

```mermaid
graph TB
    subgraph "External Sources"
        NPM[NPM Registry<br/>Claude CLI]
        GH1[GitHub Releases<br/>Codex CLI]
        GH2[GitHub Releases<br/>Gemini CLI]
        GH3[GitHub Releases<br/>Qwen CLI]
        CURSOR[Cursor Changelog<br/>cursor.com/changelog]
    end

    subgraph "CLI Gateway"
        FETCHER[Release Notes Fetcher]
        CACHE[In-Memory Cache]
        STORAGE[File Storage<br/>release_notes.json]
        SCHEDULER[Scheduler<br/>定时刷新]
        
        subgraph "HTTP Handlers"
            API[/release-notes API]
            VIEW[/release-notes/view HTML]
        end
    end

    subgraph "Clients"
        BROWSER[Browser]
        CLIENT[API Client]
    end

    NPM --> FETCHER
    GH1 --> FETCHER
    GH2 --> FETCHER
    GH3 --> FETCHER
    CURSOR --> FETCHER
    
    FETCHER --> CACHE
    CACHE <--> STORAGE
    SCHEDULER --> FETCHER
    
    CACHE --> API
    CACHE --> VIEW
    
    CLIENT --> API
    BROWSER --> VIEW
```

## Components and Interfaces

### 1. ReleaseNote 数据结构

```go
// ReleaseNote 表示单个版本的发布说明
type ReleaseNote struct {
    Version     string    `json:"version"`      // 版本号
    ReleaseDate time.Time `json:"release_date"` // 发布日期
    Changelog   string    `json:"changelog"`    // 变更内容（Markdown格式）
    URL         string    `json:"url"`          // 发布页面链接
}

// CLIReleaseNotes 表示某个CLI工具的所有发布说明
type CLIReleaseNotes struct {
    CLIName       string        `json:"cli_name"`       // CLI名称
    DisplayName   string        `json:"display_name"`   // 显示名称
    LatestVersion string        `json:"latest_version"` // 最新版本
    LocalVersion  string        `json:"local_version"`  // 本地安装版本
    UpdateAvailable bool        `json:"update_available"` // 是否有更新
    LastUpdated   time.Time     `json:"last_updated"`   // 最后更新时间
    Releases      []ReleaseNote `json:"releases"`       // 发布历史
}

// AllReleaseNotes 表示所有CLI工具的发布说明
type AllReleaseNotes struct {
    CLIs        map[string]*CLIReleaseNotes `json:"clis"`
    LastUpdated time.Time                   `json:"last_updated"`
}
```

### 2. ReleaseNoteFetcher 接口

```go
// ReleaseNoteFetcher 定义获取release notes的接口
type ReleaseNoteFetcher interface {
    // CLIName 返回CLI工具名称
    CLIName() string
    
    // DisplayName 返回显示名称
    DisplayName() string
    
    // Fetch 从外部源获取release notes
    Fetch(ctx context.Context) (*CLIReleaseNotes, error)
    
    // GetLocalVersion 获取本地安装的版本
    GetLocalVersion() (string, error)
}
```

### 3. 各CLI Fetcher实现

| CLI | Fetcher | 数据源 | 解析方式 |
|-----|---------|--------|----------|
| Claude | ClaudeFetcher | NPM Registry API | JSON解析版本列表 |
| Codex | CodexFetcher | GitHub Releases API | JSON解析releases |
| Cursor | CursorFetcher | cursor.com/changelog | HTML解析嵌入JSON |
| Gemini | GeminiFetcher | GitHub Releases API | JSON解析releases |
| Qwen | QwenFetcher | GitHub Releases API | JSON解析releases |

### 4. ReleaseNotesService

```go
// ReleaseNotesService 管理release notes的获取、缓存和定时刷新
type ReleaseNotesService struct {
    fetchers    map[string]ReleaseNoteFetcher
    cache       *AllReleaseNotes
    cacheMutex  sync.RWMutex
    storagePath string
    refreshInterval time.Duration
}

// Methods:
// - Start() - 启动服务，加载缓存，开始定时刷新
// - Stop() - 停止服务
// - GetAll() - 获取所有CLI的release notes
// - GetByCLI(name string) - 获取指定CLI的release notes
// - Refresh(force bool) - 刷新release notes
// - SaveToStorage() - 保存到文件
// - LoadFromStorage() - 从文件加载
```

### 5. HTTP Handlers

```go
// GET /release-notes
// 返回所有CLI的release notes
func handleGetAllReleaseNotes(w http.ResponseWriter, r *http.Request)

// GET /release-notes/{cli_name}
// 返回指定CLI的release notes
// Query params: include_local=true, force_refresh=true
func handleGetCLIReleaseNotes(w http.ResponseWriter, r *http.Request)

// GET /release-notes/view
// 返回HTML可视化页面
func handleReleaseNotesView(w http.ResponseWriter, r *http.Request)
```

## Data Models

### API Response Schema

```json
{
  "cli_name": "claude",
  "display_name": "Claude CLI",
  "latest_version": "2.0.53",
  "local_version": "2.0.30",
  "update_available": true,
  "last_updated": "2025-11-25T10:00:00Z",
  "releases": [
    {
      "version": "2.0.53",
      "release_date": "2025-11-25T01:12:33Z",
      "changelog": "- Bug fixes\n- Performance improvements",
      "url": "https://github.com/anthropics/claude-code/releases/tag/v2.0.53"
    }
  ]
}
```

### Cache Storage Schema (release_notes.json)

```json
{
  "clis": {
    "claude": { ... },
    "codex": { ... },
    "cursor": { ... },
    "gemini": { ... },
    "qwen": { ... }
  },
  "last_updated": "2025-11-25T10:00:00Z"
}
```

## Data Persistence

### 持久化策略

系统采用**双层缓存**架构：
1. **内存缓存** - 快速访问，服务运行期间有效
2. **文件持久化** - 服务重启后恢复数据

### 文件存储

```
data/
└── release_notes.json    # 持久化存储文件
```

### 持久化时机

| 事件 | 操作 |
|------|------|
| 服务启动 | 从 `release_notes.json` 加载到内存缓存 |
| 定时刷新成功 | 更新内存缓存，同步写入文件 |
| 强制刷新成功 | 更新内存缓存，同步写入文件 |
| 服务关闭 | 将内存缓存写入文件（graceful shutdown） |

### 文件操作

```go
// 持久化配置
type PersistenceConfig struct {
    StoragePath     string        // 存储文件路径，默认 "data/release_notes.json"
    WriteOnUpdate   bool          // 每次更新后写入文件，默认 true
    BackupEnabled   bool          // 是否保留备份，默认 true
    BackupCount     int           // 保留备份数量，默认 3
}

// 文件操作方法
func (s *ReleaseNotesService) SaveToStorage() error
func (s *ReleaseNotesService) LoadFromStorage() error
func (s *ReleaseNotesService) CreateBackup() error
```

### 数据完整性

1. **原子写入** - 先写入临时文件，成功后重命名
2. **备份机制** - 写入前备份旧文件（release_notes.json.bak.1, .bak.2, .bak.3）
3. **校验** - 加载时验证JSON格式和必要字段

### 故障恢复

| 场景 | 恢复策略 |
|------|----------|
| 文件不存在 | 立即从外部源获取数据 |
| 文件损坏 | 尝试加载最近的备份文件 |
| 备份也损坏 | 从外部源重新获取 |
| 外部源不可用 | 使用损坏前的内存缓存（如有） |



## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. 
Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

Based on the prework analysis, the following properties have been identified for property-based testing:

### Property 1: API returns valid JSON with required fields
*For any* GET request to `/release-notes` or `/release-notes/{cli_name}`, the response SHALL be valid JSON containing version, release_date, and changelog fields for each release.
**Validates: Requirements 1.1, 1.2, 1.4**

### Property 2: Invalid CLI name returns 400 error
*For any* string that is not a valid CLI name (claude, codex, cursor, gemini, qwen), a GET request to `/release-notes/{invalid_name}` SHALL return a 400 status code.
**Validates: Requirements 1.3**

### Property 3: Unavailable source returns 503 error
*For any* CLI fetcher, when the external source is unavailable, the system SHALL return a 503 status code with an error message.
**Validates: Requirements 2.6**

### Property 4: Cache prevents redundant fetches
*For any* sequence of requests within the cache TTL, only the first request SHALL trigger an external fetch; subsequent requests SHALL use cached data.
**Validates: Requirements 3.1, 3.2**

### Property 5: Cache expiration triggers refresh
*For any* cached data older than the configured TTL, a new request SHALL trigger a fresh fetch from external sources.
**Validates: Requirements 3.3**

### Property 6: Force refresh bypasses cache
*For any* request with `force_refresh=true`, the system SHALL fetch from external sources regardless of cache state.
**Validates: Requirements 3.4**

### Property 7: Version comparison correctness
*For any* two version strings where local_version differs from latest_version, the update_available field SHALL be true.
**Validates: Requirements 4.1, 4.2**

### Property 8: Consistent JSON schema across CLIs
*For any* CLI tool, the response JSON SHALL have the same structure with fields: cli_name, display_name, latest_version, local_version, update_available, last_updated, releases.
**Validates: Requirements 5.1**

### Property 9: JSON serialization round-trip
*For any* valid CLIReleaseNotes struct, serializing to JSON and deserializing back SHALL produce an equivalent struct.
**Validates: Requirements 5.4, 5.5**

### Property 10: Releases sorted by date descending
*For any* list of releases returned by the API, the releases SHALL be sorted in reverse chronological order (newest first).
**Validates: Requirements 7.1**

### Property 11: Scheduled refresh executes at interval
*For any* configured refresh interval, the system SHALL automatically fetch new data after the interval passes.
**Validates: Requirements 8.2**

### Property 12: Failed refresh preserves cache
*For any* automatic refresh that fails, the system SHALL retain the previously cached data without modification.
**Validates: Requirements 8.3**

### Property 13: Persistent storage round-trip
*For any* cached release notes, saving to file and loading back SHALL produce equivalent data.
**Validates: Requirements 8.4**

## Error Handling

### External Source Errors

| Error Type | Handling Strategy |
|------------|-------------------|
| Network timeout | Retry up to 3 times with exponential backoff, then return cached data or 503 |
| HTTP 4xx | Log error, return cached data if available, otherwise 503 |
| HTTP 5xx | Retry once, then return cached data or 503 |
| Parse error | Log error with response body, return cached data or 503 |
| Rate limiting | Respect rate limits, use cached data, schedule retry |

### Internal Errors

| Error Type | Handling Strategy |
|------------|-------------------|
| Cache read error | Log error, fetch from external source |
| Cache write error | Log error, continue with in-memory cache |
| File storage error | Log error, continue with in-memory cache only |
| Invalid CLI name | Return 400 with list of valid CLI names |

### Error Response Format

```json
{
  "error": "Service unavailable",
  "message": "Failed to fetch release notes for claude: connection timeout",
  "supported_clis": ["claude", "codex", "cursor", "gemini", "qwen"]
}
```

## Testing Strategy

### Dual Testing Approach

This feature requires both unit tests and property-based tests:

1. **Unit Tests**: Verify specific examples, edge cases, and integration points
2. **Property-Based Tests**: Verify universal properties that should hold across all inputs

### Property-Based Testing Framework

We will use **gopter** (https://github.com/leanovate/gopter) for property-based testing in Go.

Each property-based test will:
- Run a minimum of 100 iterations
- Be tagged with a comment referencing the correctness property
- Use format: `**Feature: cli-release-notes, Property {number}: {property_text}**`

### Test Categories

#### 1. Fetcher Tests
- Test each fetcher (Claude, Codex, Cursor, Gemini, Qwen) with mocked HTTP responses
- Verify correct parsing of different response formats
- Test error handling for malformed responses

#### 2. Cache Tests
- Property test: Cache hit/miss behavior
- Property test: Cache expiration
- Property test: Force refresh
- Unit test: Concurrent access safety

#### 3. API Handler Tests
- Property test: Valid JSON response schema
- Property test: Invalid CLI name handling
- Unit test: Query parameter parsing
- Unit test: HTTP status codes

#### 4. Serialization Tests
- Property test: JSON round-trip (serialize/deserialize)
- Property test: File storage round-trip
- Unit test: Edge cases (empty changelog, special characters)

#### 5. Scheduler Tests
- Property test: Refresh interval execution
- Property test: Error recovery
- Unit test: Startup behavior

#### 6. Integration Tests
- End-to-end test with real external sources (optional, manual)
- HTML view rendering test

### Test File Structure

```
release_notes_test.go       # Main test file
├── TestFetcher_*           # Fetcher unit tests
├── TestCache_*             # Cache unit tests
├── TestHandler_*           # Handler unit tests
├── TestProperty_*          # Property-based tests
└── TestIntegration_*       # Integration tests
```

