#!/bin/bash

# 测试 MCP Notify 配置
# 使用企业微信机器人发送测试消息

BOT_KEY="${WEWORK_BOT_KEY:-d8a5ed6e-a42c-4625-9cae-2519ed87eaf2}"
MESSAGE="${1:-MCP 配置成功}"

# 企业微信机器人 Webhook URL
WEBHOOK_URL="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=${BOT_KEY}"

# 发送文本消息
curl -X POST "${WEBHOOK_URL}" \
  -H 'Content-Type: application/json' \
  -d "{
    \"msgtype\": \"text\",
    \"text\": {
      \"content\": \"${MESSAGE}\"
    }
  }"

echo ""
