#!/bin/bash

# Claude CLI Gateway 启动脚本

set -e

echo "🚀 Starting Claude CLI Gateway..."
echo ""

# 检查 Go 是否安装
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is not installed"
    echo "Please install Go from https://golang.org/dl/"
    exit 1
fi

# 检查 Claude CLI 是否安装
if ! command -v claude &> /dev/null; then
    echo "❌ Error: Claude CLI is not installed"
    echo "Please install Claude CLI and ensure it's in your PATH"
    exit 1
fi

# 检查是否已构建
if [ ! -f "claude-cli-gateway" ]; then
    echo "📦 Building project..."
    go build -o claude-cli-gateway
    echo "✅ Build completed"
    echo ""
fi

# 启动服务
echo "🌐 Starting gateway service on http://localhost:8080"
echo "📝 Press Ctrl+C to stop"
echo ""

./claude-cli-gateway
