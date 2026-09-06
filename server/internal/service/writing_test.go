package service

import (
	"strings"
	"testing"
)

func TestFenceStripper(t *testing.T) {
	cases := []struct {
		name  string
		delts []string
		want  string
	}{
		{
			name:  "无围栏原样通过",
			delts: []string{"你好", "世界"},
			want:  "你好世界",
		},
		{
			name:  "头部围栏一次到达",
			delts: []string{"```markdown\n# 标题\n正文```"},
			want:  "# 标题\n正文",
		},
		{
			name:  "头部围栏跨分片",
			delts: []string{"``", "`markdown", "\n# 标题", "\n正文", "\n``", "`"},
			want:  "# 标题\n正文",
		},
		{
			name:  "尾部围栏跨分片(含换行)",
			delts: []string{"正文\n\n", "``", "`"},
			want:  "正文",
		},
		{
			name:  "尾部围栏一次到达且带换行",
			delts: []string{"正文\n```\n"},
			want:  "正文",
		},
		{
			name:  "纯围栏头无正文",
			delts: []string{"```markdown"},
			want:  "",
		},
		{
			name:  "正文中间的代码围栏不受影响",
			delts: []string{"前文\n```go\ncode()\n```\n后文"},
			want:  "前文\n```go\ncode()\n```\n后文",
		},
		{
			name:  "裸反引号短暂挂起后原样吐出",
			delts: []string{"行内 `code` 标记"},
			want:  "行内 `code` 标记",
		},
		{
			name:  "普通空行不被吞",
			delts: []string{"段落一\n\n\n段落二"},
			want:  "段落一\n\n\n段落二",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs := NewFenceStripper()
			var got string
			for _, d := range tc.delts {
				got += fs.Write(d)
			}
			got += fs.Finish()
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBuildComposeMessagesStyleCard(t *testing.T) {
	msgs := BuildComposeMessages(ComposeInput{Action: ComposeDraft, Material: "语料", StyleCard: "爱用短句与破折号"})
	if !strings.Contains(msgs[0].Content, "爱用短句与破折号") {
		t.Fatalf("style card should be injected into system prompt, got: %s", msgs[0].Content)
	}
	// 未传风格卡时回退通用文风描述
	plain := BuildComposeMessages(ComposeInput{Action: ComposeDraft, Material: "语料"})
	if strings.Contains(plain[0].Content, "爱用短句与破折号") {
		t.Fatal("style card should not leak without input")
	}
	if !strings.Contains(plain[0].Content, "文风:") {
		t.Fatal("default style hint missing")
	}
}

func TestBuildComposeMessages(t *testing.T) {
	msgs := BuildComposeMessages(ComposeInput{
		Action:      ComposeRefine,
		Material:    "  语料内容  ",
		Draft:       "稿件",
		Instruction: "更口语",
	})
	if len(msgs) != 2 {
		t.Fatalf("expect 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "system" || msgs[1].Role != "user" {
		t.Fatalf("roles wrong: %s / %s", msgs[0].Role, msgs[1].Role)
	}
	// 空段不拼; refine 的契约必须要求返回全文而非 diff
	for _, want := range []string{"<语料>", "<当前稿件>", "<本次要求>", "返回调整后的全文"} {
		if !strings.Contains(msgs[1].Content+msgs[0].Content, want) {
			t.Fatalf("prompt missing %q", want)
		}
	}
	if strings.Contains(msgs[1].Content, "<当前大纲>") {
		t.Fatalf("empty outline section should be skipped")
	}
}
