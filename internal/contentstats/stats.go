package contentstats

import (
	"math"
	"regexp"
	"strings"
	"unicode"
)

type Stats struct {
	WordCount          int
	ReadingTimeMinutes int
}

var (
	frontMatterPattern = regexp.MustCompile(`(?s)^---\s*\n.*?\n---\s*\n`)
	imagePattern       = regexp.MustCompile(`!\[([^\]]*)\]\([^)]*\)`)
	linkPattern        = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)
	htmlPattern        = regexp.MustCompile(`<[^>]+>`)
	urlPattern         = regexp.MustCompile(`https?://\S+`)
)

// Analyze 以可见 Markdown 文本为基础统计：汉字逐字、其他字母/数字连续串逐词。
func Analyze(markdown string) Stats {
	text := frontMatterPattern.ReplaceAllString(markdown, "")
	text = imagePattern.ReplaceAllString(text, "$1")
	text = linkPattern.ReplaceAllString(text, "$1")
	text = htmlPattern.ReplaceAllString(text, " ")
	text = urlPattern.ReplaceAllString(text, " ")
	text = strings.Map(func(r rune) rune {
		switch r {
		case '#', '*', '_', '~', '`', '>', '|', '[', ']', '(', ')', '{', '}':
			return ' '
		default:
			return r
		}
	}, text)

	hanCount := 0
	wordCount := 0
	inWord := false
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			hanCount++
			inWord = false
			continue
		}
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			if !inWord {
				wordCount++
				inWord = true
			}
			continue
		}
		inWord = false
	}

	total := hanCount + wordCount
	minutes := 0
	if strings.TrimSpace(text) != "" {
		minutes = int(math.Ceil(float64(hanCount)/400 + float64(wordCount)/200))
		if minutes < 1 {
			minutes = 1
		}
	}
	return Stats{WordCount: total, ReadingTimeMinutes: minutes}
}
