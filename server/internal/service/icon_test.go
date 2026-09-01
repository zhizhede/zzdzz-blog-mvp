package service

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func makeTestPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestIconSetAndReset(t *testing.T) {
	dir := t.TempDir()
	svc, err := NewIconService(dir)
	if err != nil {
		t.Fatal(err)
	}

	// 非方形源图 (200x100): 应居中裁方后生成全套
	meta, err := svc.Set(makeTestPNG(t, 200, 100))
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if meta.UpdatedAt == 0 {
		t.Fatal("updated_at should be set")
	}
	if svc.Version() == "" {
		t.Fatal("Version should be non-empty after Set")
	}

	for _, f := range iconFiles {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("missing generated file %s: %v", f, err)
		}
	}
	// source.png 留档 + meta.json
	for _, f := range []string{"source.png", "meta.json"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("missing %s: %v", f, err)
		}
	}

	// Open 白名单校验: 合法文件可开, 非法名字拒绝
	if f, err := svc.Open("favicon-32.png"); err != nil {
		t.Errorf("Open favicon-32.png: %v", err)
	} else {
		f.Close()
	}
	if _, err := svc.Open("../../secrets"); err == nil {
		t.Error("Open should reject non-whitelisted names")
	}

	// Reset 后全部清理, Version 归零
	if err := svc.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if svc.Version() != "" {
		t.Fatal("Version should be empty after Reset")
	}
	for _, f := range iconFiles {
		if _, err := os.Stat(filepath.Join(dir, f)); !os.IsNotExist(err) {
			t.Errorf("%s should be removed after Reset", f)
		}
	}
}

func TestIconSetRejectsGarbage(t *testing.T) {
	dir := t.TempDir()
	svc, err := NewIconService(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Set([]byte("not an image")); err == nil {
		t.Fatal("Set should reject non-image bytes")
	}
	if svc.Version() != "" {
		t.Fatal("failed Set should leave no version behind")
	}
}
