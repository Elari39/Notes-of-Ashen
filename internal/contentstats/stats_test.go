package contentstats

import "testing"

func TestAnalyze(t *testing.T) {
	tests := []struct {
		name    string
		content string
		words   int
		minutes int
	}{
		{name: "empty", content: "", words: 0, minutes: 0},
		{name: "chinese", content: "这是中文。", words: 4, minutes: 1},
		{name: "latin", content: "hello world 2026", words: 3, minutes: 1},
		{name: "markdown", content: "# 标题\n[OpenAI](https://openai.com) 与 `Go`", words: 5, minutes: 1},
		{name: "punctuation only", content: "……！", words: 0, minutes: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Analyze(tt.content)
			if got.WordCount != tt.words || got.ReadingTimeMinutes != tt.minutes {
				t.Fatalf("Analyze() = %+v, want words=%d minutes=%d", got, tt.words, tt.minutes)
			}
		})
	}
}
