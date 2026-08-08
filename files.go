package main

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
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

func ConvertToWebp(srcPath, destPath string, maxWidth, maxHeight int) error {
	width, height, err := imageSize(srcPath)
	if err != nil {
		return fmt.Errorf("read image size: %w", err)
	}
	args := []string{
		"-q", "85",
		"-mt",
	}
	// Добавляем resize только если реально нужно уменьшать
	if width > maxWidth || height > maxHeight {
		args = append(
			args,
			"-resize",
			fmt.Sprintf("%d", maxWidth),
			fmt.Sprintf("%d", maxHeight),
		)
	}
	args = append(
		args,
		srcPath,
		"-o",
		destPath,
	)
	cmd := exec.Command("cwebp", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cwebp failed: %w, output: %s", err, output)
	}
	if err := os.Remove(srcPath); err != nil {
		log.Println("remove source file: %w", err)
	}
	return nil
}

func imageSize(path string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}
