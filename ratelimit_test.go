package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterAllow(t *testing.T) {
	rl := newRateLimiter(3, time.Minute)
	ip := "192.168.1.1"

	for i := 0; i < 3; i++ {
		allowed, _, _ := rl.allow(ip)
		if !allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	allowed, _, _ := rl.allow(ip)
	if allowed {
		t.Fatal("4th request should be denied")
	}
}

func TestRateLimiterDifferentIPs(t *testing.T) {
	rl := newRateLimiter(1, time.Minute)

	allowed1, _, _ := rl.allow("1.1.1.1")
	allowed2, _, _ := rl.allow("2.2.2.2")

	if !allowed1 || !allowed2 {
		t.Fatal("different IPs should have separate limits")
	}
}

func TestRateLimiterWindowReset(t *testing.T) {
	rl := newRateLimiter(1, 50*time.Millisecond)
	ip := "192.168.1.1"

	allowed, _, _ := rl.allow(ip)
	if !allowed {
		t.Fatal("first request should be allowed")
	}

	allowed, _, _ = rl.allow(ip)
	if allowed {
		t.Fatal("second request should be denied")
	}

	time.Sleep(60 * time.Millisecond)

	allowed, _, _ = rl.allow(ip)
	if !allowed {
		t.Fatal("request after window reset should be allowed")
	}
}

func TestRateLimiterRemaining(t *testing.T) {
	rl := newRateLimiter(3, time.Minute)
	ip := "192.168.1.1"

	_, remaining, _ := rl.allow(ip)
	if remaining != 2 {
		t.Fatalf("expected 2 remaining, got %d", remaining)
	}

	_, remaining, _ = rl.allow(ip)
	if remaining != 1 {
		t.Fatalf("expected 1 remaining, got %d", remaining)
	}

	_, remaining, _ = rl.allow(ip)
	if remaining != 0 {
		t.Fatalf("expected 0 remaining, got %d", remaining)
	}

	_, remaining, _ = rl.allow(ip)
	if remaining != 0 {
		t.Fatalf("expected 0 remaining after exhaustion, got %d", remaining)
	}
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		forwarded  string
		realIP     string
		expected   string
	}{
		{"RemoteAddr", "192.168.1.1:12345", "", "", "192.168.1.1"},
		{"X-Forwarded-For", "10.0.0.1:12345", "203.0.113.50, 70.41.3.18", "", "203.0.113.50"},
		{"X-Real-IP", "10.0.0.1:12345", "", "203.0.113.99", "203.0.113.99"},
		{"X-Forwarded-For優先", "10.0.0.1:12345", "203.0.113.50", "203.0.113.99", "203.0.113.50"},
		{"ポートなしRemoteAddr", "192.168.1.1", "", "", "192.168.1.1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = tt.remoteAddr
			if tt.forwarded != "" {
				r.Header.Set("X-Forwarded-For", tt.forwarded)
			}
			if tt.realIP != "" {
				r.Header.Set("X-Real-IP", tt.realIP)
			}
			got := clientIP(r)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestRateLimitMiddlewareGETBypass(t *testing.T) {
	rl := newRateLimiter(1, time.Minute)
	handler := rateLimitMiddleware(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GET request %d should bypass rate limit, got %d", i+1, w.Code)
		}
	}
}

func TestRateLimitMiddlewarePOSTLimited(t *testing.T) {
	rl := newRateLimiter(2, time.Minute)
	handler := rateLimitMiddleware(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/lists", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("POST request %d should be allowed, got %d", i+1, w.Code)
		}
	}

	req := httptest.NewRequest("POST", "/lists", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("3rd POST should be rate limited, got %d", w.Code)
	}
}

func TestRateLimitHeaders(t *testing.T) {
	rl := newRateLimiter(5, time.Minute)
	handler := rateLimitMiddleware(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/lists", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("X-RateLimit-Limit") != "5" {
		t.Errorf("expected X-RateLimit-Limit=5, got %s", w.Header().Get("X-RateLimit-Limit"))
	}
	if w.Header().Get("X-RateLimit-Remaining") != "4" {
		t.Errorf("expected X-RateLimit-Remaining=4, got %s", w.Header().Get("X-RateLimit-Remaining"))
	}
	if w.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("expected X-RateLimit-Reset to be set")
	}
}

func TestRateLimiterConcurrentAccess(t *testing.T) {
	rl := newRateLimiter(100, time.Minute)
	done := make(chan bool, 50)

	for i := 0; i < 50; i++ {
		go func() {
			rl.allow("192.168.1.1")
			done <- true
		}()
	}

	for i := 0; i < 50; i++ {
		<-done
	}
}
