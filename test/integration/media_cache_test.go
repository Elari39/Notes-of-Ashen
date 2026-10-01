//go:build integration

package integration

import (
	"net/http"
	"strconv"
	"testing"
)

func TestCoreMediaDeletionInvalidatesOriginCaches(t *testing.T) {
	h := newHarness(t)
	admin := h.registerUser(t, "mediacache")
	data := onePixelPNG(t)
	asset := h.uploadMedia(t, admin.accessToken, data)
	for range 3 {
		for _, target := range []string{h.apiURL(asset.URL), h.webURL(asset.URL)} {
			response, err := h.request(t.Context(), http.MethodGet, target, nil, nil, "", "")
			if err != nil {
				t.Fatal(err)
			}
			if response.status != http.StatusOK || response.header.Get("Cache-Control") != "no-store" {
				t.Fatalf("media cache policy: status=%d cache=%q", response.status, response.header.Get("Cache-Control"))
			}
		}
	}
	deleted, err := h.jsonRequest(t.Context(), http.MethodDelete, "/api/v1/admin/media/"+strconv.FormatUint(asset.ID, 10), nil, admin.accessToken, "")
	if err != nil {
		t.Fatal(err)
	}
	expectAPISuccess(t, deleted)
	for _, target := range []string{h.apiURL(asset.URL), h.webURL(asset.URL)} {
		response, err := h.request(t.Context(), http.MethodGet, target, nil, nil, "", "")
		if err != nil {
			t.Fatal(err)
		}
		if response.status != http.StatusNotFound || response.header.Get("Cache-Control") != "no-store" {
			t.Fatalf("deleted media remains cacheable: status=%d cache=%q", response.status, response.header.Get("Cache-Control"))
		}
	}
}
