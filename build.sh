#!/bin/bash

# go-transfer 多平台构建脚本

echo "🚀 开始构建 go-transfer 多平台版本"
echo "=================================="

# 清理旧的构建文件
echo "清理旧文件..."
rm -f go-transfer-*

# 定义版本号（可选）
VERSION=$(date +%Y%m%d)
echo "构建版本: $VERSION"
echo ""

# Linux AMD64
echo "📦 构建 Linux AMD64..."
GOOS=linux GOARCH=amd64 go build -o go-transfer-linux-amd64
echo "   ✓ go-transfer-linux-amd64"

# Linux ARM64
echo "📦 构建 Linux ARM64..."
GOOS=linux GOARCH=arm64 go build -o go-transfer-linux-arm64
echo "   ✓ go-transfer-linux-arm64"

# macOS AMD64 (Intel)
echo "📦 构建 macOS AMD64 (Intel)..."
GOOS=darwin GOARCH=amd64 go build -o go-transfer-darwin-amd64
echo "   ✓ go-transfer-darwin-amd64"

# macOS ARM64 (Apple Silicon)
echo "📦 构建 macOS ARM64 (M1/M2/M3)..."
GOOS=darwin GOARCH=arm64 go build -o go-transfer-darwin-arm64
echo "   ✓ go-transfer-darwin-arm64"

# Windows AMD64
echo "📦 构建 Windows AMD64..."
GOOS=windows GOARCH=amd64 go build -o go-transfer-windows-amd64.exe
echo "   ✓ go-transfer-windows-amd64.exe"

# Windows ARM64
echo "📦 构建 Windows ARM64..."
GOOS=windows GOARCH=arm64 go build -o go-transfer-windows-arm64.exe
echo "   ✓ go-transfer-windows-arm64.exe"

echo ""
echo "=================================="
echo "✅ 构建完成！"
echo ""
echo "文件列表："
ls -lah go-transfer-*

echo ""
echo "文件说明："
echo "  • Linux 服务器 (x64):        go-transfer-linux-amd64"
echo "  • Linux 服务器 (ARM):        go-transfer-linux-arm64"
echo "  • macOS Intel:               go-transfer-darwin-amd64"
echo "  • macOS Apple Silicon:       go-transfer-darwin-arm64"
echo "  • Windows (x64):             go-transfer-windows-amd64.exe"
echo "  • Windows (ARM):             go-transfer-windows-arm64.exe"