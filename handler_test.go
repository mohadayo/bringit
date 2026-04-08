package main

import (
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
