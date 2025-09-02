@echo off
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

if exist "dist\GoTransfer.exe" (
    echo.
    echo ✅ 构建成功！
    echo 📁 文件位置: dist\GoTransfer.exe
    
    REM 显示文件大小
    for %%A in ("dist\GoTransfer.exe") do echo 📊 文件大小: %%~zA 字节
) else (
    echo.
    echo ❌ 构建失败
)

echo.
pause
