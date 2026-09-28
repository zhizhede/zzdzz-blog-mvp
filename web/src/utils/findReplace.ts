// 查找替换: 字面量匹配(用户输入按普通文本处理, 不当正则解析).
// countMatches 供匹配计数, buildHighlightHtml 供编辑器高亮 overlay 使用.

export function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

export function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

// 非重叠命中次数(split 计数, 与「全部替换」口径一致)
export function countMatches(text: string, query: string): number {
  if (!query) return 0
  return text.split(query).length - 1
}

// 命中片段包 <mark>; 在原文上切分后分段转义(先整体转义会让偏移错位).
// overlay 层把文字设为透明, 只渲染色块, 不与 textarea 文字叠影.
export function buildHighlightHtml(text: string, query: string): string {
  let out = ''
  let last = 0
  if (query) {
    const re = new RegExp(escapeRegExp(query), 'g')
    let m: RegExpExecArray | null
    while ((m = re.exec(text)) !== null) {
      out += escapeHtml(text.slice(last, m.index)) + '<mark>' + escapeHtml(m[0]) + '</mark>'
      last = m.index + m[0].length
      if (m[0].length === 0) re.lastIndex++ // 空匹配防御, 正常路径不会触发
    }
  }
  return out + escapeHtml(text.slice(last))
}
