package main

import (
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
