# sfs

Simple File Server - 一个功能丰富的静态文件服务器

## 功能特性

- 📁 文件浏览和下载
- 📤 文件上传（支持拖拽和多选）
- 📋 图片粘贴上传（Ctrl+V / Cmd+V）
- 📂 创建目录
- 📝 创建和编辑文本文件
- ✏️ 在线文件编辑器（支持 Ctrl+S / Cmd+S 保存）
- 🌐 自动显示所有可访问的 IP 地址
- 🎨 现代化的 Web 界面
- 🔒 路径安全验证

## 使用方法

```shell
Usage of sfs:
  -d string
        Directory to serve (default ".")
  -p int
        Port to listen on (default 8080)
```

启动服务器后，会显示所有可访问的 URL：

```
Starting server on port 8080 serving directory /path/to/dir
Access URLs:
  - http://localhost:8080
  - http://192.168.1.100:8080
  - http://172.17.0.1:8080
```

## 安装

```shell
go install github.com/fzdwx/sfs@main
```

或者从源码构建：

```shell
git clone https://github.com/fzdwx/sfs.git
cd sfs
go build
./sfs
```

## API 接口

- `GET /` - Web 界面
- `GET /editor` - 文件编辑器
- `GET /api/files?path=<path>` - 获取文件列表
- `POST /api/upload` - 上传文件
- `POST /api/mkdir` - 创建目录
- `GET /api/read?path=<path>` - 读取文件内容
- `POST /api/save` - 保存文件
- `GET /files/<path>` - 下载文件
