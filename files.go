package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"
)

const (
	maxImageBytes  = 5 << 20
	maxImageSide   = 6000
	maxImagePixels = 16_000_000
)

var (
	ErrBadImage = errors.New("bad image")
	cwebpSem    = make(chan struct{}, 2)
)

func ParseMultipart(r *http.Request) (*multipart.Form, *multipart.FileHeader, error) {
	err := r.ParseMultipartForm(32 << 20)
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

func inspectImage(fh *multipart.FileHeader) (image.Config, string, error) {
	if fh.Size > maxImageBytes {
		return image.Config{}, "", ErrBadImage
	}
	f, err := fh.Open()
	if err != nil {
		return image.Config{}, "", err
	}
	defer f.Close()
	cfg, format, err := image.DecodeConfig(f)
	if err != nil || (format != "png" && format != "jpeg") {
		return image.Config{}, "", ErrBadImage
	}
	if cfg.Width < 1 || cfg.Height < 1 ||
		cfg.Width > maxImageSide || cfg.Height > maxImageSide ||
		cfg.Width*cfg.Height > maxImagePixels {
		return image.Config{}, "", ErrBadImage
	}
	return cfg, format, nil
}

// CheckImage — cheap check BEFORE writing to the DB.
func CheckImage(fh *multipart.FileHeader) error {
	_, _, err := inspectImage(fh)
	return err
}

// SaveAsWebp — the only path by which an uploaded file gets into IMGS.
func SaveAsWebp(fh *multipart.FileHeader, destPath string, maxW, maxH int) error {
	cfg, format, err := inspectImage(fh)
	if err != nil {
		return err
	}
	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	tmp, err := os.CreateTemp("", "up-*."+format)
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	n, err := io.Copy(tmp, io.LimitReader(src, maxImageBytes+1))
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	if n > maxImageBytes {
		return ErrBadImage
	}

	args := []string{"-q", "85", "-mt"}
	if cfg.Width > maxW || cfg.Height > maxH {
		args = append(args, "-resize", strconv.Itoa(maxW), strconv.Itoa(maxH))
	}
	args = append(args, tmp.Name(), "-o", destPath)

	cwebpSem <- struct{}{}
	defer func() { <-cwebpSem }()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "cwebp", args...).CombinedOutput()
	if err != nil {
		os.Remove(destPath)
		return fmt.Errorf("cwebp: %w: %s", err, out)
	}
	return nil
}
