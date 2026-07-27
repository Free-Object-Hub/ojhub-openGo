package main

import (
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func ParseMultipart(r *http.Request) (*multipart.Form, *multipart.FileHeader, error) {
	err := r.ParseMultipartForm(32 << 20) // 32MB
	if err != nil {
		return nil, nil, err
	}
	var file *multipart.FileHeader
	files := r.MultipartForm.File["files"]
	if len(files) > 0 {
		file = files[0]
	}
	return r.MultipartForm, file, nil
}

func GetFileExt(file *multipart.FileHeader) string {
	if file == nil {
		return ""
	}
	ext := filepath.Ext(file.Filename)
	if len(ext) > 0 {
		return strings.TrimPrefix(
			strings.ToLower(ext),
			".",
		)
	}
	return ""
}

func SaveFile(file *multipart.FileHeader, path string) error {
	if file == nil {
		return nil
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.Create(path)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}
