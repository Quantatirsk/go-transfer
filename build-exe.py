#!/usr/bin/env python3

"""
构建脚本：将 transfer-gui.py 打包成 Windows exe 文件
"""

import os
import sys
import subprocess
import shutil

def install_dependencies():
    """安装必要的依赖"""
    print("📦 安装 PyInstaller...")
    subprocess.run([sys.executable, "-m", "pip", "install", "pyinstaller"], check=True)
    print("✅ PyInstaller 安装完成")

def build_exe():
    """构建 exe 文件"""
    print("\n🔨 开始构建 exe 文件...")
    
    # PyInstaller 参数
    args = [
        "pyinstaller",
        "--onefile",                    # 打包成单个文件
        "--windowed",                   # 不显示控制台窗口
        "--name", "GoTransfer",         # 输出文件名
        "--icon", "NONE",               # 图标（如果有的话）
        "--clean",                      # 清理临时文件
        "--noconfirm",                  # 覆盖输出目录
        "--dist", "dist",               # 输出目录
        "--workpath", "build",          # 工作目录
        "transfer-gui.py"               # 源文件
    ]
    
    # 执行打包
    result = subprocess.run(args, capture_output=True, text=True)
    
    if result.returncode == 0:
        print("✅ exe 文件构建成功")
        print(f"📁 输出位置: dist/GoTransfer.exe")
        
        # 显示文件信息
        if os.path.exists("dist/GoTransfer.exe"):
            size = os.path.getsize("dist/GoTransfer.exe") / (1024 * 1024)
            print(f"📊 文件大小: {size:.2f} MB")
    else:
        print("❌ 构建失败")
        print(result.stderr)
        return False
    
    return True

def create_spec_file():
    """创建自定义 spec 文件用于更精细的控制"""
    spec_content = """# -*- mode: python ; coding: utf-8 -*-

block_cipher = None

a = Analysis(
    ['transfer-gui.py'],
    pathex=[],
    binaries=[],
    datas=[],
    hiddenimports=[],
    hookspath=[],
    hooksconfig={},
    runtime_hooks=[],
    excludes=[
        'matplotlib',
        'numpy',
        'pandas',
        'scipy',
        'PIL',
        'pytest',
        'setuptools',
        'pip'
    ],
    win_no_prefer_redirects=False,
    win_private_assemblies=False,
    cipher=block_cipher,
    noarchive=False,
)

pyz = PYZ(a.pure, a.zipped_data, cipher=block_cipher)

exe = EXE(
    pyz,
    a.scripts,
    a.binaries,
    a.zipfiles,
    a.datas,
    [],
    name='GoTransfer',
    debug=False,
    bootloader_ignore_signals=False,
    strip=False,
    upx=True,
    upx_exclude=[],
    runtime_tmpdir=None,
    console=False,          # False = 窗口应用，True = 控制台应用
    disable_windowed_traceback=False,
    target_arch='x86',      # 指定 x86 架构
    codesign_identity=None,
    entitlements_file=None,
)
"""
    
    with open("GoTransfer.spec", "w", encoding="utf-8") as f:
        f.write(spec_content)
    
    print("📝 已创建自定义 spec 文件")

def build_with_spec():
    """使用 spec 文件构建"""
    print("\n🔨 使用 spec 文件构建 exe...")
    
    args = [
        "pyinstaller",
        "--clean",
        "--noconfirm",
        "GoTransfer.spec"
    ]
    
    result = subprocess.run(args, capture_output=True, text=True)
    
    if result.returncode == 0:
        print("✅ exe 文件构建成功")
        return True
    else:
        print("❌ 构建失败")
        print(result.stderr)
        return False

def create_batch_script():
    """创建 Windows 批处理脚本"""
    batch_content = """@echo off
echo ====================================
echo   Go-Transfer GUI 构建工具 (x86)
echo ====================================
echo.

REM 检查 Python
python --version >nul 2>&1
if errorlevel 1 (
    echo ❌ 未找到 Python，请先安装 Python 3
    pause
    exit /b 1
)

REM 安装 PyInstaller
echo 📦 安装 PyInstaller...
pip install pyinstaller

REM 构建 exe
echo.
echo 🔨 开始构建 exe 文件...
pyinstaller --onefile --windowed --name GoTransfer --clean --noconfirm transfer-gui.py

if exist "dist\\GoTransfer.exe" (
    echo.
    echo ✅ 构建成功！
    echo 📁 文件位置: dist\\GoTransfer.exe
    
    REM 显示文件大小
    for %%A in ("dist\\GoTransfer.exe") do echo 📊 文件大小: %%~zA 字节
) else (
    echo.
    echo ❌ 构建失败
)

echo.
pause
"""
    
    with open("build-exe.bat", "w", encoding="utf-8") as f:
        f.write(batch_content)
    
    print("📝 已创建 Windows 批处理脚本: build-exe.bat")

def main():
    print("="*50)
    print("  Go-Transfer GUI 打包工具")
    print("  目标平台: Windows x86")
    print("="*50)
    
    # 检查源文件
    if not os.path.exists("transfer-gui.py"):
        print("❌ 找不到 transfer-gui.py 文件")
        return
    
    # 创建批处理脚本（Windows 用户可以直接运行）
    create_batch_script()
    
    # 创建 spec 文件
    create_spec_file()
    
    # 检查是否在 Windows 环境
    if sys.platform != "win32":
        print("\n⚠️  注意：当前不是 Windows 环境")
        print("要生成真正的 Windows x86 exe 文件，请在 Windows 系统上运行：")
        print("1. 运行 build-exe.bat (双击即可)")
        print("2. 或使用命令: pyinstaller GoTransfer.spec")
        print("\n已生成以下文件供 Windows 使用：")
        print("- build-exe.bat (Windows 批处理脚本)")
        print("- GoTransfer.spec (PyInstaller 配置文件)")
        return
    
    # 在 Windows 上直接构建
    try:
        install_dependencies()
        
        # 使用 spec 文件构建
        if build_with_spec():
            print("\n🎉 打包完成！")
            print("可执行文件位置: dist/GoTransfer.exe")
            print("该文件可以在任何 Windows x86/x64 系统上运行，无需安装 Python")
    
    except Exception as e:
        print(f"❌ 错误: {e}")

if __name__ == "__main__":
    main()