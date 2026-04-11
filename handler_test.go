package main

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func setupTestServer() (*http.ServeMux, *Store) {
	store := NewStore()
	mux := http.NewServeMux()
	registerRoutes(mux, store)
	return mux, store
}

func TestHealthEndpoint(t *testing.T) {
	mux, _ := setupTestServer()
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("expected application/json content-type, got %s", ct)
	}
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status=ok, got %s", body["status"])
	}
	if body["service"] != "bringit" {
		t.Fatalf("expected service=bringit, got %s", body["service"])
	}
	if body["timestamp"] == "" {
		t.Fatal("expected non-empty timestamp")
	}
}

func TestIndexPage(t *testing.T) {
	mux, _ := setupTestServer()
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "BringIt") {
		t.Fatal("expected page to contain BringIt")
	}
}

func TestCreateListAndView(t *testing.T) {
	mux, _ := setupTestServer()

	// Create list
	form := url.Values{"title": {"夏キャンプ"}, "description": {"河口湖"}}
	req := httptest.NewRequest("POST", "/lists", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if !strings.HasPrefix(loc, "/lists/") {
		t.Fatalf("expected redirect to /lists/..., got %s", loc)
	}

	// View list
	req = httptest.NewRequest("GET", loc, nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "夏キャンプ") {
		t.Fatal("expected list title in response")
	}
	if !strings.Contains(body, "河口湖") {
		t.Fatal("expected description in response")
	}
}

func TestAddItemAndToggle(t *testing.T) {
	mux, store := setupTestServer()
	l := store.CreateList("テスト", "")
	token := l.ShareToken

	// Add item
	form := url.Values{"name": {"テント"}, "assignee": {"太郎"}, "required": {"on"}}
	req := httptest.NewRequest("POST", "/lists/"+token+"/items", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}

	// Verify item was added
	l = store.GetList(token)
	if len(l.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(l.Items))
	}
	item := l.Items[0]
	if item.Name != "テント" || item.Assignee != "太郎" || !item.Required || item.Prepared {
		t.Fatalf("unexpected item state: %+v", item)
	}

	// Toggle prepared
	req = httptest.NewRequest("POST", "/lists/"+token+"/items/"+item.ID+"/toggle-prepared", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if !store.GetList(token).Items[0].Prepared {
		t.Fatal("expected item to be prepared after toggle")
	}

	// Toggle required
	req = httptest.NewRequest("POST", "/lists/"+token+"/items/"+item.ID+"/toggle-required", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if store.GetList(token).Items[0].Required {
		t.Fatal("expected item to be optional after toggle")
	}
}

func TestDeleteItem(t *testing.T) {
	mux, store := setupTestServer()
	l := store.CreateList("テスト", "")
	store.AddItem(l.ShareToken, "寝袋", "", true)

	item := store.GetList(l.ShareToken).Items[0]
	req := httptest.NewRequest("POST", "/lists/"+l.ShareToken+"/items/"+item.ID+"/delete", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if len(store.GetList(l.ShareToken).Items) != 0 {
		t.Fatal("expected item to be deleted")
	}
}

func TestUpdateAssignee(t *testing.T) {
	mux, store := setupTestServer()
	l := store.CreateList("テスト", "")
	store.AddItem(l.ShareToken, "クーラーボックス", "太郎", true)

	item := store.GetList(l.ShareToken).Items[0]
	form := url.Values{"assignee": {"花子"}}
	req := httptest.NewRequest("POST", "/lists/"+l.ShareToken+"/items/"+item.ID+"/assignee", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if store.GetList(l.ShareToken).Items[0].Assignee != "花子" {
		t.Fatal("expected assignee to be updated")
	}
}

func TestNotFoundList(t *testing.T) {
	mux, _ := setupTestServer()
	req := httptest.NewRequest("GET", "/lists/nonexistent", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCreateListEmptyTitle(t *testing.T) {
	mux, _ := setupTestServer()
	form := url.Values{"title": {""}}
	req := httptest.NewRequest("POST", "/lists", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateListWhitespaceTitle(t *testing.T) {
	mux, _ := setupTestServer()
	form := url.Values{"title": {"   "}}
	req := httptest.NewRequest("POST", "/lists", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for whitespace-only title, got %d", w.Code)
	}
}

func TestDeleteList(t *testing.T) {
	mux, store := setupTestServer()
	l := store.CreateList("削除テスト", "")
	token := l.ShareToken

	req := httptest.NewRequest("POST", "/lists/"+token+"/delete", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/" {
		t.Fatalf("expected redirect to /, got %s", loc)
	}
	if store.GetList(token) != nil {
		t.Fatal("expected list to be deleted from store")
	}
}

func TestDeleteListNotFound(t *testing.T) {
	mux, _ := setupTestServer()
	req := httptest.NewRequest("POST", "/lists/nonexistent-token/delete", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestAddItemEmptyName(t *testing.T) {
	mux, store := setupTestServer()
	l := store.CreateList("テスト", "")
	token := l.ShareToken

	form := url.Values{"name": {""}, "assignee": {"太郎"}}
	req := httptest.NewRequest("POST", "/lists/"+token+"/items", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Should redirect without adding item
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}
	if len(store.GetList(token).Items) != 0 {
		t.Fatal("expected no items to be added for empty name")
	}
}

func TestAddItemWhitespaceName(t *testing.T) {
	mux, store := setupTestServer()
	l := store.CreateList("テスト", "")
	token := l.ShareToken

	form := url.Values{"name": {"   "}}
	req := httptest.NewRequest("POST", "/lists/"+token+"/items", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}
	if len(store.GetList(token).Items) != 0 {
		t.Fatal("expected no items to be added for whitespace-only name")
	}
}

func TestTogglePreparedInvalidItem(t *testing.T) {
	mux, store := setupTestServer()
	l := store.CreateList("テスト", "")
	token := l.ShareToken

	req := httptest.NewRequest("POST", "/lists/"+token+"/items/nonexistent-id/toggle-prepared", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Handler should redirect gracefully even for invalid item ID
	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect for invalid item ID, got %d", w.Code)
	}
}

func TestToggleRequiredInvalidItem(t *testing.T) {
	mux, store := setupTestServer()
	l := store.CreateList("テスト", "")
	token := l.ShareToken

	req := httptest.NewRequest("POST", "/lists/"+token+"/items/nonexistent-id/toggle-required", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect for invalid item ID, got %d", w.Code)
	}
}

func TestUpdateAssigneeInvalidItem(t *testing.T) {
	mux, store := setupTestServer()
	l := store.CreateList("テスト", "")
	token := l.ShareToken

	form := url.Values{"assignee": {"花子"}}
	req := httptest.NewRequest("POST", "/lists/"+token+"/items/nonexistent-id/assignee", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect for invalid item ID, got %d", w.Code)
	}
}

func TestDeleteItemInvalidItem(t *testing.T) {
	mux, store := setupTestServer()
	l := store.CreateList("テスト", "")
	store.AddItem(l.ShareToken, "アイテム", "", true)
	token := l.ShareToken

	req := httptest.NewRequest("POST", "/lists/"+token+"/items/nonexistent-id/delete", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect for invalid item ID, got %d", w.Code)
	}
	// Existing item should still be there
	if len(store.GetList(token).Items) != 1 {
		t.Fatal("expected existing item to remain after deleting nonexistent ID")
	}
}

func TestIndexPageNotFound(t *testing.T) {
	mux, _ := setupTestServer()
	req := httptest.NewRequest("GET", "/nonexistent-path", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown path, got %d", w.Code)
	}
}

func TestRequestLoggerMiddleware(t *testing.T) {
	mux, _ := setupTestServer()
	handler := requestLogger(mux)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 through middleware, got %d", w.Code)
	}
}

func TestResponseWriterStatusCode(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	rw.WriteHeader(http.StatusNotFound)
	if rw.statusCode != http.StatusNotFound {
		t.Fatalf("expected statusCode=404, got %d", rw.statusCode)
	}
}

func TestTruncateRunes(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello", 3, "hel"},
		{"あいうえお", 3, "あいう"},
		{"", 5, ""},
		{"abc", 0, ""},
		{"テスト文字列", 6, "テスト文字列"},
		{"テスト文字列", 4, "テスト文"},
	}
	for _, tt := range tests {
		result := truncateRunes(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateRunes(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestCreateListTitleTruncation(t *testing.T) {
	mux, store := setupTestServer()

	// 101文字のタイトル（最大100文字に切り詰められる）
	longTitle := strings.Repeat("あ", 101)
	form := url.Values{"title": {longTitle}}
	req := httptest.NewRequest("POST", "/lists", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", w.Code)
	}

	loc := w.Header().Get("Location")
	token := strings.TrimPrefix(loc, "/lists/")
	l := store.GetList(token)
	if l == nil {
		t.Fatal("expected list to be created")
	}
	titleRunes := []rune(l.Title)
	if len(titleRunes) != maxTitleLen {
		t.Fatalf("expected title to be truncated to %d runes, got %d", maxTitleLen, len(titleRunes))
	}
}

func TestCreateListDescriptionTruncation(t *testing.T) {
	mux, store := setupTestServer()

	longDesc := strings.Repeat("x", 501)
	form := url.Values{"title": {"テスト"}, "description": {longDesc}}
	req := httptest.NewRequest("POST", "/lists", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	loc := w.Header().Get("Location")
	token := strings.TrimPrefix(loc, "/lists/")
	l := store.GetList(token)
	if len(l.Description) != maxDescriptionLen {
		t.Fatalf("expected description to be truncated to %d chars, got %d", maxDescriptionLen, len(l.Description))
	}
}

func TestAddItemNameTruncation(t *testing.T) {
	mux, store := setupTestServer()
	l := store.CreateList("テスト", "")
	token := l.ShareToken

	longName := strings.Repeat("い", 150)
	form := url.Values{"name": {longName}, "assignee": {"太郎"}}
	req := httptest.NewRequest("POST", "/lists/"+token+"/items", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	items := store.GetList(token).Items
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	nameRunes := []rune(items[0].Name)
	if len(nameRunes) != maxItemNameLen {
		t.Fatalf("expected item name to be truncated to %d runes, got %d", maxItemNameLen, len(nameRunes))
	}
}

func TestAddItemAssigneeTruncation(t *testing.T) {
	mux, store := setupTestServer()
	l := store.CreateList("テスト", "")
	token := l.ShareToken

	longAssignee := strings.Repeat("う", 80)
	form := url.Values{"name": {"アイテム"}, "assignee": {longAssignee}}
	req := httptest.NewRequest("POST", "/lists/"+token+"/items", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	items := store.GetList(token).Items
	assigneeRunes := []rune(items[0].Assignee)
	if len(assigneeRunes) != maxAssigneeLen {
		t.Fatalf("expected assignee to be truncated to %d runes, got %d", maxAssigneeLen, len(assigneeRunes))
	}
}

func TestUpdateAssigneeTruncation(t *testing.T) {
	mux, store := setupTestServer()
	l := store.CreateList("テスト", "")
	store.AddItem(l.ShareToken, "アイテム", "太郎", true)

	item := store.GetList(l.ShareToken).Items[0]
	longAssignee := strings.Repeat("え", 60)
	form := url.Values{"assignee": {longAssignee}}
	req := httptest.NewRequest("POST", "/lists/"+l.ShareToken+"/items/"+item.ID+"/assignee", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	updated := store.GetList(l.ShareToken).Items[0]
	assigneeRunes := []rune(updated.Assignee)
	if len(assigneeRunes) != maxAssigneeLen {
		t.Fatalf("expected assignee to be truncated to %d runes, got %d", maxAssigneeLen, len(assigneeRunes))
	}
}

func TestSecurityHeaders(t *testing.T) {
	mux, _ := setupTestServer()
	handler := securityHeaders(mux)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	tests := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":       "DENY",
		"Referrer-Policy":       "strict-origin-when-cross-origin",
		"Permissions-Policy":    "camera=(), microphone=(), geolocation=()",
	}
	for header, expected := range tests {
		got := w.Header().Get(header)
		if got != expected {
			t.Errorf("header %s: expected %q, got %q", header, expected, got)
		}
	}
}

func TestBuildShareURL(t *testing.T) {
	tests := []struct {
		name     string
		proto    string // X-Forwarded-Proto
		useTLS   bool
		host     string
		token    string
		expected string
	}{
		{
			name:     "デフォルトHTTP",
			host:     "example.com",
			token:    "abc123",
			expected: "http://example.com/lists/abc123",
		},
		{
			name:     "X-Forwarded-ProtoでHTTPS",
			proto:    "https",
			host:     "example.com",
			token:    "abc123",
			expected: "https://example.com/lists/abc123",
		},
		{
			name:     "TLS接続でHTTPS",
			useTLS:   true,
			host:     "example.com:443",
			token:    "xyz",
			expected: "https://example.com:443/lists/xyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.Host = tt.host
			if tt.proto != "" {
				req.Header.Set("X-Forwarded-Proto", tt.proto)
			}
			if tt.useTLS {
				req.TLS = &tls.ConnectionState{}
			}
			got := buildShareURL(req, tt.token)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
