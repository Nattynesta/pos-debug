package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
)

const (
	thumbMaxWidth  = 200
	compressMaxW   = 1200
	compressQuality = 80
)

func compressImage(path string) error {
	src, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("abrir: %w", err)
	}
	defer src.Close()

	srcImg, format, err := image.Decode(src)
	if err != nil {
		return fmt.Errorf("decodificar: %w", err)
	}
	src.Close()

	bounds := srcImg.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	if w <= compressMaxW && format == "jpeg" {
		return nil
	}

	newW, newH := w, h
	if w > compressMaxW {
		ratio := float64(compressMaxW) / float64(w)
		newW = compressMaxW
		newH = int(float64(h) * ratio)
	}

	var dstImg image.Image = srcImg
	if newW != w {
		dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
		draw.CatmullRom.Scale(dst, dst.Bounds(), srcImg, srcImg.Bounds(), draw.Over, nil)
		dstImg = dst
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".webp" {
		path = strings.TrimSuffix(path, ext) + ".jpg"
	}

	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("crear: %w", err)
	}
	defer out.Close()

	switch ext {
	case ".png":
		return png.Encode(out, dstImg)
	default:
		return jpeg.Encode(out, dstImg, &jpeg.Options{Quality: compressQuality})
	}
}

func createThumbnail(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("abrir origen: %w", err)
	}
	defer src.Close()

	srcImg, format, err := image.Decode(src)
	if err != nil {
		return fmt.Errorf("decodificar imagen: %w", err)
	}

	bounds := srcImg.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	if w <= thumbMaxWidth {
		return copyFile(srcPath, dstPath)
	}

	ratio := float64(thumbMaxWidth) / float64(w)
	newW := thumbMaxWidth
	newH := int(float64(h) * ratio)

	dstImg := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(dstImg, dstImg.Bounds(), srcImg, srcImg.Bounds(), draw.Over, nil)

	os.MkdirAll(filepath.Dir(dstPath), 0755)

	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("crear thumbnail: %w", err)
	}
	defer dst.Close()

	switch strings.ToLower(format) {
	case "png":
		return png.Encode(dst, dstImg)
	default:
		return jpeg.Encode(dst, dstImg, &jpeg.Options{Quality: 80})
	}
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	os.MkdirAll(filepath.Dir(dst), 0755)
	return os.WriteFile(dst, data, 0644)
}

func thumbnailPath(codigo, originalPath string) string {
	ext := filepath.Ext(originalPath)
	return filepath.Join("static", "img", "productos", "thumbs", codigo+ext)
}

func thumbnailURL(codigo, originalURL string) string {
	if originalURL == "" {
		return ""
	}
	return "/static/img/productos/thumbs/" + codigo + filepath.Ext(originalURL)
}
