package service

import "strings"

const (
	// 按 rune 计(中文一字一 rune). 目标块长 700, 超过 800 必须再切.
	chunkTargetRunes = 700
	chunkMaxRunes    = 800
)

// Chunk 是文章切分后的最小检索单元.
type Chunk struct {
	Heading string // 所属小节标题路径; 标题前的引言部分为空
	Content string // 小节正文, 不含标题行
}

// SplitMarkdown 按 Markdown 标题切块; 超长小节按空行段落续切,
// 单段超长按 chunkTargetRunes 硬切. 只负责结构切分, 标题/时间等语义
// 由调用方(recall.go embedText)拼进向量文本.
func SplitMarkdown(md string) []Chunk {
	md = strings.ReplaceAll(md, "\r\n", "\n")

	type section struct {
		heading string
		body    strings.Builder
	}
	sections := []*section{{}}
	cur := sections[0]
	for _, line := range strings.Split(md, "\n") {
		if h, ok := headingOf(line); ok {
			cur = &section{heading: h}
			sections = append(sections, cur)
			continue
		}
		cur.body.WriteString(line)
		cur.body.WriteString("\n")
	}

	var chunks []Chunk
	for _, sec := range sections {
		body := strings.TrimSpace(sec.body.String())
		if body == "" {
			continue
		}
		chunks = append(chunks, splitLongSection(sec.heading, body)...)
	}
	return chunks
}

// headingOf 识别 ATX 标题行(# ~ ######), 返回去掉 # 的标题文本.
func headingOf(line string) (string, bool) {
	trimmed := strings.TrimLeft(line, " ")
	if !strings.HasPrefix(trimmed, "#") {
		return "", false
	}
	rest := strings.TrimLeft(trimmed, "#")
	// "#tag" / "###" 这种 # 后没空格的不是标题
	if !strings.HasPrefix(rest, " ") {
		return "", false
	}
	h := strings.TrimSpace(rest)
	return h, h != ""
}

// splitLongSection 把小节正文控制到 chunkMaxRunes 以内: 整节不超长直接一块,
// 否则按段落(空行分隔)归并, 单段仍超长则硬切.
func splitLongSection(heading, body string) []Chunk {
	if len([]rune(body)) <= chunkMaxRunes {
		return []Chunk{{Heading: heading, Content: body}}
	}
	paras := strings.Split(body, "\n\n")
	var chunks []Chunk
	buf := ""
	flush := func() {
		if strings.TrimSpace(buf) != "" {
			chunks = append(chunks, Chunk{Heading: heading, Content: strings.TrimSpace(buf)})
		}
		buf = ""
	}
	for _, p := range paras {
		// 单段超长, 硬切后直接成块
		for len([]rune(p)) > chunkMaxRunes {
			flush()
			runes := []rune(p)
			chunks = append(chunks, Chunk{Heading: heading, Content: string(runes[:chunkTargetRunes])})
			p = string(runes[chunkTargetRunes:])
		}
		if len([]rune(buf))+len([]rune(p))+2 > chunkMaxRunes {
			flush()
		}
		if buf != "" {
			buf += "\n\n"
		}
		buf += p
	}
	flush()
	return chunks
}
