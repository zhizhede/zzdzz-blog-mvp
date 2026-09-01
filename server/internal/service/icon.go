package service

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	idraw "image/draw"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"time"

	xdraw "golang.org/x/image/draw"

	// 注册 png/jpeg/gif 解码器, 上传图标三种类别都收
	_ "image/gif"
	_ "image/jpeg"
)

// IconService 管理站点自定义 favicon:
//   - 上传一张源图(任意尺寸, 自动居中裁方), 生成全套尺寸落到 iconDir
//   - 文件即状态: meta.json 记录 updated_at, 不建表
//   - 生成是"写临时文件 + rename"的原子替换, 中途失败不会留下半套图标
type IconService struct {
	dir string
	mu  sync.Mutex
}

// 生成的文件名与 web/public 内置图标同名, 覆盖读取时优先生效
var iconFiles = []string{
	"favicon.svg",
	"favicon.ico",
	"apple-touch-icon.png",
	"favicon-32.png", "favicon-48.png", "favicon-64.png",
	"favicon-128.png", "favicon-256.png", "favicon-512.png",
}

func NewIconService(dir string) (*IconService, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create icon dir: %w", err)
	}
	return &IconService{dir: dir}, nil
}

type IconMeta struct {
	UpdatedAt int64 `json:"updated_at"` // unix 秒; 0 表示无自定义图标
}

// Version 返回当前自定义图标的版本(unix 秒字符串), 无自定义时返回空串.
// 用作 favicon URL 的 ?v= 参数, URL 变化即绕开一切浏览器缓存.
func (s *IconService) Version() string {
	m, err := s.meta()
	if err != nil || m.UpdatedAt == 0 {
		return ""
	}
	return fmt.Sprintf("%d", m.UpdatedAt)
}

// Open 打开自定义图标文件; 不存在返回 os.ErrNotExist, 由调用方回退内置图标.
func (s *IconService) Open(name string) (*os.File, error) {
	for _, f := range iconFiles {
		if f != name {
			continue
		}
		return os.Open(filepath.Join(s.dir, name))
	}
	return nil, os.ErrNotExist
}

// Dir 返回图标存储目录, 供 handler 直接读文件.
func (s *IconService) Dir() string { return s.dir }

// Set 解码源图并生成全套图标, 原子替换目录内容.
func (s *IconService) Set(src []byte) (*IconMeta, error) {
	img, _, err := image.Decode(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("无法解码图片, 请上传 PNG/JPG: %w", err)
	}

	square := centerCropSquare(img)

	tmp, err := os.MkdirTemp(s.dir, ".tmp-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	// 512 方形源, 全部尺寸由它缩放
	square512 := scale(square, 512)

	// favicon.svg: 内嵌 256px PNG base64 (与 scripts/build-favicon.mjs 同构)
	png256, err := encodePNG(scale(square512, 256))
	if err != nil {
		return nil, err
	}
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" width="256" height="256">
  <image href="data:image/png;base64,%s" width="256" height="256"/>
</svg>
`, base64.StdEncoding.EncodeToString(png256))
	if err := writeFile(tmp, "favicon.svg", []byte(svg)); err != nil {
		return nil, err
	}

	// 各尺寸 PNG
	for _, size := range []int{32, 48, 64, 128, 256, 512} {
		buf, err := encodePNG(scale(square512, size))
		if err != nil {
			return nil, err
		}
		if err := writeFile(tmp, fmt.Sprintf("favicon-%d.png", size), buf); err != nil {
			return nil, err
		}
	}

	// apple-touch-icon 180
	apple, err := encodePNG(scale(square512, 180))
	if err != nil {
		return nil, err
	}
	if err := writeFile(tmp, "apple-touch-icon.png", apple); err != nil {
		return nil, err
	}

	// favicon.ico: 16+32+48, PNG 内嵌式容器
	ico, err := buildICO(square512)
	if err != nil {
		return nil, err
	}
	if err := writeFile(tmp, "favicon.ico", ico); err != nil {
		return nil, err
	}

	// 原始件留档
	if err := writeFile(tmp, "source.png", src); err != nil {
		return nil, err
	}

	meta := IconMeta{UpdatedAt: time.Now().Unix()}
	metaBuf, _ := json.Marshal(meta)
	if err := writeFile(tmp, "meta.json", metaBuf); err != nil {
		return nil, err
	}

	// 原子生效: 逐个 rename 到位. rename 同分区上是原子操作,
	// 生成的文件自洽(互为独立副本), 单个失败重跑 Set 即恢复.
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, f := range append(iconFiles, "source.png", "meta.json") {
		if err := os.Rename(filepath.Join(tmp, f), filepath.Join(s.dir, f)); err != nil {
			return nil, fmt.Errorf("replace %s: %w", f, err)
		}
	}
	return &meta, nil
}

// Reset 删除全部自定义图标, 恢复到内置默认. 已是默认时静默成功.
func (s *IconService) Reset() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, f := range append(iconFiles, "source.png", "meta.json") {
		if err := os.Remove(filepath.Join(s.dir, f)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (s *IconService) meta() (IconMeta, error) {
	var m IconMeta
	buf, err := os.ReadFile(filepath.Join(s.dir, "meta.json"))
	if err != nil {
		return m, err
	}
	return m, json.Unmarshal(buf, &m)
}

// centerCropSquare 居中裁方: 取短边, 长边两侧各去一半.
func centerCropSquare(img image.Image) image.Image {
	b := img.Bounds()
	side := min(b.Dx(), b.Dy())
	r := image.Rect(
		b.Min.X+(b.Dx()-side)/2,
		b.Min.Y+(b.Dy()-side)/2,
		b.Min.X+(b.Dx()-side)/2+side,
		b.Min.Y+(b.Dy()-side)/2+side,
	)
	if sub, ok := img.(interface{ SubImage(image.Rectangle) image.Image }); ok {
		return sub.SubImage(r)
	}
	// 解码器返回的类型不支持 SubImage 时手动拷贝
	dst := image.NewNRGBA(image.Rect(0, 0, side, side))
	idraw.Draw(dst, dst.Bounds(), img, r.Min, idraw.Src)
	return dst
}

// scale 用 CatmullRom 缩放到 size x size.
func scale(src image.Image, size int) image.Image {
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	return dst
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeFile(dir, name string, data []byte) error {
	return os.WriteFile(filepath.Join(dir, name), data, 0o644)
}

// buildICO 构造 PNG 内嵌式 ICO (16/32/48), 容器格式与 scripts/build-favicon.mjs 一致.
func buildICO(square512 image.Image) ([]byte, error) {
	sizes := []int{16, 32, 48}
	entries := make([][]byte, len(sizes))
	total := 6 + 16*len(sizes)
	for i, s := range sizes {
		buf, err := encodePNG(scale(square512, s))
		if err != nil {
			return nil, err
		}
		entries[i] = buf
		total += len(buf)
	}

	out := bytes.NewBuffer(make([]byte, 0, total))
	binary.Write(out, binary.LittleEndian, uint16(0))               // reserved
	binary.Write(out, binary.LittleEndian, uint16(1))               // type: ICO
	binary.Write(out, binary.LittleEndian, uint16(len(entries)))    // count
	offset := 6 + 16*len(entries)
	for i, s := range sizes {
		dim := uint8(s)
		if s >= 256 {
			dim = 0
		}
		b := byte(0) // colors
		out.WriteByte(dim)
		out.WriteByte(dim)
		out.WriteByte(b)
		out.WriteByte(0) // reserved
		binary.Write(out, binary.LittleEndian, uint16(1))  // planes
		binary.Write(out, binary.LittleEndian, uint16(32)) // bpp
		binary.Write(out, binary.LittleEndian, uint32(len(entries[i])))
		binary.Write(out, binary.LittleEndian, uint32(offset))
		offset += len(entries[i])
	}
	for _, buf := range entries {
		out.Write(buf)
	}
	return out.Bytes(), nil
}
