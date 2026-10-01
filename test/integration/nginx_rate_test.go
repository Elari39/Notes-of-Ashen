//go:build integration

package integration

import (
	"net/http"
	"sync"
	"testing"
)

// Run against the unmodified production 10r/s, burst 20 Nginx configuration.
func TestStrictNginxRateLimitStatus(t *testing.T) {
	env := loadEnvironment(t)
	client := &http.Client{}
	const count = 45
	statuses := make([]int, count)
	errs := make([]error, count)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range count {
		wg.Go(func() {
			<-start
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, env.webBaseURL+"/api/v1/site/settings", nil)
			if err != nil {
				errs[i] = err
				return
			}
			resp, err := client.Do(req)
			if err != nil {
				errs[i] = err
				return
			}
			defer resp.Body.Close()
			statuses[i] = resp.StatusCode
		})
	}
	close(start)
	wg.Wait()
	counts := make(map[int]int)
	for i, status := range statuses {
		if errs[i] != nil {
			t.Fatalf("request %d: %v", i, errs[i])
		}
		if status != http.StatusOK && status != http.StatusTooManyRequests {
			t.Fatalf("request %d returned %d, want 200 or 429", i, status)
		}
		counts[status]++
	}
	if counts[http.StatusOK] == 0 || counts[http.StatusTooManyRequests] == 0 {
		t.Fatalf("burst did not exercise the production limit: %v", counts)
	}
	t.Logf("production rate limit statuses: %v", counts)
}
