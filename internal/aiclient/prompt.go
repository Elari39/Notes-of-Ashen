package aiclient

import "strings"

func systemPrompt(action string) string {
	switch action {
	case "metadata":
		return `你是博客文章 SEO 编辑助手。必须只输出 json，不要输出 Markdown。
JSON 字段必须为 summary、seoDescription、seoKeywords。
summary 约 100 个中文字符，提炼文章核心观点，不要写成标题；seoDescription 不超过 180 字；seoKeywords 为逗号分隔关键词。
示例 JSON：{"summary":"本文围绕主题提炼核心内容，保留关键背景、问题与结论，方便读者快速判断是否继续阅读。","seoDescription":"文章内容摘要。","seoKeywords":"关键词一,关键词二"}`
	case "proofread":
		return `你是中文博客文章校对助手。必须只输出 json，不要输出 Markdown。
JSON 字段必须为 revisedContent、suggestions。
保留原意和 Markdown 结构，只修正错别字、病句、标点和明显语法问题。
示例 JSON：{"revisedContent":"修订后的 Markdown 正文","suggestions":["修改说明"]}`
	case "polish":
		return `你是中文博客文章润色助手。必须只输出 json，不要输出 Markdown。
JSON 字段必须为 revisedContent、suggestions。
保留原意和 Markdown 结构，让表达更清晰自然，不要扩写事实。
示例 JSON：{"revisedContent":"润色后的 Markdown 正文","suggestions":["修改说明"]}`
	case "expand":
		return `你是中文博客文章伴写助手。必须只输出 json，不要输出 Markdown。
JSON 字段必须为 revisedContent、suggestions。
在不虚构事实的前提下扩写用户给出的段落，让论述更完整、衔接更自然，并保留 Markdown 结构。
示例 JSON：{"revisedContent":"扩写后的 Markdown 段落","suggestions":["扩写说明"]}`
	case "shorten":
		return `你是中文博客文章压缩助手。必须只输出 json，不要输出 Markdown。
JSON 字段必须为 revisedContent、suggestions。
保留核心信息和语气，删去冗余表达，让段落更短更清晰，并保留 Markdown 结构。
示例 JSON：{"revisedContent":"缩写后的 Markdown 段落","suggestions":["缩写说明"]}`
	case "translate":
		return `你是技术博客翻译助手。必须只输出 json，不要输出 Markdown。
JSON 字段必须为 revisedContent、suggestions。
将用户给出的段落翻译为自然英文，保留 Markdown 结构、代码、链接和专有名词；如果原文主要是英文，则翻译为中文。
示例 JSON：{"revisedContent":"Translated Markdown paragraph","suggestions":["翻译说明"]}`
	default:
		return `你是博客文章编辑助手。必须只输出 json，不要输出 Markdown。`
	}
}

func userPrompt(req Request) string {
	var builder strings.Builder
	if strings.TrimSpace(req.Title) != "" {
		builder.WriteString("标题：")
		builder.WriteString(strings.TrimSpace(req.Title))
		builder.WriteString("\n\n")
	}
	builder.WriteString("正文：\n")
	builder.WriteString(req.Content)
	return builder.String()
}

func maxTokens(action string) int {
	switch action {
	case "metadata":
		return 800
	case "proofread", "polish":
		return 12000
	case "expand", "shorten", "translate":
		return 4000
	default:
		return 4000
	}
}
