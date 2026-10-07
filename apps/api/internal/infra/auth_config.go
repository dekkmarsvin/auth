package infra

import (
	"encoding/json"
	"net/http"
	"strings"
)

// NewAuthConfigHandler serves only public runtime configuration. Keeping the
// site key out of the bundle lets deployments use their own widget unchanged.
func NewAuthConfigHandler(siteKey string) http.HandlerFunc {
	siteKey = strings.TrimSpace(siteKey)
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if siteKey == "" {
			http.Error(w, "人机验证服务暂不可用，请稍后再试", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			TurnstileSiteKey string `json:"turnstileSiteKey"`
		}{TurnstileSiteKey: siteKey})
	}
}
