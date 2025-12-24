package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed static
var staticFS embed.FS

var serveDir string

func main() {
	port := flag.Int("p", 8080, "Port to listen on")
	dir := flag.String("d", ".", "Directory to serve")
	flag.Parse()

	startServer(*port, *dir)
}

func startServer(port int, dir string) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	serveDir = absDir

	ips := getLocalIPs()
	fmt.Printf("Starting server on port %d serving directory %s\n", port, absDir)
	fmt.Printf("Access URLs:\n")
	fmt.Printf("  - http://localhost:%d\n", port)
	for _, ip := range ips {
		fmt.Printf("  - http://%s:%d\n", ip, port)
	}

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/editor", handleEditor)
	http.HandleFunc("/api/files", handleFileList)
	http.HandleFunc("/api/upload", handleUpload)
	http.HandleFunc("/api/put", handlePutFile)
	http.HandleFunc("/api/mkdir", handleMkdir)
	http.HandleFunc("/api/rename", handleRename)
	http.HandleFunc("/api/read", handleReadFile)
	http.HandleFunc("/api/save", handleSaveFile)

	// Serve embedded static files
	staticSubFS, _ := fs.Sub(staticFS, "static")
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSubFS))))
	http.HandleFunc("/files/", handleFileServe)

	err = http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func getLocalIPs() []string {
	var ips []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				ips = append(ips, ipNet.IP.String())
			}
		}
	}
	return ips
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	content, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "Failed to load page", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(content)
}

type FileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	IsDir   bool   `json:"isDir"`
	ModTime int64  `json:"modTime"`
}

func handleFileList(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	fullPath := filepath.Join(serveDir, path)

	if !strings.HasPrefix(filepath.Clean(fullPath), serveDir) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		filePath := filepath.Join(path, entry.Name())
		files = append(files, FileInfo{
			Name:    entry.Name(),
			Path:    filePath,
			Size:    info.Size(),
			IsDir:   entry.IsDir(),
			ModTime: info.ModTime().Unix(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"files":   files,
	})
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	path := r.FormValue("path")

	// 检测是否是图片上传（通过检查Content-Type或文件扩展名）
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "No files uploaded",
		})
		return
	}

	var uploadedPath string

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		defer file.Close()

		// 判断是否是图片
		isImage := false
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg"}
		for _, imgExt := range imageExts {
			if ext == imgExt {
				isImage = true
				break
			}
		}

		var destPath string
		if isImage {
			// 图片保存到 /assert/ 目录，使用时间命名
			assertDir := filepath.Join(serveDir, "assert")
			err := os.MkdirAll(assertDir, 0755)
			if err != nil {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"error":   err.Error(),
				})
				return
			}

			// 使用时间格式命名: 20060102_150405.png
			timestamp := time.Now().Format("20060102_150405")
			filename := timestamp + ext
			destPath = filepath.Join(assertDir, filename)
			uploadedPath = "assert/" + filename
		} else {
			// 非图片文件按原路径保存
			fullPath := filepath.Join(serveDir, path)
			if !strings.HasPrefix(filepath.Clean(fullPath), serveDir) {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"error":   "Invalid path",
				})
				return
			}
			destPath = filepath.Join(fullPath, fileHeader.Filename)
			if path != "" {
				uploadedPath = path + "/" + fileHeader.Filename
			} else {
				uploadedPath = fileHeader.Filename
			}
		}

		dest, err := os.Create(destPath)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		defer dest.Close()

		_, err = io.Copy(dest, file)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"path":    uploadedPath,
	})
}

func handlePutFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从查询参数获取目标路径
	path := r.URL.Query().Get("path")
	if path == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Missing path parameter",
		})
		return
	}

	fullPath := filepath.Join(serveDir, path)

	// 路径安全验证
	if !strings.HasPrefix(filepath.Clean(fullPath), serveDir) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid path",
		})
		return
	}

	// 确保父目录存在
	dir := filepath.Dir(fullPath)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// 创建目标文件
	dest, err := os.Create(fullPath)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}
	defer dest.Close()

	// 将请求体写入文件
	_, err = io.Copy(dest, r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"path":    path,
		"message": "File uploaded successfully",
	})
}

func handleMkdir(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	fullPath := filepath.Join(serveDir, req.Path, req.Name)

	if !strings.HasPrefix(filepath.Clean(fullPath), serveDir) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid path",
		})
		return
	}

	err = os.MkdirAll(fullPath, 0755)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

func handleRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OldPath string `json:"oldPath"`
		NewName string `json:"newName"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	oldFullPath := filepath.Join(serveDir, req.OldPath)

	if !strings.HasPrefix(filepath.Clean(oldFullPath), serveDir) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid path",
		})
		return
	}

	dir := filepath.Dir(oldFullPath)
	newFullPath := filepath.Join(dir, req.NewName)

	if !strings.HasPrefix(filepath.Clean(newFullPath), serveDir) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid new name",
		})
		return
	}

	err = os.Rename(oldFullPath, newFullPath)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

func handleFileServe(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/files/")
	fullPath := filepath.Join(serveDir, path)

	if !strings.HasPrefix(filepath.Clean(fullPath), serveDir) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	http.ServeFile(w, r, fullPath)
}

func handleEditor(w http.ResponseWriter, r *http.Request) {
	content, err := staticFS.ReadFile("static/editor.html")
	if err != nil {
		http.Error(w, "Failed to load page", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(content)
}

func handleReadFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	fullPath := filepath.Join(serveDir, path)

	if !strings.HasPrefix(filepath.Clean(fullPath), serveDir) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid path",
		})
		return
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"content": string(content),
	})
}

func handleSaveFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	fullPath := filepath.Join(serveDir, req.Path)

	if !strings.HasPrefix(filepath.Clean(fullPath), serveDir) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid path",
		})
		return
	}

	dir := filepath.Dir(fullPath)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	err = os.WriteFile(fullPath, []byte(req.Content), 0644)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}
