package service

import (
	"strings"
	"testing"
)

func TestSplitMarkdownByHeadings(t *testing.T) {
	md := `引言部分,没有标题。

## 安装

先安装依赖,再初始化。

## 使用

### 启动服务

执行 start 命令。
`
	chunks := SplitMarkdown(md)
	if len(chunks) != 3 {
		t.Fatalf("want 3 chunks, got %d: %+v", len(chunks), chunks)
	}
	if chunks[0].Heading != "" || !strings.Contains(chunks[0].Content, "引言部分") {
		t.Errorf("chunk 0 should be heading-less intro, got %+v", chunks[0])
	}
	if chunks[1].Heading != "安装" || !strings.Contains(chunks[1].Content, "先安装依赖") {
		t.Errorf("chunk 1 mismatch: %+v", chunks[1])
	}
	if chunks[2].Heading != "启动服务" || !strings.Contains(chunks[2].Content, "start 命令") {
		t.Errorf("chunk 2 mismatch: %+v", chunks[2])
	}
}

func TestSplitMarkdownLongSection(t *testing.T) {
	para := strings.Repeat("这是一段足够长的中文内容,用来撑大小节体积。", 30) // ~660 runes
	sep := "段落分隔。"
	md := "## 长小节\n\n" + para + "\n\n" + sep + "\n\n" + para + "\n\n" + para

	chunks := SplitMarkdown(md)
	if len(chunks) < 2 {
		t.Fatalf("long section should split into multiple chunks, got %d", len(chunks))
	}
	total := 0
	for _, c := range chunks {
		total += len([]rune(c.Content))
		if c.Heading != "长小节" {
			t.Errorf("all chunks should keep heading, got %q", c.Heading)
		}
		if len([]rune(c.Content)) > chunkMaxRunes {
			t.Errorf("chunk over max runes: %d", len([]rune(c.Content)))
		}
	}
	// 内容不丢: 总 rune 数应与原文接近(允许去空白差异), 至少不能少于最大块
	if total < len([]rune(para)) {
		t.Errorf("content lost: total %d runes < single para %d", total, len([]rune(para)))
	}
}

func TestSplitMarkdownHugeParagraphHardSplit(t *testing.T) {
	huge := strings.Repeat("硬", chunkMaxRunes+500)
	chunks := SplitMarkdown(huge)
	if len(chunks) != 2 {
		t.Fatalf("huge paragraph should hard-split into 2 chunks, got %d", len(chunks))
	}
	if got := len([]rune(chunks[0].Content)); got != chunkTargetRunes {
		t.Errorf("first hard chunk should be %d runes, got %d", chunkTargetRunes, got)
	}
}

func TestSplitMarkdownHashTagNotHeading(t *testing.T) {
	md := "正文提到 #技术标签 和 topic。\n\n## 真标题\n\n内容。"
	chunks := SplitMarkdown(md)
	if len(chunks) != 2 {
		t.Fatalf("want 2 chunks, got %d: %+v", len(chunks), chunks)
	}
	if chunks[0].Heading != "" || !strings.Contains(chunks[0].Content, "#技术标签") {
		t.Errorf("hash-tag line must not become a heading: %+v", chunks[0])
	}
}

func TestSplitMarkdownEmpty(t *testing.T) {
	for _, md := range []string{"", "   \n\n  ", "## 只有标题"} {
		if got := SplitMarkdown(md); len(got) != 0 {
			t.Errorf("SplitMarkdown(%q) should be empty, got %+v", md, got)
		}
	}
}

func TestVectorLiteral(t *testing.T) {
	got := vectorLiteral([]float32{0.1, -2, 3.5})
	if got != "[0.1,-2,3.5]" {
		t.Errorf("unexpected literal: %s", got)
	}
}
