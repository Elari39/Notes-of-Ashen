package rag

import "testing"

func TestNormalizeMarkdownRemovesMultilineFrontMatter(t *testing.T) {
	content := "\ufeff---\n" +
		"title: 测试文章\n" +
		"tags:\n" +
		"  - rag\n" +
		"summary: |\n" +
		"  这是一段多行摘要\n" +
		"---\n\n" +
		"# 正文\n\n" +
		"保留这一段。\n"

	got := NormalizeMarkdown(content)
	want := "# 正文\n\n保留这一段。"
	if got != want {
		t.Fatalf("NormalizeMarkdown() = %q, want %q", got, want)
	}
}

func TestNormalizeMarkdownPreservesFencedCodeWhitespace(t *testing.T) {
	content := "# 示例\n\n" +
		"普通   文本   会被压缩。   \n\n" +
		"```go\n" +
		"func main() {    \n" +
		"\t//  缩进与连续空白必须保留    \n" +
		"\n\n" +
		"\tfmt.Println(\"![不是图片](https://example.test/x.png)\")\n" +
		"}\n" +
		"```\n\n" +
		"结尾。"

	got := NormalizeMarkdown(content)
	want := "# 示例\n\n" +
		"普通 文本 会被压缩。\n\n" +
		"```go\n" +
		"func main() {    \n" +
		"\t//  缩进与连续空白必须保留    \n" +
		"\n\n" +
		"\tfmt.Println(\"![不是图片](https://example.test/x.png)\")\n" +
		"}\n" +
		"```\n\n" +
		"结尾。"
	if got != want {
		t.Fatalf("NormalizeMarkdown() = %q, want %q", got, want)
	}
}

func TestNormalizeMarkdownKeepsUnclosedFrontMatter(t *testing.T) {
	content := "---\n这不是有效 front matter\n正文\n"
	if got := NormalizeMarkdown(content); got != "---\n这不是有效 front matter\n正文" {
		t.Fatalf("NormalizeMarkdown() unexpectedly drops unclosed front matter: %q", got)
	}
}

func TestSplitMarkdownDoesNotBreakFencedCodeAtBlankLines(t *testing.T) {
	content := "# 示例\n\n```go\nfunc main() {\n\n\tfmt.Println(\"ok\")\n}\n```\n\n后续说明"
	chunks := SplitMarkdown(content)
	if len(chunks) != 1 {
		t.Fatalf("SplitMarkdown() chunk count = %d, want 1: %#v", len(chunks), chunks)
	}
	want := "示例\n```go\nfunc main() {\n\n\tfmt.Println(\"ok\")\n}\n```\n\n示例\n后续说明"
	if chunks[0] != want {
		t.Fatalf("SplitMarkdown() = %q, want %q", chunks[0], want)
	}
}
