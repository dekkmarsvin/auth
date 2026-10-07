package infra

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	testHostname = "n.novelia.cc"
	testAction   = "signup"
)

// siteverifyStub records the received form and replies with the given body.
func siteverifyStub(t *testing.T, status int, body string) (*httptest.Server, *map[string]string) {
	t.Helper()
	received := &map[string]string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		values := map[string]string{}
		for key := range r.PostForm {
			values[key] = r.PostForm.Get(key)
		}
		*received = values
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if _, err := io.WriteString(w, body); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	t.Cleanup(server.Close)
	return server, received
}

// successBody is the siteverify reply for a token minted by an allowed frontend.
func successBody(hostname string, action string) string {
	return `{"success":true,"hostname":"` + hostname + `","action":"` + action + `"}`
}

func newTestVerifier(secret string, endpoint string, hostnames ...string) TurnstileVerifier {
	if hostnames == nil {
		hostnames = []string{testHostname}
	}
	return NewTurnstileVerifier(
		secret,
		hostnames,
		WithTurnstileEndpoint(endpoint),
	)
}

func TestTurnstileVerifySuccess(t *testing.T) {
	server, received := siteverifyStub(t, http.StatusOK, successBody(testHostname, testAction))
	verifier := newTestVerifier("secret", server.URL)

	if err := verifier.Verify(t.Context(), "token-abc", "192.0.2.10", testAction); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if (*received)["secret"] != "secret" {
		t.Fatalf("expected secret to be sent, got %q", (*received)["secret"])
	}
	if (*received)["response"] != "token-abc" {
		t.Fatalf("expected response token to be sent, got %q", (*received)["response"])
	}
	if (*received)["remoteip"] != "192.0.2.10" {
		t.Fatalf("expected remoteip to be sent, got %q", (*received)["remoteip"])
	}
}

func TestTurnstileVerifyOmitsEmptyRemoteIp(t *testing.T) {
	server, received := siteverifyStub(t, http.StatusOK, successBody(testHostname, testAction))
	verifier := newTestVerifier("secret", server.URL)

	if err := verifier.Verify(t.Context(), "token-abc", "", testAction); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if _, ok := (*received)["remoteip"]; ok {
		t.Fatal("expected remoteip to be omitted when empty")
	}
}

func TestTurnstileVerifyRejectsInvalidToken(t *testing.T) {
	server, _ := siteverifyStub(t, http.StatusOK, `{"success":false,"error-codes":["invalid-input-response"]}`)
	verifier := newTestVerifier("secret", server.URL)

	err := verifier.Verify(t.Context(), "bad-token", "", testAction)
	if !errors.Is(err, ErrTurnstileInvalid) {
		t.Fatalf("expected ErrTurnstileInvalid, got %v", err)
	}
}

func TestTurnstileVerifyRejectsRedeemedToken(t *testing.T) {
	server, _ := siteverifyStub(t, http.StatusOK, `{"success":false,"error-codes":["timeout-or-duplicate"]}`)
	verifier := newTestVerifier("secret", server.URL)
	err := verifier.Verify(t.Context(), "redeemed-elsewhere", "", testAction)
	if !errors.Is(err, ErrTurnstileInvalid) || errors.Is(err, ErrTurnstileReplay) || !strings.Contains(err.Error(), "timeout-or-duplicate") {
		t.Fatalf("unexpected provider duplicate result: %v", err)
	}
}

func TestTurnstileAcceptedTokenRejectsReplayWithoutProvider(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		// The guard must work even if the provider accepts the replay again.
		io.WriteString(w, successBody("auth.kotoban.top", testAction))
	}))
	defer server.Close()
	verifier := newTestVerifier("fixture-secret", server.URL, "auth.kotoban.top")
	if err := verifier.Verify(t.Context(), "single-use-token", "", testAction); err != nil {
		t.Fatalf("fresh token failed: %v", err)
	}
	if err := verifier.Verify(t.Context(), "single-use-token", "", testAction); !errors.Is(err, ErrTurnstileReplay) || !errors.Is(err, ErrTurnstileInvalid) {
		t.Fatalf("redeemed token was accepted: %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("replay reached the provider: %d requests", requests.Load())
	}
}

func TestTurnstileConcurrentReplayCallsProviderOnce(t *testing.T) {
	var requests atomic.Int32
	entered := make(chan struct{})
	proceed := make(chan struct{})
	var release sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			close(entered)
		}
		<-proceed
		io.WriteString(w, successBody(testHostname, testAction))
	}))
	t.Cleanup(server.Close)
	t.Cleanup(func() { release.Do(func() { close(proceed) }) })
	verifier := newTestVerifier("secret", server.URL)
	first := make(chan error, 1)
	go func() { first <- verifier.Verify(t.Context(), "pending-token", "", testAction) }()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("first request did not reach the provider")
	}
	const duplicates = 16
	results := make(chan error, duplicates)
	var workers sync.WaitGroup
	for i := 0; i < duplicates; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			results <- verifier.Verify(t.Context(), "pending-token", "", testAction)
		}()
	}
	workers.Wait()
	close(results)
	for err := range results {
		if !errors.Is(err, ErrTurnstileReplay) {
			t.Fatalf("pending replay was not rejected: %v", err)
		}
	}
	release.Do(func() { close(proceed) })
	if err := <-first; err != nil {
		t.Fatalf("original request failed: %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("concurrent requests reached the provider: %d", requests.Load())
	}
}

func TestTurnstileProviderFailureReleasesClaim(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"HTTP failure", http.StatusBadGateway, `{}`, ErrTurnstileUnavailable},
		{"malformed body", http.StatusOK, `not JSON`, ErrTurnstileUnavailable},
		{"invalid token", http.StatusOK, `{"success":false,"error-codes":["invalid-input-response"]}`, ErrTurnstileInvalid},
		{"invalid secret", http.StatusOK, `{"success":false,"error-codes":["invalid-input-secret"]}`, ErrTurnstileUnavailable},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if requests.Add(1) == 1 {
					w.WriteHeader(test.status)
					io.WriteString(w, test.body)
				} else {
					io.WriteString(w, successBody(testHostname, testAction))
				}
			}))
			defer server.Close()
			verifier := newTestVerifier("secret", server.URL)
			if err := verifier.Verify(t.Context(), "retry-token", "", testAction); !errors.Is(err, test.want) {
				t.Fatalf("unexpected first failure: %v", err)
			}
			if err := verifier.Verify(t.Context(), "retry-token", "", testAction); err != nil {
				t.Fatalf("failed claim was not released: %v", err)
			}
			if requests.Load() != 2 {
				t.Fatalf("expected two provider calls, got %d", requests.Load())
			}
		})
	}
}

func TestTurnstileInvalidTokensDoNotFillReplayGuard(t *testing.T) {
	server, _ := siteverifyStub(t, http.StatusOK, `{"success":false,"error-codes":["invalid-input-response"]}`)
	verifier := newTestVerifier("secret", server.URL).(*turnstileVerifier)
	verifier.replayLimit = 1
	for _, token := range []string{"invalid-one", "invalid-two", "invalid-three"} {
		if err := verifier.Verify(t.Context(), token, "", testAction); !errors.Is(err, ErrTurnstileInvalid) {
			t.Fatalf("invalid token consumed replay capacity: %v", err)
		}
		if len(verifier.replay) != 0 {
			t.Fatal("failed claim was retained")
		}
	}
}

func TestTurnstileSuccessfulProviderMismatchRetainsClaim(t *testing.T) {
	tests := []struct {
		name        string
		hostname    string
		action      string
		retryAction string
	}{
		{"foreign hostname", "evil.example.com", testAction, testAction},
		{"wrong expected action", testHostname, "password_reset", "password_reset"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				io.WriteString(w, successBody(test.hostname, test.action))
			}))
			defer server.Close()
			verifier := newTestVerifier("secret", server.URL)
			if err := verifier.Verify(t.Context(), "mismatched-token", "", testAction); !errors.Is(err, ErrTurnstileInvalid) || errors.Is(err, ErrTurnstileReplay) {
				t.Fatalf("first request did not reject the mismatch: %v", err)
			}
			if err := verifier.Verify(t.Context(), "mismatched-token", "", test.retryAction); !errors.Is(err, ErrTurnstileReplay) {
				t.Fatalf("mismatched token could be repurposed: %v", err)
			}
			if requests.Load() != 1 {
				t.Fatalf("mismatched token reached the provider again: %d", requests.Load())
			}
		})
	}
}

func TestTurnstileReplayExpiry(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		io.WriteString(w, successBody(testHostname, testAction))
	}))
	defer server.Close()
	verifier := newTestVerifier("secret", server.URL).(*turnstileVerifier)
	now := time.Unix(100, 0)
	verifier.now = func() time.Time { return now }
	if err := verifier.Verify(t.Context(), "expiry-token", "", testAction); err != nil {
		t.Fatal(err)
	}
	now = now.Add(turnstileReplayTTL - time.Nanosecond)
	if err := verifier.Verify(t.Context(), "expiry-token", "", testAction); !errors.Is(err, ErrTurnstileReplay) {
		t.Fatalf("claim expired early: %v", err)
	}
	now = now.Add(time.Nanosecond)
	if err := verifier.Verify(t.Context(), "expiry-token", "", testAction); err != nil {
		t.Fatalf("expired claim was not cleaned: %v", err)
	}
	if requests.Load() != 2 || len(verifier.replay) != 1 {
		t.Fatalf("unexpected expiry state: %d provider calls, %d entries", requests.Load(), len(verifier.replay))
	}
}

func TestTurnstileReplayCapacityFailsClosedWithoutEviction(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		io.WriteString(w, successBody(testHostname, testAction))
	}))
	defer server.Close()
	verifier := newTestVerifier("secret", server.URL).(*turnstileVerifier)
	verifier.replayLimit = 2
	now := time.Unix(100, 0)
	verifier.now = func() time.Time { return now }
	for _, token := range []string{"first-token", "second-token"} {
		if err := verifier.Verify(t.Context(), token, "", testAction); err != nil {
			t.Fatal(err)
		}
	}
	if err := verifier.Verify(t.Context(), "third-token", "", testAction); !errors.Is(err, ErrTurnstileUnavailable) {
		t.Fatalf("full guard did not fail closed: %v", err)
	}
	if err := verifier.Verify(t.Context(), "first-token", "", testAction); !errors.Is(err, ErrTurnstileReplay) {
		t.Fatalf("unexpired claim was evicted: %v", err)
	}
	if requests.Load() != 2 || len(verifier.replay) != 2 {
		t.Fatalf("capacity guard changed state: %d requests, %d entries", requests.Load(), len(verifier.replay))
	}
	now = now.Add(turnstileReplayTTL)
	if err := verifier.Verify(t.Context(), "third-token", "", testAction); err != nil {
		t.Fatalf("expired capacity was not reclaimed: %v", err)
	}
	if len(verifier.replay) != 1 {
		t.Fatalf("expired entries remain: %d", len(verifier.replay))
	}
}

func TestTurnstileLateReleasePreservesNewClaim(t *testing.T) {
	verifier := newTestVerifier("secret", "http://unused.example").(*turnstileVerifier)
	now := time.Unix(100, 0)
	verifier.now = func() time.Time { return now }
	key, oldClaim, err := verifier.claimToken("same-token")
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(turnstileReplayTTL)
	_, newClaim, err := verifier.claimToken("same-token")
	if err != nil {
		t.Fatal(err)
	}
	verifier.releaseToken(key, oldClaim)
	if verifier.replay[key] != newClaim {
		t.Fatal("late failure released a newer claim")
	}
	if _, _, err := verifier.claimToken("same-token"); !errors.Is(err, ErrTurnstileReplay) {
		t.Fatalf("new pending claim was not protected: %v", err)
	}
}

func TestTurnstileReplayStateIsPerVerifier(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		io.WriteString(w, successBody(testHostname, testAction))
	}))
	defer server.Close()
	for i := 0; i < 2; i++ {
		verifier := newTestVerifier("secret", server.URL)
		if err := verifier.Verify(t.Context(), "same-token", "", testAction); err != nil {
			t.Fatalf("independent verifier reused global state: %v", err)
		}
	}
	if requests.Load() != 2 {
		t.Fatalf("expected two independent provider calls, got %d", requests.Load())
	}
}

func TestTurnstileVerifyRejectsForeignAction(t *testing.T) {
	// A token minted for another surface on the same widget must not be replayed.
	server, _ := siteverifyStub(t, http.StatusOK, successBody(testHostname, "contact"))
	verifier := newTestVerifier("secret", server.URL)

	err := verifier.Verify(t.Context(), "token-abc", "", testAction)
	if !errors.Is(err, ErrTurnstileInvalid) {
		t.Fatalf("expected ErrTurnstileInvalid, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "action") {
		t.Fatalf("expected the action to be named in the error, got %v", err)
	}
}

func TestTurnstileVerifyRejectsForeignHostname(t *testing.T) {
	// A token minted on another domain that also hosts this widget is rejected.
	server, _ := siteverifyStub(t, http.StatusOK, successBody("evil.example.com", testAction))
	verifier := newTestVerifier("secret", server.URL)

	err := verifier.Verify(t.Context(), "token-abc", "", testAction)
	if !errors.Is(err, ErrTurnstileInvalid) {
		t.Fatalf("expected ErrTurnstileInvalid, got %v", err)
	}
	if err == nil || !strings.Contains(err.Error(), "hostname") {
		t.Fatalf("expected the hostname to be named in the error, got %v", err)
	}
}

func TestTurnstileVerifyMatchesHostnameCaseInsensitively(t *testing.T) {
	server, _ := siteverifyStub(t, http.StatusOK, successBody("N.Novelia.CC", testAction))
	verifier := newTestVerifier("secret", server.URL, "n.novelia.cc")

	if err := verifier.Verify(t.Context(), "token-abc", "", testAction); err != nil {
		t.Fatalf("expected a case-insensitive hostname match, got %v", err)
	}
}

func TestTurnstileVerifyRejectsEmptyTokenWithoutCallingCloudflare(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer server.Close()

	verifier := newTestVerifier("secret", server.URL)

	err := verifier.Verify(t.Context(), "", "", testAction)
	if !errors.Is(err, ErrTurnstileInvalid) {
		t.Fatalf("expected ErrTurnstileInvalid, got %v", err)
	}
	if called {
		t.Fatal("expected siteverify not to be called for an empty token")
	}
}

func TestTurnstileVerifyRejectsOversizedTokenWithoutCallingCloudflare(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer server.Close()

	verifier := newTestVerifier("secret", server.URL)

	err := verifier.Verify(t.Context(), strings.Repeat("a", turnstileMaxTokenLength+1), "", testAction)
	if !errors.Is(err, ErrTurnstileInvalid) {
		t.Fatalf("expected ErrTurnstileInvalid, got %v", err)
	}
	if called {
		t.Fatal("expected siteverify not to be called for an oversized token")
	}
}

func TestTurnstileVerifyFailsClosed(t *testing.T) {
	unreachable := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	unreachableUrl := unreachable.URL
	unreachable.Close()

	tests := []struct {
		name     string
		secret   string
		endpoint func(t *testing.T) string
		hostname []string
		token    string
	}{
		{
			name:     "missing secret",
			secret:   "",
			endpoint: func(t *testing.T) string { return "http://127.0.0.1:1" },
			token:    "token",
		},
		{
			name:     "missing hostname allowlist",
			secret:   "secret",
			endpoint: func(t *testing.T) string { return "http://127.0.0.1:1" },
			hostname: []string{},
			token:    "token",
		},
		{
			name:     "blank hostname allowlist",
			secret:   "secret",
			endpoint: func(t *testing.T) string { return "http://127.0.0.1:1" },
			hostname: []string{"  ", ""},
			token:    "token",
		},
		{
			name:     "endpoint unreachable",
			secret:   "secret",
			endpoint: func(t *testing.T) string { return unreachableUrl },
			token:    "token",
		},
		{
			name:   "unexpected status",
			secret: "secret",
			endpoint: func(t *testing.T) string {
				server, _ := siteverifyStub(t, http.StatusInternalServerError, `{}`)
				return server.URL
			},
			token: "token",
		},
		{
			name:   "malformed body",
			secret: "secret",
			endpoint: func(t *testing.T) string {
				server, _ := siteverifyStub(t, http.StatusOK, `not json`)
				return server.URL
			},
			token: "token",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			verifier := newTestVerifier(test.secret, test.endpoint(t), test.hostname...)

			err := verifier.Verify(t.Context(), test.token, "", testAction)
			if !errors.Is(err, ErrTurnstileUnavailable) {
				t.Fatalf("expected ErrTurnstileUnavailable, got %v", err)
			}
			if len(verifier.(*turnstileVerifier).replay) != 0 {
				t.Fatal("unavailable verification retained a token claim")
			}
		})
	}
}

func TestTurnstileVerifyHostnamePatterns(t *testing.T) {
	tests := []struct {
		name     string
		allowed  []string
		hostname string
		wantOK   bool
	}{
		{"subdomain", []string{"*.novelia.cc"}, "forum.novelia.cc", true},
		{"nested subdomain", []string{"*.novelia.cc"}, "dev.forum.novelia.cc", true},
		{"case and config whitespace", []string{"  *.Novelia.CC  "}, "Forum.NOVELIA.cc", true},
		{"bare domain excluded", []string{"*.novelia.cc"}, "novelia.cc", false},
		{"bare domain explicitly allowed", []string{"novelia.cc", "*.novelia.cc"}, "novelia.cc", true},
		{"mixed exact and wildcard", []string{"auth.example.com", "*.novelia.cc"}, "auth.example.com", true},
		{"exact does not include children", []string{"forum.novelia.cc"}, "dev.forum.novelia.cc", false},
		{"suffix without label boundary", []string{"*.novelia.cc"}, "evilnovelia.cc", false},
		{"appended foreign domain", []string{"*.novelia.cc"}, "forum.novelia.cc.evil.com", false},
		{"unrelated domain", []string{"*.novelia.cc"}, "forum.example.com", false},
		{"empty subdomain", []string{"*.novelia.cc"}, ".novelia.cc", false},
		{"empty hostname", []string{"*.novelia.cc"}, "", false},
		{"bare wildcard unsupported", []string{"*"}, "forum.novelia.cc", false},
		{"empty wildcard base", []string{"*."}, "forum.novelia.cc", false},
		{"partial label wildcard unsupported", []string{"forum*.novelia.cc"}, "forum1.novelia.cc", false},
		{"embedded wildcard unsupported", []string{"*.*.novelia.cc"}, "dev.forum.novelia.cc", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, _ := siteverifyStub(t, http.StatusOK, successBody(test.hostname, testAction))
			verifier := newTestVerifier("secret", server.URL, test.allowed...)
			err := verifier.Verify(t.Context(), "token-abc", "", testAction)
			if test.wantOK {
				if err != nil {
					t.Fatalf("expected hostname %q to be allowed, got %v", test.hostname, err)
				}
			} else if !errors.Is(err, ErrTurnstileInvalid) {
				t.Fatalf("expected ErrTurnstileInvalid for hostname %q, got %v", test.hostname, err)
			}
		})
	}
}
