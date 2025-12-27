#!/bin/bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ADMIN_UI_DIR="${ROOT_DIR}/web/admin-ui"
OUTPUT_DIR="${ADMIN_UI_DIR}/out"
TARGET_DIR="${ROOT_DIR}/internal/handler/admin_ui_static"

if ! command -v npm >/dev/null 2>&1; then
  echo "❌ Error: npm is not installed"
  exit 1
fi

if [ ! -d "${ADMIN_UI_DIR}" ]; then
  echo "❌ Error: admin UI directory not found: ${ADMIN_UI_DIR}"
  exit 1
fi

if [ ! -d "${ADMIN_UI_DIR}/node_modules" ]; then
  echo "📦 Installing admin UI dependencies..."
  (cd "${ADMIN_UI_DIR}" && npm install)
else
  echo "📦 Using existing admin UI dependencies"
fi

echo "🎨 Building admin UI..."
(cd "${ADMIN_UI_DIR}" && npm run build)

if [ ! -d "${OUTPUT_DIR}" ]; then
  echo "❌ Error: build output not found: ${OUTPUT_DIR}"
  exit 1
fi

echo "📂 Syncing admin UI static files..."
mkdir -p "${TARGET_DIR}"
if command -v rsync >/dev/null 2>&1; then
  rsync -a --delete "${OUTPUT_DIR}/" "${TARGET_DIR}/"
else
  rm -rf "${TARGET_DIR}"
  mkdir -p "${TARGET_DIR}"
  cp -R "${OUTPUT_DIR}/." "${TARGET_DIR}/"
fi

echo "✅ Admin UI build/export completed"
