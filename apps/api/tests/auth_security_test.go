//go:build integration

package tests

import (
	"auth/internal/hash"
	"auth/internal/infra"
	"auth/internal/repository"
	authservice "auth/internal/service/auth"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestAuthLoginRejectsBannedUser(t *testing.T) {
	resetDatabase(t)
	createUser(t, "banned-user", "banned@example.com", "Password123!", repository.RoleBanned)

	SendRequestAndExpectError(
		t,
		http.MethodPost,
		"/v1/auth/login",
		map[string]string{
			"app":      infra.AppAuth,
			"username": "banned-user",
			"password": "Password123!",
		},
		http.StatusForbidden,
		"用户已被封禁",
	)
}

func TestAuthRefreshRejectsNewlyBannedUser(t *testing.T) {
	resetDatabase(t)
	user := createUser(t, "refresh-user", "refresh@example.com", "Password123!", repository.RoleMember)
	resp, _ := sendJSON(t, http.MethodPost, "/v1/auth/login", map[string]string{
		"app":      infra.AppAuth,
		"username": user.Username,
		"password": "Password123!",
	}, "192.0.2.20", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("seed login returned status %d", resp.StatusCode)
	}
	refreshCookie := findCookie(resp, authservice.RefreshTokenCookieName)
	if refreshCookie == nil {
		t.Fatal("seed login did not return a refresh token")
	}

	user.Role = repository.RoleBanned
	if err := userRepo.UpdateRole(user); err != nil {
		t.Fatalf("update user role: %v", err)
	}

	refreshResp, body := sendJSON(
		t,
		http.MethodPost,
		"/v1/auth/refresh?app="+infra.AppAuth,
		nil,
		"192.0.2.20",
		refreshCookie,
	)
	if refreshResp.StatusCode != http.StatusForbidden || body != "用户已被封禁" {
		t.Fatalf("expected refresh status %d and banned message, got %d and %q", http.StatusForbidden, refreshResp.StatusCode, body)
	}
}

func TestAuthRestrictedUserCanLoginAndRefresh(t *testing.T) {
	resetDatabase(t)
	user := createUser(t, "restricted-user", "restricted@example.com", "Password123!", repository.RoleRestricted)
	resp, body := sendJSON(t, http.MethodPost, "/v1/auth/login", map[string]string{
		"app":      infra.AppAuth,
		"username": user.Username,
		"password": "Password123!",
	}, "192.0.2.21", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("restricted user login returned status %d: %s", resp.StatusCode, body)
	}
	refreshCookie := findCookie(resp, authservice.RefreshTokenCookieName)
	if refreshCookie == nil {
		t.Fatal("restricted user login did not return a refresh token")
	}

	refreshResp, body := sendJSON(
		t,
		http.MethodPost,
		"/v1/auth/refresh?app="+infra.AppAuth,
		nil,
		"192.0.2.21",
		refreshCookie,
	)
	if refreshResp.StatusCode != http.StatusOK {
		t.Fatalf("restricted user refresh returned status %d: %s", refreshResp.StatusCode, body)
	}
}

func TestAuthResetPasswordConsumesOtp(t *testing.T) {
	resetDatabase(t)
	user := createUser(t, "reset-user", "reset@example.com", "Password123!", repository.RoleMember)
	otp, err := otpRepo.SetOtp(repository.OtpResetPassword, user.Email)
	if err != nil {
		t.Fatalf("prepare reset OTP: %v", err)
	}
	body := map[string]string{
		"email":    user.Email,
		"otp":      otp,
		"password": "NewPassword123!",
	}

	resp, responseBody := sendJSON(t, http.MethodPost, "/v1/auth/password/reset", body, "192.0.2.30", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first reset returned status %d: %s", resp.StatusCode, responseBody)
	}

	resp, responseBody = sendJSON(t, http.MethodPost, "/v1/auth/password/reset", body, "192.0.2.30", nil)
	if resp.StatusCode != http.StatusUnauthorized || responseBody != "无效的验证码" {
		t.Fatalf("expected reused OTP to be rejected, got status %d: %s", resp.StatusCode, responseBody)
	}
}

func TestAuthOtpRequestIsRateLimited(t *testing.T) {
	resetDatabase(t)
	const ip = "198.51.100.100"

	body := map[string]string{
		"email":          "rate-limit@example.com",
		"type":           repository.OtpVerify,
		"turnstileToken": turnstileGoodToken,
	}
	for i := 0; i < 100; i++ {
		// Each legitimate request solves a fresh challenge. Token replay is
		// covered separately and must not bypass this rate-limit assertion.
		body["turnstileToken"] = fmt.Sprintf("rate-limit-token-%d", i)
		resp, responseBody := sendJSON(t, http.MethodPost, "/v1/auth/otp/request", body, ip, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("OTP request %d unexpectedly returned status %d: %s", i+1, resp.StatusCode, responseBody)
		}
	}

	body["turnstileToken"] = "rate-limit-token-100"
	resp, responseBody := sendJSON(t, http.MethodPost, "/v1/auth/otp/request", body, ip, nil)
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected OTP request 101 to be rate limited, got status %d: %s", resp.StatusCode, responseBody)
	}
}

func TestAuthOtpRequestRejectsReplayedTurnstileToken(t *testing.T) {
	resetDatabase(t)
	body := map[string]string{
		"email":          "replay@example.com",
		"type":           repository.OtpVerify,
		"turnstileToken": "replay-test-token",
	}
	resp, responseBody := sendJSON(t, http.MethodPost, "/v1/auth/otp/request", body, "203.0.113.42", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fresh token returned status %d: %s", resp.StatusCode, responseBody)
	}
	var originalOtp string
	if err := testDB.QueryRow("SELECT encode(code_hash, 'hex') FROM auth_otp WHERE email = $1", body["email"]).Scan(&originalOtp); err != nil {
		t.Fatalf("read original OTP: %v", err)
	}
	resp, responseBody = sendJSON(t, http.MethodPost, "/v1/auth/otp/request", body, "203.0.113.42", nil)
	if resp.StatusCode != http.StatusBadRequest || responseBody != "人机验证失败，请重试" {
		t.Fatalf("replayed token returned status %d: %s", resp.StatusCode, responseBody)
	}
	var currentOtp string
	var events int
	if err := testDB.QueryRow("SELECT encode(code_hash, 'hex') FROM auth_otp WHERE email = $1", body["email"]).Scan(&currentOtp); err != nil {
		t.Fatalf("read OTP after replay: %v", err)
	}
	if err := testDB.QueryRow("SELECT count(*) FROM auth_event WHERE action = $1", authservice.EventOtp).Scan(&events); err != nil {
		t.Fatalf("count OTP events: %v", err)
	}
	if currentOtp != originalOtp || events != 1 {
		t.Fatalf("replay changed OTP or emitted an event: unchanged=%v events=%d", currentOtp == originalOtp, events)
	}
}

func TestAuthOtpRequestRejectsInvalidTurnstileToken(t *testing.T) {
	resetDatabase(t)
	resp, responseBody := sendJSON(t, http.MethodPost, "/v1/auth/otp/request", map[string]string{
		"email":          "invalid@example.com",
		"type":           repository.OtpVerify,
		"turnstileToken": turnstileBadToken,
	}, "203.0.113.41", nil)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid token to be rejected, got status %d: %s", resp.StatusCode, responseBody)
	}
	if responseBody != "人机验证失败，请重试" {
		t.Fatalf("unexpected invalid token response body: %q", responseBody)
	}
}

func createUser(t *testing.T, username, email, password, role string) *repository.User {
	t.Helper()
	passwordHash, err := hash.GenerateHash(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := &repository.User{
		Username:  username,
		Email:     email,
		Role:      role,
		Password:  passwordHash,
		CreatedAt: time.Now(),
		LastLogin: time.Now(),
		Attr:      "{}",
	}
	if err := userRepo.Save(user); err != nil {
		t.Fatalf("save user: %v", err)
	}
	savedUser, err := userRepo.FindByUsername(username)
	if err != nil {
		t.Fatalf("find saved user: %v", err)
	}
	if savedUser == nil {
		t.Fatal("saved user was not found")
	}
	return savedUser
}

func sendJSON(t *testing.T, method, path string, body any, ip string, cookie *http.Cookie) (*http.Response, string) {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("encode request: %v", err)
	}
	req, err := http.NewRequest(method, Url+path, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Real-Ip", ip)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	resp, err := Client.Do(req)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return resp, string(responseBody)
}
