#!/usr/bin/env python3

"""
go-transfer GUI客户端
基于Tkinter的文件传输工具，支持文件和文件夹上传，实时进度显示
"""

import tkinter as tk
from tkinter import ttk, filedialog, messagebox
import threading
import time
import urllib.request
import urllib.parse
from pathlib import Path
import json

class TransferGUI:
    def __init__(self, root):
        self.root = root
        self.root.title("Go-Transfer 传输助手")
        self.root.geometry("800x600")
        
        # 设置样式
        self.setup_styles()
        
        # 文件列表
        self.files = []
        self.is_uploading = False
        self.current_upload_index = 0
        
        # 加载配置
        self.load_config()
        
        # 创建界面
        self.create_widgets()
        
        # 窗口关闭时保存配置
        self.root.protocol("WM_DELETE_WINDOW", self.on_closing)
        
    def setup_styles(self):
        """设置界面样式"""
        style = ttk.Style()
        style.theme_use('clam')
        
        # 配置颜色
        bg_color = '#f5f5f5'
        fg_color = '#333333'
        accent_color = '#2563eb'
        
        self.root.configure(bg=bg_color)
        
        # 配置按钮样式
        style.configure('Primary.TButton',
                       background=accent_color,
                       foreground='white',
                       borderwidth=0,
                       focuscolor='none',
                       font=('Arial', 10))
        style.map('Primary.TButton',
                 background=[('active', '#1d4ed8')])
        
        style.configure('Secondary.TButton',
                       background='#e5e7eb',
                       foreground=fg_color,
                       borderwidth=0,
                       focuscolor='none',
                       font=('Arial', 10))
        style.map('Secondary.TButton',
                 background=[('active', '#d1d5db')])
                 
    def create_widgets(self):
        """创建界面组件"""
        # 主容器
        main_frame = ttk.Frame(self.root, padding="20")
        main_frame.grid(row=0, column=0, sticky="nsew")
        
        # 配置网格权重
        self.root.columnconfigure(0, weight=1)
        self.root.rowconfigure(0, weight=1)
        main_frame.columnconfigure(0, weight=1)
        
        # 标题
        title_label = ttk.Label(main_frame, text="Go-Transfer 传输助手", 
                               font=('Arial', 18, 'bold'))
        title_label.grid(row=0, column=0, columnspan=2, pady=(0, 10))
        
        subtitle_label = ttk.Label(main_frame, text="极简文件传输工具，支持文件和文件夹上传", 
                                  font=('Arial', 10), foreground='gray')
        subtitle_label.grid(row=1, column=0, columnspan=2, pady=(0, 20))
        
        # 服务器地址
        server_frame = ttk.LabelFrame(main_frame, text="服务器配置", padding="10")
        server_frame.grid(row=2, column=0, columnspan=2, sticky="we", pady=(0, 20))
        server_frame.columnconfigure(1, weight=1)
        
        ttk.Label(server_frame, text="服务器地址:").grid(row=0, column=0, sticky=tk.W, padx=(0, 10))
        self.server_entry = ttk.Entry(server_frame, font=('Arial', 10))
        self.server_entry.grid(row=0, column=1, sticky="we")
        self.server_entry.insert(0, self.config.get('server_url', 'http://10.193.44.211:5000/sender'))
        
        # 文件选择按钮
        button_frame = ttk.Frame(main_frame)
        button_frame.grid(row=3, column=0, columnspan=2, pady=(0, 20))
        
        self.select_files_btn = ttk.Button(button_frame, text="📁 选择文件", 
                                          command=self.select_files,
                                          style='Secondary.TButton',
                                          width=20)
        self.select_files_btn.grid(row=0, column=0, padx=5)
        
        self.select_folder_btn = ttk.Button(button_frame, text="📂 选择文件夹", 
                                           command=self.select_folder,
                                           style='Secondary.TButton',
                                           width=20)
        self.select_folder_btn.grid(row=0, column=1, padx=5)
        
        # 文件列表
        list_frame = ttk.LabelFrame(main_frame, text="文件列表", padding="10")
        list_frame.grid(row=4, column=0, columnspan=2, sticky="nsew", pady=(0, 20))
        list_frame.columnconfigure(0, weight=1)
        list_frame.rowconfigure(0, weight=1)
        main_frame.rowconfigure(4, weight=1)
        
        # 创建Treeview
        columns = ('size', 'progress', 'speed', 'status')
        self.file_tree = ttk.Treeview(list_frame, columns=columns, show='tree headings', height=10)
        self.file_tree.grid(row=0, column=0, sticky="nsew")
        
        # 配置列
        self.file_tree.heading('#0', text='文件名')
        self.file_tree.heading('size', text='大小')
        self.file_tree.heading('progress', text='进度')
        self.file_tree.heading('speed', text='速度')
        self.file_tree.heading('status', text='状态')
        
        self.file_tree.column('#0', width=300)
        self.file_tree.column('size', width=100)
        self.file_tree.column('progress', width=100)
        self.file_tree.column('speed', width=100)
        self.file_tree.column('status', width=100)
        
        # 滚动条
        scrollbar = ttk.Scrollbar(list_frame, orient=tk.VERTICAL, command=self.file_tree.yview)
        scrollbar.grid(row=0, column=1, sticky="ns")
        self.file_tree.configure(yscrollcommand=scrollbar.set)
        
        # 进度条
        self.progress_var = tk.DoubleVar()
        self.progress_bar = ttk.Progressbar(main_frame, variable=self.progress_var, 
                                           maximum=100, length=300)
        self.progress_bar.grid(row=5, column=0, columnspan=2, sticky="we", pady=(0, 10))
        
        # 状态标签
        self.status_label = ttk.Label(main_frame, text="就绪", font=('Arial', 10))
        self.status_label.grid(row=6, column=0, columnspan=2, pady=(0, 10))
        
        # 控制按钮
        control_frame = ttk.Frame(main_frame)
        control_frame.grid(row=7, column=0, columnspan=2)
        
        self.upload_btn = ttk.Button(control_frame, text="⬆️ 开始上传", 
                                    command=self.start_upload,
                                    style='Primary.TButton',
                                    width=20)
        self.upload_btn.grid(row=0, column=0, padx=5)
        self.upload_btn.configure(state='disabled')
        
        self.clear_btn = ttk.Button(control_frame, text="🗑️ 清空列表", 
                                   command=self.clear_files,
                                   style='Secondary.TButton',
                                   width=20)
        self.clear_btn.grid(row=0, column=1, padx=5)
        
    def load_config(self):
        """加载配置"""
        self.config_file = Path.home() / '.go-transfer-gui.json'
        self.config = {}
        
        if self.config_file.exists():
            try:
                with open(self.config_file, 'r') as f:
                    self.config = json.load(f)
            except:
                pass
                
    def save_config(self):
        """保存配置"""
        self.config['server_url'] = self.server_entry.get()
        
        try:
            with open(self.config_file, 'w') as f:
                json.dump(self.config, f)
        except:
            pass
            
    def on_closing(self):
        """窗口关闭时"""
        self.save_config()
        self.root.destroy()
        
    def select_files(self):
        """选择文件"""
        files = filedialog.askopenfilenames(
            title="选择文件",
            filetypes=[("所有文件", "*.*")]
        )
        
        if files:
            self.add_files(files)
            
    def select_folder(self):
        """选择文件夹"""
        folder = filedialog.askdirectory(title="选择文件夹")
        
        if folder:
            # 获取文件夹中的所有文件
            folder_path = Path(folder)
            files = []
            
            for file_path in folder_path.rglob('*'):
                if file_path.is_file():
                    files.append(str(file_path))
                    
            if files:
                self.add_files(files, base_path=folder)
            else:
                messagebox.showinfo("提示", "选择的文件夹中没有文件")
                
    def add_files(self, file_paths, base_path=None):
        """添加文件到列表"""
        for file_path in file_paths:
            path = Path(file_path)
            
            # 计算相对路径（用于文件夹上传）
            if base_path:
                try:
                    relative_path = path.relative_to(base_path)
                    display_name = str(relative_path)
                except:
                    display_name = path.name
            else:
                display_name = path.name
                
            # 检查是否已存在
            exists = any(f['path'] == file_path for f in self.files)
            if exists:
                continue
                
            file_info = {
                'path': file_path,
                'name': display_name,
                'size': path.stat().st_size,
                'status': '等待中',
                'progress': 0,
                'speed': 0,
                'tree_id': None
            }
            
            self.files.append(file_info)
            
            # 添加到树形视图
            tree_id = self.file_tree.insert('', 'end', 
                                          text=display_name,
                                          values=(self.format_size(file_info['size']),
                                                 '0%',
                                                 '-',
                                                 '等待中'))
            file_info['tree_id'] = tree_id
            
        # 启用上传按钮
        if self.files:
            self.upload_btn.configure(state='normal')
            self.status_label.configure(text=f"已选择 {len(self.files)} 个文件")
            
    def clear_files(self):
        """清空文件列表"""
        self.files = []
        self.file_tree.delete(*self.file_tree.get_children())
        self.upload_btn.configure(state='disabled')
        self.progress_var.set(0)
        self.status_label.configure(text="就绪")
        self.is_uploading = False
        
    def format_size(self, bytes):
        """格式化文件大小"""
        for unit in ['B', 'KB', 'MB', 'GB']:
            if bytes < 1024.0:
                return f"{bytes:.2f} {unit}"
            bytes /= 1024.0
        return f"{bytes:.2f} TB"
        
    def start_upload(self):
        """开始上传"""
        if self.is_uploading:
            return
            
        server_url = self.server_entry.get().strip()
        if not server_url:
            messagebox.showerror("错误", "请输入服务器地址")
            return
            
        # 保存配置
        self.save_config()
        
        # 在线程中执行上传
        self.is_uploading = True
        self.upload_btn.configure(state='disabled')
        
        thread = threading.Thread(target=self.upload_worker, args=(server_url,))
        thread.daemon = True
        thread.start()
        
    def upload_worker(self, server_url):
        """上传工作线程"""
        total_files = len(self.files)
        success_count = 0
        error_count = 0
        
        for index, file_info in enumerate(self.files):
            self.current_upload_index = index
            
            # 更新总进度
            overall_progress = (index / total_files) * 100
            self.progress_var.set(overall_progress)
            
            # 更新状态
            self.root.after(0, self.status_label.configure, 
                          text=f"上传中 ({index + 1}/{total_files}): {file_info['name']}")
            
            # 上传文件
            success = self.upload_file(file_info, server_url)
            
            if success:
                success_count += 1
            else:
                error_count += 1
                
        # 完成
        self.progress_var.set(100)
        
        status_text = f"上传完成: 成功 {success_count} 个"
        if error_count > 0:
            status_text += f", 失败 {error_count} 个"
            
        self.root.after(0, self.status_label.configure, text=status_text)
        self.root.after(0, self.upload_btn.configure, state='normal')
        
        self.is_uploading = False
        
        # 显示完成消息
        if error_count == 0:
            self.root.after(0, messagebox.showinfo, "完成", f"全部文件上传成功！共 {success_count} 个文件")
        else:
            self.root.after(0, messagebox.showwarning, "完成", status_text)
            
    def upload_file(self, file_info, server_url):
        """上传单个文件"""
        try:
            # 更新状态
            self.update_file_status(file_info, '上传中', 0, 0)
            
            # 读取文件
            file_path = Path(file_info['path'])
            file_size = file_info['size']
            
            # 构建URL
            if not server_url.endswith('/'):
                server_url += '/'
            url = f"{server_url}upload?name={urllib.parse.quote(file_info['name'])}"
            
            # 创建请求
            req = urllib.request.Request(url, method='POST')
            req.add_header('Content-Type', 'application/octet-stream')
            req.add_header('Content-Length', str(file_size))
            
            # 上传文件（带进度跟踪）
            start_time = time.time()
            
            with open(file_path, 'rb') as f:
                # 包装文件对象以跟踪进度
                class ProgressFileWrapper:
                    def __init__(self, file, callback, total_size):
                        self.file = file
                        self.callback = callback
                        self.total_size = total_size
                        self.uploaded = 0
                        
                    def read(self, size=8192):
                        data = self.file.read(size)
                        if data:
                            self.uploaded += len(data)
                            self.callback(self.uploaded, self.total_size)
                        return data
                
                def progress_callback(uploaded, total):
                    elapsed = time.time() - start_time
                    speed = uploaded / elapsed if elapsed > 0 else 0
                    progress = (uploaded / total) * 100 if total > 0 else 0
                    
                    # 更新界面
                    self.update_file_status(file_info, '上传中', progress, speed)
                
                wrapped_file = ProgressFileWrapper(f, progress_callback, file_size)
                req.data = wrapped_file
                
                # 执行上传
                response = urllib.request.urlopen(req, timeout=3600)
                
                # 检查响应
                if response.status == 200:
                    self.update_file_status(file_info, '✅ 完成', 100, 0)
                    return True
                else:
                    self.update_file_status(file_info, '❌ 失败', 0, 0)
                    return False
                    
        except Exception as e:
            self.update_file_status(file_info, f'❌ 错误', 0, 0)
            print(f"上传失败: {file_info['name']} - {str(e)}")
            return False
            
    def update_file_status(self, file_info, status, progress, speed):
        """更新文件状态（线程安全）"""
        def update():
            if file_info['tree_id']:
                self.file_tree.item(file_info['tree_id'],
                                  values=(self.format_size(file_info['size']),
                                         f"{progress:.1f}%",
                                         f"{self.format_size(speed)}/s" if speed > 0 else '-',
                                         status))
                                         
        self.root.after(0, update)

def main():
    root = tk.Tk()
    app = TransferGUI(root)
    root.mainloop()

if __name__ == "__main__":
    main()