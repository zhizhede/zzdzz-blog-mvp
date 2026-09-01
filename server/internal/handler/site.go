package handler

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"zzdzz-blog/server/internal/service"
	"zzdzz-blog/server/pkg/response"
)

// SiteHandler 站点级设置: 自定义 favicon.
type SiteHandler struct {
	icons *service.IconService
}

func NewSiteHandler(icons *service.IconService) *SiteHandler {
	return &SiteHandler{icons: icons}
}

const maxIconUpload = 5 << 20 // 5MB

// Upload PUT /site/icon : 上传 PNG/JPG 源图, 生成全套 favicon.
func (h *SiteHandler) Upload(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "missing file field")
		return
	}
	if fh.Size > maxIconUpload {
		response.Fail(c, 400, 4002, "图片超过 5MB 限制")
		return
	}
	if !strings.HasSuffix(strings.ToLower(fh.Filename), ".png") && !strings.HasSuffix(strings.ToLower(fh.Filename), ".jpg") && !strings.HasSuffix(strings.ToLower(fh.Filename), ".jpeg") {
		response.Fail(c, 400, 4002, "仅支持 PNG/JPG 源图")
		return
	}
	f, err := fh.Open()
	if err != nil {
		response.BadRequest(c, "read upload failed")
		return
	}
	defer f.Close()
	// 上传件上限已限, 限长读防意外大 body
	src, err := io.ReadAll(io.LimitReader(f, maxIconUpload+1))
	if err != nil || len(src) > maxIconUpload {
		response.Fail(c, 400, 4002, "图片超过 5MB 限制")
		return
	}
	meta, serr := h.icons.Set(src)
	if serr != nil {
		response.Fail(c, 400, 4002, serr.Error())
		return
	}
	response.OK(c, gin.H{"updated_at": meta.UpdatedAt})
}

// Reset DELETE /site/icon : 恢复内置默认图标.
func (h *SiteHandler) Reset(c *gin.Context) {
	if err := h.icons.Reset(); err != nil {
		response.ServerError(c, err.Error())
		return
	}
	response.OK(c, nil)
}

// Meta GET /site/icon/meta : 当前图标状态(后台设置页用).
func (h *SiteHandler) Meta(c *gin.Context) {
	v := h.icons.Version()
	response.OK(c, gin.H{
		"custom":     v != "",
		"updated_at": v,
	})
}

// indexHTMLCache 首页 HTML 的 favicon 版本注入.
// 匹配 href="/favicon..." 的 8 个 link 标签, 追加 ?v=<版本>.
var faviconHrefRe = regexp.MustCompile(`(href="/(?:favicon[^"]*|apple-touch-icon\.png))"`)

// ServeIndex 返回 index.html; 有自定义图标时给 favicon 链接追加 ?v=,
// URL 变化使所有访客的浏览器缓存立即失效.
func (h *SiteHandler) ServeIndex(webDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		buf, err := os.ReadFile(webDir + "/index.html")
		if err != nil {
			c.String(http.StatusNotFound, "index.html not found")
			return
		}
		html := string(buf)
		if v := h.icons.Version(); v != "" {
			html = faviconHrefRe.ReplaceAllString(html, `$1?v=`+v+`"`)
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	}
}

// ServeIconFile favicon 类静态文件: 优先自定义图标, 无则回退内置.
// 带版本号的 URL 一律 immutable 长缓存; 无版本(默认图标)走短缓存,
// 以便"恢复默认"后尽快生效.
func (h *SiteHandler) ServeIconFile(webDir, name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if v := h.icons.Version(); v != "" {
			p := filepath.Join(h.icons.Dir(), name)
			if _, err := os.Stat(p); err == nil {
				if c.Query("v") == v {
					c.Header("Cache-Control", "public, max-age=31536000, immutable")
				} else {
					c.Header("Cache-Control", "no-cache")
				}
				c.File(p)
				return
			}
		}
		p := filepath.Join(webDir, name)
		if _, err := os.Stat(p); err != nil {
			c.String(http.StatusNotFound, "not found")
			return
		}
		c.Header("Cache-Control", "no-cache")
		c.File(p)
	}
}
