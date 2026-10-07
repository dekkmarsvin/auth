package infra

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// DefaultTurnstileEndpoint is Cloudflare's token validation endpoint.
const DefaultTurnstileEndpoint = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// turnstileMaxTokenLength rejects oversized tokens before they reach Cloudflare.
const turnstileMaxTokenLength = 2048

const (
	turnstileReplayTTL      = 5 * time.Minute
	turnstileReplayMaxItems = 16384
)

var (
	// ErrTurnstileInvalid is returned when Cloudflare rejects the token, or when
	// the token was minted for another action or frontend hostname.
	ErrTurnstileInvalid = errors.New("turnstile token is invalid")
	// ErrTurnstileUnavailable is returned when the token could not be checked.
	ErrTurnstileUnavailable = errors.New("turnstile verification is unavailable")
	// ErrTurnstileReplay distinguishes a local duplicate from a provider rejection
	// while preserving the existing invalid-token HTTP contract.
	ErrTurnstileReplay = fmt.Errorf("%w: timeout-or-duplicate (local replay)", ErrTurnstileInvalid)
)

// TurnstileVerifier validates Turnstile tokens against siteverify. The token is
// rejected unless Cloudflare returns success, an allowed frontend hostname, and
// the action it was minted for. This mirrors the canonical siteverify contract.
type TurnstileVerifier interface {
	Verify(ctx context.Context, token string, remoteIp string, expectedAction string) error
}

type turnstileVerifier struct {
	secret    string
	endpoint  string
	client    *http.Client
	hostnames map[string]struct{}

	replayMu    sync.Mutex
	replay      map[[sha256.Size]byte]*turnstileClaim
	replayLimit int
	now         func() time.Time
}

type turnstileClaim struct {
	expiresAt time.Time
}

// claimToken reserves a hash before the outbound request so concurrent replays
// cannot both reach Siteverify. The single production API uses one verifier.
func (v *turnstileVerifier) claimToken(token string) ([sha256.Size]byte, *turnstileClaim, error) {
	key := sha256.Sum256([]byte(token))
	v.replayMu.Lock()
	defer v.replayMu.Unlock()
	now := v.now()
	for hash, claim := range v.replay {
		if !now.Before(claim.expiresAt) {
			delete(v.replay, hash)
		}
	}
	if _, exists := v.replay[key]; exists {
		return key, nil, ErrTurnstileReplay
	}
	if len(v.replay) >= v.replayLimit {
		return key, nil, fmt.Errorf("%w: replay guard capacity reached", ErrTurnstileUnavailable)
	}
	claim := &turnstileClaim{expiresAt: now.Add(turnstileReplayTTL)}
	v.replay[key] = claim
	return key, claim, nil
}

func (v *turnstileVerifier) releaseToken(key [sha256.Size]byte, claim *turnstileClaim) {
	v.replayMu.Lock()
	defer v.replayMu.Unlock()
	// An expired pending request must not delete a newer claim for this hash.
	if v.replay[key] == claim {
		delete(v.replay, key)
	}
}

// TurnstileOption customises a Turnstile verifier.
type TurnstileOption func(*turnstileVerifier)

// WithTurnstileEndpoint overrides the siteverify URL. It exists so tests can
// point the verifier at a stub instead of Cloudflare.
func WithTurnstileEndpoint(endpoint string) TurnstileOption {
	return func(v *turnstileVerifier) {
		if endpoint != "" {
			v.endpoint = endpoint
		}
	}
}

func resolveTurnstileHostnames(hostnames []string) map[string]struct{} {
	resolved := make(map[string]struct{}, len(hostnames))
	for _, hostname := range hostnames {
		trimmed := strings.ToLower(strings.TrimSpace(hostname))
		if trimmed != "" {
			resolved[trimmed] = struct{}{}
		}
	}
	return resolved
}

// allowsHostname matches exact names or leading *. patterns. Wildcards cover
// one or more subdomain levels, but never the base domain itself.
func (v *turnstileVerifier) allowsHostname(hostname string) bool {
	hostname = strings.ToLower(hostname)
	if _, ok := v.hostnames[hostname]; ok {
		return true
	}
	for pattern := range v.hostnames {
		base, wildcard := strings.CutPrefix(pattern, "*.")
		if !wildcard || base == "" || strings.Contains(base, "*") {
			continue
		}
		suffix := "." + base
		if len(hostname) > len(suffix) && strings.HasSuffix(hostname, suffix) {
			return true
		}
	}
	return false
}

// NewTurnstileVerifier creates a verifier. It always fails closed: a missing
// secret, an empty hostname allowlist, or an unreachable siteverify all reject
// the request rather than letting it through.
func NewTurnstileVerifier(secret string, hostnames []string, options ...TurnstileOption) TurnstileVerifier {
	verifier := &turnstileVerifier{
		secret:      secret,
		endpoint:    DefaultTurnstileEndpoint,
		client:      &http.Client{},
		hostnames:   resolveTurnstileHostnames(hostnames),
		replay:      make(map[[sha256.Size]byte]*turnstileClaim),
		replayLimit: turnstileReplayMaxItems,
		now:         time.Now,
	}
	for _, option := range options {
		option(verifier)
	}
	if verifier.secret == "" {
		slog.Warn("TURNSTILE_SECRET is not set, OTP requests will be rejected")
	}
	if len(verifier.hostnames) == 0 {
		slog.Warn("TURNSTILE_HOSTNAMES is not set, OTP requests will be rejected")
	}
	return verifier
}

type turnstileResponse struct {
	Success    bool     `json:"success"`
	Action     string   `json:"action"`
	Hostname   string   `json:"hostname"`
	ErrorCodes []string `json:"error-codes"`
}

func (v *turnstileVerifier) Verify(ctx context.Context, token string, remoteIp string, expectedAction string) error {
	if v.secret == "" || len(v.hostnames) == 0 {
		return ErrTurnstileUnavailable
	}
	if token == "" {
		return ErrTurnstileInvalid
	}
	if len(token) > turnstileMaxTokenLength {
		return fmt.Errorf("%w: token exceeds %d characters", ErrTurnstileInvalid, turnstileMaxTokenLength)
	}
	key, claim, err := v.claimToken(token)
	if err != nil {
		return err
	}
	consumed := false
	defer func() {
		if !consumed {
			v.releaseToken(key, claim)
		}
	}()

	form := url.Values{}
	form.Set("secret", v.secret)
	form.Set("response", token)
	if remoteIp != "" {
		form.Set("remoteip", remoteIp)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		v.endpoint,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return fmt.Errorf("%w: build siteverify request: %w", ErrTurnstileUnavailable, err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := v.client.Do(request)
	if err != nil {
		return fmt.Errorf("%w: call siteverify: %w", ErrTurnstileUnavailable, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: siteverify returned status %d", ErrTurnstileUnavailable, response.StatusCode)
	}

	var result turnstileResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return fmt.Errorf("%w: decode siteverify response: %w", ErrTurnstileUnavailable, err)
	}
	if !result.Success {
		for _, code := range result.ErrorCodes {
			switch code {
			case "internal-error", "missing-input-secret", "invalid-input-secret":
				return fmt.Errorf("%w: %s", ErrTurnstileUnavailable, strings.Join(result.ErrorCodes, ","))
			}
		}
		return fmt.Errorf("%w: %s", ErrTurnstileInvalid, strings.Join(result.ErrorCodes, ","))
	}
	// A successful Siteverify consumes the token even when local action/hostname
	// checks reject it. Retain the claim so retrying cannot change its purpose.
	consumed = true
	if !v.allowsHostname(result.Hostname) {
		return fmt.Errorf("%w: hostname %q is not allowed", ErrTurnstileInvalid, result.Hostname)
	}
	if result.Action != expectedAction {
		return fmt.Errorf("%w: action %q does not match %q", ErrTurnstileInvalid, result.Action, expectedAction)
	}
	return nil
}
