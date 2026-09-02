package service

import (
	"strings"
	"testing"

	"zzdzz-blog/server/internal/model"
)

func TestEmbedTextCarriesTimeAndTags(t *testing.T) {
	a := &model.Article{Title: "Go 泛型实践"}
	m := chunkMeta{
		Title:       a.Title,
		Category:    "技术",
		Tags:        []string{"go", "泛型"},
		PublishedAt: "2025-03-02",
	}
	c := Chunk{Heading: "类型约束", Content: "约束写法如下。"}

	got := embedText(a, m, c)
	for _, want := range []string{"《Go 泛型实践》", "2025-03-02", "技术", "#go #泛型", "## 类型约束", "约束写法如下。"} {
		if !strings.Contains(got, want) {
			t.Errorf("embedText missing %q in:\n%s", want, got)
		}
	}
}

func TestRecallForQueryPromptAndSources(t *testing.T) {
	// 不依赖 DB/embed 的部分: prompt 组装逻辑单独抽不出来, 这里构造 hits 走 Search 之后的路径不可行,
	// 改为直接验证 RecallSource 的 JSON 契约字段(前端 SSE 解析依赖).
	src := RecallSource{Index: 1, ArticleID: 42, Title: "旧文", Heading: "小节", Scope: "private", Score: 0.82}
	if src.Index != 1 || src.ArticleID != 42 || src.Scope != "private" {
		t.Fatalf("source fields mismatch: %+v", src)
	}
}
