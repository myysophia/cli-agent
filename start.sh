#!/bin/bash

# CLI Gateway 启动脚本

set -e

# 解析命令行参数
PORT="${PORT:-8080}"
while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--port)
            PORT="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -p, --port PORT    Set the port number (default: 8080)"
            echo "  -h, --help         Show this help message"
            echo ""
            echo "Environment variables:"
            echo "  PORT               Set the port number (default: 8080)"
            echo ""
            echo "Examples:"
            echo "  $0                 # Start on default port 8080"
            echo "  $0 -p 3000         # Start on port 3000"
            echo "  PORT=9000 $0       # Start on port 9000"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use -h or --help for usage information"
            exit 1
            ;;
    esac
done

echo "🚀 Starting CLI Gateway..."
echo ""

# 检查 Go 是否安装
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is not installed"
    echo "Please install Go from https://golang.org/dl/"
    exit 1
fi

# 检查是否已构建
if [ ! -f "claude-cli-gateway" ]; then
    echo "📦 Building project..."
    go build -o claude-cli-gateway ./cmd/server
    echo "✅ Build completed"
    echo ""
fi

# 启动服务
export PORT="$PORT"
echo "🌐 Starting gateway service on http://localhost:$PORT"
echo "📝 Press Ctrl+C to stop"
echo ""

./claude-cli-gateway
