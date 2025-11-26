#!/bin/bash

# 获取行家说今天的新闻并发送到企微

echo "📰 正在获取行家说今天的新闻..."

# 第一步：使用playwright获取行家说今天的新闻列表和摘要
RESPONSE=$(curl -s -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{
    "profile": "codex",
    "prompt": "请访问 https://www.hangjianet.com/news?page=1 获取今天（'$(date +%Y-%m-%d)'）的新闻列表。对于每条新闻，请提取：1. 标题 2. 发布时间 3. 文章摘要（100-150字）。请以Markdown格式整理，格式如下：\n\n## 行家说今日新闻 ('$(date +%Y-%m-%d)')\n\n### 新闻1：标题\n- **时间**：发布时间\n- **摘要**：文章摘要\n\n### 新闻2：标题\n- **时间**：发布时间\n- **摘要**：文章摘要\n\n...",
    "allowed_tools": ["playwright"],
    "permission_mode": "bypassPermissions",
    "new_session": true
  }')

echo "📥 收到响应，正在解析..."

# 提取响应内容
# 先尝试解析answer字段中的JSON
ANSWER_JSON=$(echo "$RESPONSE" | jq -r '.answer' 2>/dev/null)

if [ -n "$ANSWER_JSON" ] && [ "$ANSWER_JSON" != "null" ]; then
  # 尝试从JSON中提取response或codex字段
  NEWS_CONTENT=$(echo "$ANSWER_JSON" | jq -r '.response // .codex // empty' 2>/dev/null)
  
  # 如果还是空，尝试直接解析整个answer作为JSON
  if [ -z "$NEWS_CONTENT" ] || [ "$NEWS_CONTENT" = "null" ]; then
    # 尝试将answer作为JSON字符串解析
    NEWS_CONTENT=$(echo "$ANSWER_JSON" | jq -r 'if type == "string" then . else tostring end' 2>/dev/null)
  fi
fi

# 如果还是空，尝试直接使用answer字段
if [ -z "$NEWS_CONTENT" ] || [ "$NEWS_CONTENT" = "null" ]; then
  NEWS_CONTENT="$ANSWER_JSON"
fi

if [ -z "$NEWS_CONTENT" ] || [ "$NEWS_CONTENT" = "null" ]; then
  echo "❌ 无法获取新闻内容"
  echo "响应: $RESPONSE"
  exit 1
fi

echo "✅ 成功获取新闻内容"
echo ""
echo "📋 新闻摘要："
echo "$NEWS_CONTENT" | head -50
echo ""
echo ""

# 第二步：发送到企微
echo "📤 正在发送到企微..."

BOT_KEY="${WEWORK_BOT_KEY:-d8a5ed6e-a42c-4625-9cae-2519ed87eaf2}"
WEBHOOK_URL="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=${BOT_KEY}"

# 清理新闻内容，移除JSON转义字符
CLEAN_CONTENT=$(echo "$NEWS_CONTENT" | sed 's/\\n/\n/g' | sed 's/\\"/"/g' | sed 's/\\t/ /g')

# 准备消息内容（企微markdown格式）
# 企微markdown支持有限，使用简单的格式
MARKDOWN_CONTENT="## 📰 行家说今日新闻 ($(date +%Y-%m-%d))\n\n${CLEAN_CONTENT}"

# 转义JSON特殊字符（兼容macOS和Linux）
ESCAPED_CONTENT=$(echo "$MARKDOWN_CONTENT" | sed 's/\\/\\\\/g' | sed 's/"/\\"/g' | awk '{printf "%s\\n", $0}' | sed 's/\\n$//')

MARKDOWN_MSG=$(cat <<EOF
{
  "msgtype": "markdown",
  "markdown": {
    "content": "${ESCAPED_CONTENT}"
  }
}
EOF
)

# 发送到企微
RESULT=$(curl -s -X POST "${WEBHOOK_URL}" \
  -H 'Content-Type: application/json' \
  -d "${MARKDOWN_MSG}")

if echo "$RESULT" | grep -q '"errcode":0'; then
  echo "✅ 消息已成功发送到企微"
else
  echo "⚠️  发送可能失败，响应：$RESULT"
  echo ""
  echo "尝试使用文本格式发送..."
  
  # 如果markdown失败，尝试文本格式
  TEXT_CONTENT="📰 行家说今日新闻 ($(date +%Y-%m-%d))\n\n${CLEAN_CONTENT}"
  ESCAPED_TEXT=$(echo "$TEXT_CONTENT" | sed 's/\\/\\\\/g' | sed 's/"/\\"/g' | awk '{printf "%s\\n", $0}' | sed 's/\\n$//')
  
  TEXT_MSG=$(cat <<EOF
{
  "msgtype": "text",
  "text": {
    "content": "${ESCAPED_TEXT}"
  }
}
EOF
)
  
  curl -X POST "${WEBHOOK_URL}" \
    -H 'Content-Type: application/json' \
    -d "${TEXT_MSG}"
  
  echo ""
  echo "✅ 已使用文本格式发送"
fi
