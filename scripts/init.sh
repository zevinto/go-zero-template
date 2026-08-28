#!/usr/bin/env bash

set -euo pipefail

MODULE="${1:-}"

# 0. 解析模板当前的模块名（动态读取，保证与 go.mod 一致）
if [[ ! -f "go.mod" ]]; then
    echo "Error: go.mod not found."
    echo "Please run this script from the project root."
    exit 1
fi

TEMPLATE_MODULE="$(awk 'NR==1 && $1=="module" {print $2}' go.mod)"
if [[ -z "$TEMPLATE_MODULE" ]]; then
    echo "Error: failed to parse module path from go.mod."
    exit 1
fi

# 1. 检查参数
if [[ -z "$MODULE" ]]; then
    echo "Usage: ./scripts/init.sh <module-path>"
    echo
    echo "Example:"
    echo "  ./scripts/init.sh github.com/your-org/order-service"
    exit 1
fi

if [[ "$MODULE" == "$TEMPLATE_MODULE" ]]; then
    echo "Error: module path is already $MODULE."
    exit 1
fi

echo "Initializing Go project..."
echo "  Module: $TEMPLATE_MODULE -> $MODULE"
echo

# 2. 修改 go.mod
echo "==> Updating go.mod..."
go mod edit -module "$MODULE"

# 3. 替换 Go 源码中的模板 module
echo "==> Updating imports..."

find . \
    -type f \
    -name "*.go" \
    -not -path "./.git/*" \
    -exec sed -i.bak "s|$TEMPLATE_MODULE|$MODULE|g" {} +

find . \
    -type f \
    -name "*.bak" \
    -delete

# 4. 整理依赖
echo "==> Running go mod tidy..."
go mod tidy

echo
echo "Project initialized successfully."
echo
echo "Module:"
echo "  $MODULE"
echo
echo "Next steps:"
echo "  make build"
echo "  make test"
echo "  make run"
