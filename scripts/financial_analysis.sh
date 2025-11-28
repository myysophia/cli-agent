#!/bin/bash

# 金融市场定时分析脚本
# 用于 crontab 定时执行，通过 HTTP 接口调用 AI 进行黄金和 A股 LED 板块分析
# 作者: Auto-generated
# 日期: 2025-11-28

set -e

# ==================== 配置区 ====================

# 脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
LOG_DIR="${PROJECT_ROOT}/logs"
LOG_FILE="${LOG_DIR}/financial_analysis_$(date +%Y%m%d).log"

# Prompts 配置文件
PROMPTS_FILE="${SCRIPT_DIR}/prompts.json"

# cli-agent HTTP 服务地址
API_URL="${API_URL:-http://localhost:8081/chat}"

# 使用的 profile（从 configs.json 中选择）
PROFILE="${PROFILE:-cursor}"

# 请求超时时间（秒）
TIMEOUT="${TIMEOUT:-300}"

# ==================== 函数定义 ====================

# 日志函数
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

# 错误处理
error_exit() {
    log "ERROR: $1"
    exit 1
}

# 调用 HTTP 接口进行分析
call_api() {
    local prompt="$1"
    local task_name="$2"
    
    log "开始执行任务: $task_name"
    log "API 地址: $API_URL"
    log "使用 Profile: $PROFILE"
    
    # 转义 prompt 中的特殊字符
    local escaped_prompt=$(echo "$prompt" | jq -Rs .)
    
    # 构建请求 JSON
    local request_json=$(cat <<EOF
{
    "message": $escaped_prompt,
    "profile": "$PROFILE"
}
EOF
)
    
    # 调用 HTTP 接口
    local response_file="${LOG_DIR}/${task_name}_$(date +%Y%m%d_%H%M%S).json"
    
    log "发送请求..."
    http_code=$(curl -s -w "%{http_code}" -o "$response_file" \
        --max-time "$TIMEOUT" \
        -X POST "$API_URL" \
        -H "Content-Type: application/json" \
        -d "$request_json")
    
    if [ "$http_code" != "200" ]; then
        log "HTTP 状态码: $http_code"
        log "响应内容: $(cat "$response_file")"
        error_exit "API 调用失败: $task_name"
    fi
    
    log "任务完成: $task_name"
    log "响应已保存到: $response_file"
    
    # 输出响应内容到日志
    if command -v jq >/dev/null 2>&1; then
        cat "$response_file" | jq -r '.response // .message // .content // .' >> "$LOG_FILE" 2>&1 || cat "$response_file" >> "$LOG_FILE"
    else
        cat "$response_file" >> "$LOG_FILE"
    fi
}

# 从配置文件读取任务信息
get_task_info() {
    local task_id="$1"
    local field="$2"
    
    if [ ! -f "$PROMPTS_FILE" ]; then
        error_exit "配置文件不存在: $PROMPTS_FILE"
    fi
    
    local value=$(jq -r ".tasks.${task_id}.${field} // empty" "$PROMPTS_FILE")
    
    if [ -z "$value" ]; then
        error_exit "任务 '${task_id}' 的字段 '${field}' 不存在"
    fi
    
    echo "$value"
}

# 执行单个分析任务
run_task() {
    local task_id="$1"
    
    # 检查任务是否存在
    if ! jq -e ".tasks.${task_id}" "$PROMPTS_FILE" >/dev/null 2>&1; then
        error_exit "任务 '${task_id}' 不存在，请检查 prompts.json"
    fi
    
    local task_name=$(get_task_info "$task_id" "name")
    local prompt=$(get_task_info "$task_id" "prompt")
    
    log "========== 开始 ${task_name} =========="
    
    call_api "$prompt" "${task_id}_analysis"
    
    log "${task_name} 完成"
}

# 执行任务组
run_group() {
    local group_name="$1"
    
    # 检查组是否存在
    if ! jq -e ".groups.${group_name}" "$PROMPTS_FILE" >/dev/null 2>&1; then
        error_exit "任务组 '${group_name}' 不存在，请检查 prompts.json"
    fi
    
    # 获取组中的任务列表
    local tasks=$(jq -r ".groups.${group_name}[]" "$PROMPTS_FILE")
    
    log "执行任务组: ${group_name}"
    
    for task_id in $tasks; do
        run_task "$task_id"
        echo ""
    done
}

# 列出所有可用的任务和任务组
list_tasks() {
    echo "=========================================="
    echo "可用的分析任务："
    echo "=========================================="
    echo ""
    
    jq -r '.tasks | to_entries[] | "  \(.key) - \(.value.name)"' "$PROMPTS_FILE"
    
    echo ""
    echo "=========================================="
    echo "可用的任务组："
    echo "=========================================="
    echo ""
    
    jq -r '.groups | to_entries[] | "  \(.key) - [\(.value | join(", "))]"' "$PROMPTS_FILE"
    
    echo ""
}

# ==================== 主程序 ====================

main() {
    # 处理 list 命令
    if [ "$1" = "list" ]; then
        list_tasks
        exit 0
    fi
    
    log "========== 金融分析任务开始 =========="
    
    # 创建日志目录
    mkdir -p "$LOG_DIR"
    
    # 检查必要的命令
    command -v curl >/dev/null 2>&1 || error_exit "curl 未安装"
    command -v jq >/dev/null 2>&1 || error_exit "jq 未安装，请先安装: brew install jq"
    
    # 检查配置文件
    if [ ! -f "$PROMPTS_FILE" ]; then
        error_exit "配置文件不存在: $PROMPTS_FILE"
    fi
    
    # 测试 API 连接
    log "测试 API 连接..."
    if ! curl -s --max-time 5 -o /dev/null -w "%{http_code}" "$API_URL" | grep -q "200\|405"; then
        log "WARNING: API 连接测试失败，但继续执行"
    fi
    
    log "API 地址: $API_URL"
    log "使用 Profile: $PROFILE"
    
    # 获取任务参数
    local task_arg="${1:-all}"
    
    # 检查是否为任务组
    if jq -e ".groups.${task_arg}" "$PROMPTS_FILE" >/dev/null 2>&1; then
        run_group "$task_arg"
    # 检查是否为单个任务
    elif jq -e ".tasks.${task_arg}" "$PROMPTS_FILE" >/dev/null 2>&1; then
        run_task "$task_arg"
    else
        log "ERROR: 未知的任务或任务组: $task_arg"
        echo ""
        list_tasks
        exit 1
    fi
    
    log "========== 金融分析任务完成 =========="
}

# 执行主程序
main "$@"
