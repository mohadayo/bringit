package main

import (
	"os"
	"sync"
	"testing"
)

func TestNewStore(t *testing.T) {
	s := NewStore()
	if s == nil {
		t.Fatal("NewStore returned nil")
	}
	if s.lists == nil {
		t.Fatal("store.lists is nil")
	}
}

func TestCreateListAndGetList(t *testing.T) {
	s := NewStore()
	l := s.CreateList("テスト", "説明")
	if l == nil {
		t.Fatal("CreateList returned nil")
	}
	if l.Title != "テスト" {
		t.Fatalf("expected title=テスト, got %s", l.Title)
	}
	if l.Description != "説明" {
		t.Fatalf("expected description=説明, got %s", l.Description)
	}
	if l.ShareToken == "" {
		t.Fatal("ShareToken must not be empty")
	}
	if l.Items == nil {
		t.Fatal("Items must be initialized (not nil)")
	}

	got := s.GetList(l.ShareToken)
	if got == nil {
		t.Fatal("GetList returned nil for valid token")
	}
	if got.Title != l.Title {
		t.Fatalf("expected title=%s, got %s", l.Title, got.Title)
	}
}

func TestGetListNonExistent(t *testing.T) {
	s := NewStore()
	if s.GetList("nonexistent") != nil {
		t.Fatal("expected nil for non-existent token")
	}
}

func TestAddItemToNonExistentList(t *testing.T) {
	s := NewStore()
	item := s.AddItem("nonexistent", "アイテム", "", true)
	if item != nil {
		t.Fatal("expected nil when adding item to non-existent list")
	}
}

func TestAddItemAndRetrieve(t *testing.T) {
	s := NewStore()
	l := s.CreateList("リスト", "")
	item := s.AddItem(l.ShareToken, "テントが必要", "太郎", true)
	if item == nil {
		t.Fatal("AddItem returned nil")
	}
	if item.Name != "テントが必要" {
		t.Fatalf("expected name=テントが必要, got %s", item.Name)
	}
	if item.Assignee != "太郎" {
		t.Fatalf("expected assignee=太郎, got %s", item.Assignee)
	}
	if !item.Required {
		t.Fatal("expected Required=true")
	}
	if item.Prepared {
		t.Fatal("expected Prepared=false initially")
	}
}

func TestDeleteListReturnValues(t *testing.T) {
	s := NewStore()
	l := s.CreateList("削除テスト", "")
	token := l.ShareToken

	// 存在するリストの削除 → true
	if !s.DeleteList(token) {
		t.Fatal("expected DeleteList to return true for existing list")
	}
	// 再度削除 → false
	if s.DeleteList(token) {
		t.Fatal("expected DeleteList to return false for already-deleted list")
	}
	// GetListは nil を返す
	if s.GetList(token) != nil {
		t.Fatal("expected nil after deletion")
	}
}

func TestUniqueTokens(t *testing.T) {
	s := NewStore()
	tokens := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		l := s.CreateList("リスト", "")
		if _, dup := tokens[l.ShareToken]; dup {
			t.Fatalf("duplicate token generated: %s", l.ShareToken)
		}
		tokens[l.ShareToken] = struct{}{}
	}
}

func TestConcurrentCreateList(t *testing.T) {
	s := NewStore()
	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	tokens := make([]string, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			l := s.CreateList("並行テスト", "")
			tokens[idx] = l.ShareToken
		}(i)
	}
	wg.Wait()

	// 全トークンが取得可能であることを確認
	for _, tok := range tokens {
		if s.GetList(tok) == nil {
			t.Fatalf("list with token %s not found after concurrent creation", tok)
		}
	}
}

func TestConcurrentAddItem(t *testing.T) {
	s := NewStore()
	l := s.CreateList("並行アイテム追加", "")
	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			s.AddItem(l.ShareToken, "アイテム", "", true)
		}()
	}
	wg.Wait()

	got := s.GetList(l.ShareToken)
	if len(got.Items) != goroutines {
		t.Fatalf("expected %d items, got %d", goroutines, len(got.Items))
	}
}

func TestStoreStats(t *testing.T) {
	s := NewStore()

	listCount, itemCount := s.Stats()
	if listCount != 0 || itemCount != 0 {
		t.Fatalf("expected 0/0, got %d/%d", listCount, itemCount)
	}

	l1 := s.CreateList("リスト1", "")
	s.AddItem(l1.ShareToken, "アイテム1", "", true)
	s.AddItem(l1.ShareToken, "アイテム2", "", false)

	l2 := s.CreateList("リスト2", "")
	s.AddItem(l2.ShareToken, "アイテム3", "", true)

	listCount, itemCount = s.Stats()
	if listCount != 2 {
		t.Fatalf("expected 2 lists, got %d", listCount)
	}
	if itemCount != 3 {
		t.Fatalf("expected 3 items, got %d", itemCount)
	}

	s.DeleteList(l1.ShareToken)
	listCount, itemCount = s.Stats()
	if listCount != 1 {
		t.Fatalf("expected 1 list after deletion, got %d", listCount)
	}
	if itemCount != 1 {
		t.Fatalf("expected 1 item after deletion, got %d", itemCount)
	}
}

func TestGetEnv(t *testing.T) {
	// 環境変数が設定されている場合
	os.Setenv("TEST_VAR", "hello")
	defer os.Unsetenv("TEST_VAR")
	if v := getEnv("TEST_VAR", "default"); v != "hello" {
		t.Fatalf("expected hello, got %s", v)
	}

	// 環境変数が未設定の場合
	os.Unsetenv("TEST_VAR")
	if v := getEnv("TEST_VAR", "default"); v != "default" {
		t.Fatalf("expected default, got %s", v)
	}
}
