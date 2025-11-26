# Requirements Document

## Introduction

本功能为 CLI Gateway 添加获取各类 CLI 工具（Claude、Codex、Cursor、Gemini、Qwen）最新 Release Note 变更内容的能力。用户可以通过 HTTP 接口查询各 CLI 工具的版本更新信息，并通过可视化的 HTML 页面交互查看每个 CLI 的变更历史。

## Glossary

- **CLI Gateway**: 本项目的 HTTP 网关服务，将 HTTP 请求桥接到各种 CLI 工具
- **Release Note**: CLI 工具的版本发布说明，包含新功能、修复、变更等信息
- **CLI Tool**: 命令行工具，包括 Claude CLI、Codex CLI、Cursor Agent CLI、Gemini CLI、Qwen CLI
- **Version**: CLI 工具的版本号
- **Changelog**: 版本变更日志，记录各版本的更新内容
- **Release Notes Viewer**: 用于展示 CLI 变更历史的 HTML 可视化界面

## Requirements

### Requirement 1

**User Story:** As a developer, I want to query the latest release notes for CLI tools via API, so that I can stay informed about new features and changes.

#### Acceptance Criteria

1. WHEN a user sends a GET request to `/release-notes` THEN the CLI Gateway SHALL return the latest release notes for all supported CLI tools in JSON format
2. WHEN a user sends a GET request to `/release-notes/{cli_name}` THEN the CLI Gateway SHALL return the latest release notes for the specified CLI tool in JSON format
3. WHEN the specified CLI tool name is invalid THEN the CLI Gateway SHALL return a 400 error with a list of supported CLI tools
4. WHEN release notes are successfully retrieved THEN the CLI Gateway SHALL return JSON containing version, release date, and changelog content

### Requirement 2

**User Story:** As a developer, I want to get release notes from multiple sources, so that I can access the most accurate and up-to-date information.

#### Acceptance Criteria

1. WHEN fetching Claude CLI release notes THEN the CLI Gateway SHALL retrieve version information from the NPM registry API (https://registry.npmjs.org/@anthropic-ai/claude-code)
2. WHEN fetching Codex CLI release notes THEN the CLI Gateway SHALL retrieve information from the GitHub releases API (https://api.github.com/repos/openai/codex/releases)
3. WHEN fetching Cursor Agent CLI release notes THEN the CLI Gateway SHALL retrieve information from the Cursor changelog page (https://www.cursor.com/changelog) by parsing the embedded JSON data
4. WHEN fetching Gemini CLI release notes THEN the CLI Gateway SHALL retrieve information from the GitHub releases API (https://api.github.com/repos/google-gemini/gemini-cli/releases)
5. WHEN fetching Qwen CLI release notes THEN the CLI Gateway SHALL retrieve information from the GitHub releases API (https://api.github.com/repos/QwenLM/qwen-code/releases)
6. WHEN the release notes source is unavailable THEN the CLI Gateway SHALL return a 503 error with an appropriate message

### Requirement 3

**User Story:** As a developer, I want release notes to be cached, so that repeated requests are fast and do not overload external sources.

#### Acceptance Criteria

1. WHEN release notes are fetched from external sources THEN the CLI Gateway SHALL cache the results for a configurable duration (default 1 hour)
2. WHEN cached release notes exist and are not expired THEN the CLI Gateway SHALL return the cached data without fetching from external sources
3. WHEN the cache expires THEN the CLI Gateway SHALL fetch fresh release notes from external sources
4. WHEN a user requests release notes with a `force_refresh=true` parameter THEN the CLI Gateway SHALL bypass the cache and fetch fresh data

### Requirement 4

**User Story:** As a developer, I want to compare my local CLI version with the latest version, so that I can know if updates are available.

#### Acceptance Criteria

1. WHEN a user sends a GET request to `/release-notes/{cli_name}` with `include_local=true` THEN the CLI Gateway SHALL include the locally installed CLI version in the response
2. WHEN the local CLI version differs from the latest version THEN the CLI Gateway SHALL indicate that an update is available
3. WHEN the local CLI is not installed THEN the CLI Gateway SHALL indicate that the CLI is not installed locally

### Requirement 5

**User Story:** As a developer, I want release notes to be formatted consistently, so that I can easily parse and display them.

#### Acceptance Criteria

1. WHEN returning release notes THEN the CLI Gateway SHALL use a consistent JSON schema across all CLI tools
2. WHEN parsing release notes from external sources THEN the CLI Gateway SHALL extract version number, release date, and changelog sections
3. WHEN changelog content contains markdown THEN the CLI Gateway SHALL preserve the markdown formatting in the response
4. WHEN serializing release notes to JSON THEN the CLI Gateway SHALL produce valid JSON output
5. WHEN deserializing release notes from JSON cache THEN the CLI Gateway SHALL reconstruct the original release note structure

### Requirement 6

**User Story:** As a developer, I want to view CLI release notes in a visual HTML interface, so that I can easily browse and compare version changes across different CLI tools.

#### Acceptance Criteria

1. WHEN a user visits `/release-notes/view` in a browser THEN the CLI Gateway SHALL serve an HTML page displaying release notes for all CLI tools
2. WHEN the HTML page loads THEN the system SHALL display a tabbed interface with one tab per CLI tool (Claude, Codex, Cursor, Gemini, Qwen)
3. WHEN a user clicks on a CLI tab THEN the system SHALL display the changelog history for that CLI tool
4. WHEN displaying changelog entries THEN the system SHALL show version number, release date, and changelog content in a readable format
5. WHEN changelog content contains markdown THEN the system SHALL render the markdown as formatted HTML
6. WHEN a user clicks a refresh button THEN the system SHALL fetch the latest release notes and update the display

### Requirement 7

**User Story:** As a developer, I want the release notes viewer to support filtering and searching, so that I can quickly find specific version information.

#### Acceptance Criteria

1. WHEN the HTML viewer displays changelog entries THEN the system SHALL show entries in reverse chronological order (newest first)
2. WHEN a user enters text in a search box THEN the system SHALL filter changelog entries to show only those containing the search text
3. WHEN displaying multiple versions THEN the system SHALL provide pagination or infinite scroll for large changelogs
4. WHEN a user clicks on a version entry THEN the system SHALL expand to show the full changelog details

### Requirement 8

**User Story:** As a developer, I want the release notes to be fetched on a schedule, so that the data is always up-to-date when I access it.

#### Acceptance Criteria

1. WHEN the CLI Gateway starts THEN the system SHALL fetch release notes for all CLI tools immediately
2. WHEN a configurable interval passes (default 1 hour) THEN the system SHALL automatically refresh release notes from external sources
3. WHEN automatic refresh fails THEN the system SHALL log the error and retain the previously cached data
4. WHEN the system restarts THEN the system SHALL load cached release notes from persistent storage if available

