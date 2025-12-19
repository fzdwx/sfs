# sfs

Simple File Server - 一个功能丰富的静态文件服务器

## 功能特性

- 📁 文件浏览和下载
- 📤 文件上传（支持拖拽和多选）
- 📋 图片粘贴上传（Ctrl+V / Cmd+V）
- 📂 创建目录
- 📝 创建和编辑文本文件
- ✏️ 在线文件编辑器（支持 Ctrl+S / Cmd+S 保存）
- 🔍 文件搜索和排序（名称、大小、时间）
- 🖼️ 图片预览和轮播（支持键盘导航）
- 🎨 智能文件图标（20+ 种文件类型）
- 🌐 自动显示所有可访问的 IP 地址
- 💻 支持 curl 命令行上传
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

## curl 上传文件

使用 curl 命令上传文件到服务器：

### 基本用法

```shell
# 上传单个文件
curl -X PUT "http://localhost:8080/api/put?path=filename.txt" --data-binary @localfile.txt

# 上传到子目录
curl -X PUT "http://localhost:8080/api/put?path=docs/report.pdf" --data-binary @report.pdf

# 使用 POST 方法也可以
curl -X POST "http://localhost:8080/api/put?path=image.jpg" --data-binary @photo.jpg
```

### 示例

```shell
# 上传文本文件
echo "Hello World" | curl -X PUT "http://localhost:8080/api/put?path=hello.txt" --data-binary @-

# 上传图片
curl -X PUT "http://localhost:8080/api/put?path=screenshots/screen1.png" --data-binary @screenshot.png

# 上传大文件
curl -X PUT "http://localhost:8080/api/put?path=videos/movie.mp4" --data-binary @movie.mp4
```

### 响应格式

成功上传后返回 JSON：

```json
{
  "success": true,
  "path": "filename.txt",
  "message": "File uploaded successfully"
}
```

## API 接口

- `GET /` - Web 界面
- `GET /editor` - 文件编辑器
- `GET /api/files?path=<path>` - 获取文件列表
- `POST /api/upload` - 上传文件（multipart/form-data）
- `PUT /api/put?path=<path>` - 上传文件（用于 curl，二进制数据）
- `POST /api/mkdir` - 创建目录
- `GET /api/read?path=<path>` - 读取文件内容
- `POST /api/save` - 保存文件
- `GET /files/<path>` - 下载文件
