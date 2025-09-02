#!/usr/bin/env python3

"""
go-transfer 优雅的文件传输客户端
支持实时进度显示，无需额外依赖
"""

import sys
import os
import time
import urllib.request
import urllib.parse
from pathlib import Path

class ProgressUploader:
    def __init__(self, filepath, server_url):
        self.filepath = Path(filepath)
        self.server_url = server_url
        self.filesize = self.filepath.stat().st_size
        self.filename = self.filepath.name
        self.uploaded = 0
        self.start_time = None
        
    def format_size(self, bytes):
        """格式化文件大小"""
        for unit in ['B', 'KB', 'MB', 'GB']:
            if bytes < 1024.0:
                return f"{bytes:.2f} {unit}"
            bytes /= 1024.0
        return f"{bytes:.2f} TB"
    
    def print_progress(self):
        """打印进度条"""
        if self.filesize == 0:
            return
            
        percentage = (self.uploaded / self.filesize) * 100
        elapsed = time.time() - self.start_time if self.start_time else 0
        
        if elapsed > 0:
            speed = self.uploaded / elapsed
            eta = (self.filesize - self.uploaded) / speed if speed > 0 else 0
        else:
            speed = 0
            eta = 0
        
        # 构建进度条
        bar_length = 40
        filled = int(bar_length * self.uploaded / self.filesize)
        bar = '█' * filled + '░' * (bar_length - filled)
        
        # 清除当前行并打印进度
        sys.stdout.write('\r')
        sys.stdout.write(f'上传进度: [{bar}] {percentage:.1f}% ')
        sys.stdout.write(f'({self.format_size(self.uploaded)}/{self.format_size(self.filesize)}) ')
        sys.stdout.write(f'速度: {self.format_size(speed)}/s ')
        if eta > 0:
            sys.stdout.write(f'剩余: {int(eta)}秒')
        sys.stdout.flush()
    
    def upload(self):
        """执行上传"""
        print(f"\n{'='*50}")
        print(f"📁 文件: {self.filename}")
        print(f"📊 大小: {self.format_size(self.filesize)}")
        print(f"🎯 目标: {self.server_url}")
        print(f"{'='*50}\n")
        
        # 构建上传URL (支持代理路径)
        if self.server_url.endswith('/'):
            self.server_url = self.server_url[:-1]
        
        # 如果URL已经包含 /sender 或其他路径，直接添加 /upload
        # 否则添加 /upload
        if '/sender' in self.server_url or self.server_url.count('/') > 3:
            url = f"{self.server_url}/upload?name={urllib.parse.quote(self.filename)}"
        else:
            url = f"{self.server_url}/upload?name={urllib.parse.quote(self.filename)}"
        
        # 创建请求
        req = urllib.request.Request(url, method='POST')
        req.add_header('Content-Type', 'application/octet-stream')
        req.add_header('Content-Length', str(self.filesize))
        
        # 开始上传
        self.start_time = time.time()
        
        with open(self.filepath, 'rb') as f:
            # 包装文件对象以跟踪进度
            class ProgressFileWrapper:
                def __init__(self, file, callback):
                    self.file = file
                    self.callback = callback
                    
                def read(self, size=-1):
                    data = self.file.read(size)
                    if data:
                        self.callback(len(data))
                    return data
            
            def update_progress(bytes_read):
                self.uploaded += bytes_read
                self.print_progress()
            
            wrapped_file = ProgressFileWrapper(f, update_progress)
            req.data = wrapped_file
            
            try:
                print("⏳ 开始上传...\n")
                response = urllib.request.urlopen(req)
                result = response.read().decode('utf-8')
                
                # 完成
                elapsed = time.time() - self.start_time
                avg_speed = self.filesize / elapsed if elapsed > 0 else 0
                
                print(f"\n\n✅ 上传成功！")
                print(f"   总耗时: {elapsed:.1f}秒")
                print(f"   平均速度: {self.format_size(avg_speed)}/s")
                print(f"\n服务器响应: {result}")
                
            except Exception as e:
                print(f"\n\n❌ 上传失败: {e}")
                sys.exit(1)

def main():
    if len(sys.argv) != 3:
        print("go-transfer Python客户端")
        print("\n使用方法: python transfer.py <文件> <服务器地址>")
        print("示例: python transfer.py testfile http://10.193.44.211:5000/sender")
        sys.exit(1)
    
    filepath = sys.argv[1]
    server = sys.argv[2]
    
    if not os.path.exists(filepath):
        print(f"❌ 文件不存在: {filepath}")
        sys.exit(1)
    
    uploader = ProgressUploader(filepath, server)
    uploader.upload()

if __name__ == "__main__":
    main()