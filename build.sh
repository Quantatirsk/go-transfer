#!/bin/bash

# gt (go-transfer) 多平台构建脚本

echo "🚀 开始构建 gt (go-transfer) 多平台版本"
echo "=================================="

# 创建 dist 目录
echo "创建输出目录..."
mkdir -p dist

# 清理旧的构建文件
echo "清理旧文件..."
rm -f dist/gt-*
rm -f dist/go-transfer-*  # 也清理旧名称的文件

# 定义版本号（可选）
VERSION=$(date +%Y%m%d)
echo "构建版本: $VERSION"
echo "输出目录: dist/"
echo ""

# Linux AMD64
echo "📦 构建 Linux AMD64..."
GOOS=linux GOARCH=amd64 go build -o dist/gt-linux-amd64
echo "   ✓ dist/gt-linux-amd64"

# Linux ARM64
echo "📦 构建 Linux ARM64..."
GOOS=linux GOARCH=arm64 go build -o dist/gt-linux-arm64
echo "   ✓ dist/gt-linux-arm64"

# macOS AMD64 (Intel)
echo "📦 构建 macOS AMD64 (Intel)..."
GOOS=darwin GOARCH=amd64 go build -o dist/gt-darwin-amd64
echo "   ✓ dist/gt-darwin-amd64"

# macOS ARM64 (Apple Silicon)
echo "📦 构建 macOS ARM64 (M1/M2/M3)..."
GOOS=darwin GOARCH=arm64 go build -o dist/gt-darwin-arm64
echo "   ✓ dist/gt-darwin-arm64"

# Windows AMD64
echo "📦 构建 Windows AMD64..."
GOOS=windows GOARCH=amd64 go build -o dist/gt-windows-amd64.exe
echo "   ✓ dist/gt-windows-amd64.exe"

# Windows ARM64
echo "📦 构建 Windows ARM64..."
GOOS=windows GOARCH=arm64 go build -o dist/gt-windows-arm64.exe
echo "   ✓ dist/gt-windows-arm64.exe"

echo ""
echo "=================================="
echo "✅ 构建完成！"
echo ""
echo "文件列表："
ls -lah dist/gt-*

echo ""
echo "文件说明："
echo "  • Linux 服务器 (x64):        dist/gt-linux-amd64"
echo "  • Linux 服务器 (ARM):        dist/gt-linux-arm64"
echo "  • macOS Intel:               dist/gt-darwin-amd64"
echo "  • macOS Apple Silicon:       dist/gt-darwin-arm64"
echo "  • Windows (x64):             dist/gt-windows-amd64.exe"
echo "  • Windows (ARM):             dist/gt-windows-arm64.exe"