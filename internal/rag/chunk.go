package rag

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	chunkMaxRunes     = 800
	chunkOverlapRunes = 120
)

var (
	imagePattern = regexp.MustCompile(`!\[[^\]]*\]\([^\)]*\)`)
	spacePattern = regexp.MustCompile(`[ \t]+`)
)

// NormalizeMarkdown 将文章 Markdown 规范化为供 embedding 的稳定来源。标题、段落
// 与代码块保留；front matter 和图片链接移除，避免将无检索价值的媒体 URL 外发。
func NormalizeMarkdown(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	lines := stripFrontMatter(strings.Split(content, "\n"))

	// 围栏代码块是 Markdown 的语义内容，不可把其中的缩进、连续空行或尾随空白
	// 当作普通段落噪声压缩。图片语法也仅在普通正文中移除，避免改变示例代码。
	normalized := make([]string, 0, len(lines))
	inFence := false
	var fenceMarker byte
	var fenceLength int
	for _, line := range lines {
		if inFence {
			normalized = append(normalized, line)
			if isFenceClose(line, fenceMarker, fenceLength) {
				inFence = false
			}
			continue
		}

		if marker, length, ok := fenceStart(line); ok {
			normalized = append(normalized, line)
			inFence, fenceMarker, fenceLength = true, marker, length
			continue
		}
		line = imagePattern.ReplaceAllString(line, "")
		normalized = append(normalized, strings.TrimRight(spacePattern.ReplaceAllString(line, " "), " "))
	}
	return collapseOuterBlankLines(normalized)
}

// stripFrontMatter 只移除文档开头、且有闭合标记的 YAML front matter。逐行识别
// 避免正则的 . 不匹配换行，导致多行 front matter 被错误保留在向量文本中。
func stripFrontMatter(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}
	lines[0] = strings.TrimPrefix(lines[0], "\ufeff")
	if strings.TrimSpace(lines[0]) != "---" {
		return lines
	}
	for index := 1; index < len(lines); index++ {
		marker := strings.TrimSpace(lines[index])
		if marker == "---" || marker == "..." {
			return lines[index+1:]
		}
	}
	// 未闭合的内容不能确定是 front matter，完整保留以避免静默丢失正文。
	return lines
}

func fenceStart(line string) (byte, int, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if len(trimmed) < 3 || (trimmed[0] != '`' && trimmed[0] != '~') {
		return 0, 0, false
	}
	length := 0
	for length < len(trimmed) && trimmed[length] == trimmed[0] {
		length++
	}
	return trimmed[0], length, length >= 3
}

func isFenceClose(line string, marker byte, minimumLength int) bool {
	trimmed := strings.TrimLeft(line, " \t")
	if len(trimmed) < minimumLength || trimmed[0] != marker {
		return false
	}
	length := 0
	for length < len(trimmed) && trimmed[length] == marker {
		length++
	}
	return length >= minimumLength && strings.TrimSpace(trimmed[length:]) == ""
}

func collapseOuterBlankLines(lines []string) string {
	first, last := 0, len(lines)
	for first < last && strings.TrimSpace(lines[first]) == "" {
		first++
	}
	for last > first && strings.TrimSpace(lines[last-1]) == "" {
		last--
	}
	if first == last {
		return ""
	}

	result := make([]string, 0, last-first)
	inFence := false
	var fenceMarker byte
	var fenceLength int
	previousBlank := false
	for _, line := range lines[first:last] {
		if inFence {
			result = append(result, line)
			if isFenceClose(line, fenceMarker, fenceLength) {
				inFence = false
			}
			previousBlank = false
			continue
		}
		if marker, length, ok := fenceStart(line); ok {
			result = append(result, line)
			inFence, fenceMarker, fenceLength = true, marker, length
			previousBlank = false
			continue
		}
		if strings.TrimSpace(line) == "" {
			if previousBlank {
				continue
			}
			previousBlank = true
		} else {
			previousBlank = false
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}

func SourceHash(title, summary, content string) string {
	normalized := strings.TrimSpace(title) + "\n" + strings.TrimSpace(summary) + "\n" + NormalizeMarkdown(content)
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

// SplitMarkdown 优先在标题和段落边界切段；超长段落才按 rune 硬切。每段附带此前
// 的标题路径，确保孤立文段仍保留语义上下文。
func SplitMarkdown(content string) []string {
	normalized := NormalizeMarkdown(content)
	if normalized == "" {
		return []string{}
	}
	blocks := markdownBlocks(normalized)
	headings := make([]string, 0, 6)
	units := make([]string, 0, len(blocks))
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		if level, title, ok := heading(block); ok {
			if level > len(headings) {
				level = len(headings) + 1
			}
			headings = append(headings[:level-1], title)
			continue
		}
		prefix := ""
		if len(headings) > 0 {
			prefix = strings.Join(headings, " / ") + "\n"
		}
		units = append(units, prefix+block)
	}
	if len(units) == 0 && normalized != "" {
		units = []string{normalized}
	}
	chunks := make([]string, 0, len(units))
	current := ""
	for _, unit := range units {
		if runeLen(unit) > chunkMaxRunes {
			if current != "" {
				chunks = append(chunks, current)
				current = ""
			}
			chunks = append(chunks, splitLong(unit)...)
			continue
		}
		candidate := unit
		if current != "" {
			candidate = current + "\n\n" + unit
		}
		if runeLen(candidate) <= chunkMaxRunes {
			current = candidate
			continue
		}
		chunks = append(chunks, current)
		current = unit
	}
	if current != "" {
		chunks = append(chunks, current)
	}
	return chunks
}

// markdownBlocks 只在围栏代码块之外按空行切分；代码中的连续空行、缩进和尾随
// 空白属于代码语义的一部分，不能作为段落边界处理。
func markdownBlocks(content string) []string {
	lines := strings.Split(content, "\n")
	blocks := make([]string, 0, len(lines)/2)
	current := make([]string, 0)
	inFence := false
	var fenceMarker byte
	var fenceLength int
	flush := func() {
		if len(current) == 0 {
			return
		}
		block := joinTrimmedBlankLines(current)
		if block != "" {
			blocks = append(blocks, block)
		}
		current = current[:0]
	}
	for _, line := range lines {
		if inFence {
			current = append(current, line)
			if isFenceClose(line, fenceMarker, fenceLength) {
				inFence = false
				flush()
			}
			continue
		}
		if marker, length, ok := fenceStart(line); ok {
			flush()
			current = append(current, line)
			inFence, fenceMarker, fenceLength = true, marker, length
			continue
		}
		if _, _, ok := heading(line); ok {
			flush()
			blocks = append(blocks, line)
			continue
		}
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		current = append(current, line)
	}
	flush()
	return blocks
}

func joinTrimmedBlankLines(lines []string) string {
	first, last := 0, len(lines)
	for first < last && strings.TrimSpace(lines[first]) == "" {
		first++
	}
	for last > first && strings.TrimSpace(lines[last-1]) == "" {
		last--
	}
	if first == last {
		return ""
	}
	return strings.Join(lines[first:last], "\n")
}

func heading(block string) (int, string, bool) {
	line := strings.TrimSpace(strings.SplitN(block, "\n", 2)[0])
	level := 0
	for level < len(line) && level < 6 && line[level] == '#' {
		level++
	}
	if level == 0 || level >= len(line) || line[level] != ' ' {
		return 0, "", false
	}
	title := strings.TrimSpace(line[level:])
	return level, title, title != ""
}

func splitLong(value string) []string {
	runes := []rune(value)
	if len(runes) <= chunkMaxRunes {
		return []string{value}
	}
	items := make([]string, 0, (len(runes)/chunkMaxRunes)+1)
	for start := 0; start < len(runes); {
		end := start + chunkMaxRunes
		if end > len(runes) {
			end = len(runes)
		}
		items = append(items, string(runes[start:end]))
		if end == len(runes) {
			break
		}
		start = end - chunkOverlapRunes
	}
	return items
}

func runeLen(value string) int { return utf8.RuneCountInString(value) }
