package article

import (
	"testing"
	"time"

	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
)

func TestArticleListCacheDeadlineTracksPublication(t *testing.T) {
	now := time.Now()
	soon := now.Add(20 * time.Second)
	later := now.Add(time.Hour)
	past := now.Add(-time.Second)
	for _, tc := range []struct {
		name string
		next *time.Time
		want time.Time
	}{
		{"no scheduled articles", nil, now.Add(2 * time.Minute)},
		{"upcoming publication", &soon, soon},
		{"outside cache lifetime", &later, now.Add(2 * time.Minute)},
		{"publication crossed during list query", &past, past},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := articleListCacheDeadline(now, tc.next); !got.Equal(tc.want) {
				t.Fatalf("deadline = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestExpiredListIsNotWrittenAsPermanentCache(t *testing.T) {
	// nil ServiceContext makes any attempted cache access fail immediately.
	var service *svc.ServiceContext
	setCachedArticleList(t.Context(), service, "key", &types.ArticleListResp{}, time.Now().Add(-time.Second))
	setCachedArticleList(t.Context(), service, "key", &types.ArticleListResp{}, time.Time{})
}
