#!/bin/bash

set -e

echo "🚀 Generating Release Notes HTML..."

# Build the generator
echo "📦 Building generator..."
go build -o generate-html ./cmd/generate-html

# Run the generator
echo "🔄 Fetching release notes..."
./generate-html

# Clean up
rm -f generate-html

echo ""
echo "✅ Done! HTML file generated: release-notes.html"
echo "📄 You can open it in your browser to preview"
