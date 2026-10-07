package infra

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthConfigServesOnlyPublicSiteKey(t *testing.T) {
	w := httptest.NewRecorder()
	NewAuthConfigHandler("  public-site-key  ")(w, httptest.NewRequest(http.MethodGet, "/v1/auth/config", nil))
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("unexpected status/cache policy: %d %q", w.Code, w.Header().Get("Cache-Control"))
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 1 || body["turnstileSiteKey"] != "public-site-key" {
		t.Fatalf("unexpected public configuration: %v", body)
	}
}

func TestAuthConfigFailsClosedWithoutSiteKey(t *testing.T) {
	w := httptest.NewRecorder()
	NewAuthConfigHandler(" \t ")(w, httptest.NewRequest(http.MethodGet, "/v1/auth/config", nil))
	if w.Code != http.StatusServiceUnavailable || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("unexpected status/cache policy: %d %q", w.Code, w.Header().Get("Cache-Control"))
	}
}
