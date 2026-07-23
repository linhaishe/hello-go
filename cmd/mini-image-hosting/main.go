package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	uploadDir     = "uploads"
	maxUploadSize = 16 << 20 // 16 MB
)

type FileInfo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Size int64  `json:"size"`
}

func main() {
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatalf("failed to create upload directory: %v", err)
	}

	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/files", filesHandler)
	http.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir))))
	http.HandleFunc("/", rootHandler)

	addr := ":8080"
	log.Printf("listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Mini Image Hosting</title>
</head>
<body>
<h1>Mini Image Hosting</h1>
<form action="/upload" method="post" enctype="multipart/form-data">
  <label for="file">Select image:</label>
  <input type="file" id="file" name="file" accept="image/*" required>
  <button type="submit">Upload</button>
</form>
<p>Uploaded files: <a href="/files">/files</a></p>
<p>Preview images directly at <code>/uploads/&lt;filename&gt;</code>.</p>
</body>
</html>`)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		log.Printf("upload parse error: %v", err)
		http.Error(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("upload form file error: %v", err)
		http.Error(w, "missing file field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	filename := fmt.Sprintf("%d-%s", time.Now().UnixNano(), filepath.Base(header.Filename))
	dstPath := filepath.Join(uploadDir, filename)

	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "failed to create file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}

	response := struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}{
		Name: filename,
		URL:  "/uploads/" + filename,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func filesHandler(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		http.Error(w, "failed to read upload directory", http.StatusInternalServerError)
		return
	}

	var files []FileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, FileInfo{
			Name: entry.Name(),
			URL:  "/uploads/" + entry.Name(),
			Size: info.Size(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}
