package main

import (
	"errors"
	"os"
	"sync"
	"testing"
)

func TestNewStore(t *testing.T) {
	s := NewStore(0, 0)
	if s == nil {
		t.Fatal("NewStore returned nil")
	}
	if s.lists == nil {
		t.Fatal("store.lists is nil")
	}
}

func TestCreateListAndGetList(t *testing.T) {
	s := NewStore(0, 0)
	l, err := s.CreateList("テスト", "説明")
	if err != nil {
		t.Fatalf("CreateList returned error: %v", err)
	}
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
	s := NewStore(0, 0)
	if s.GetList("nonexistent") != nil {
		t.Fatal("expected nil for non-existent token")
	}
}

func TestAddItemToNonExistentList(t *testing.T) {
	s := NewStore(0, 0)
	item, _ := s.AddItem("nonexistent", "アイテム", "", true)
	if item != nil {
		t.Fatal("expected nil when adding item to non-existent list")
	}
}

func TestAddItemAndRetrieve(t *testing.T) {
	s := NewStore(0, 0)
	l, _ := s.CreateList("リスト", "")
	item, err := s.AddItem(l.ShareToken, "テントが必要", "太郎", true)
	if err != nil {
		t.Fatalf("AddItem returned error: %v", err)
	}
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
	s := NewStore(0, 0)
	l, _ := s.CreateList("削除テスト", "")
	token := l.ShareToken

	if !s.DeleteList(token) {
		t.Fatal("expected DeleteList to return true for existing list")
	}
	if s.DeleteList(token) {
		t.Fatal("expected DeleteList to return false for already-deleted list")
	}
	if s.GetList(token) != nil {
		t.Fatal("expected nil after deletion")
	}
}

func TestUniqueTokens(t *testing.T) {
	s := NewStore(0, 0)
	tokens := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		l, _ := s.CreateList("リスト", "")
		if _, dup := tokens[l.ShareToken]; dup {
			t.Fatalf("duplicate token generated: %s", l.ShareToken)
		}
		tokens[l.ShareToken] = struct{}{}
	}
}

func TestConcurrentCreateList(t *testing.T) {
	s := NewStore(0, 0)
	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	tokens := make([]string, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			l, _ := s.CreateList("並行テスト", "")
			tokens[idx] = l.ShareToken
		}(i)
	}
	wg.Wait()

	for _, tok := range tokens {
		if s.GetList(tok) == nil {
			t.Fatalf("list with token %s not found after concurrent creation", tok)
		}
	}
}

func TestConcurrentAddItem(t *testing.T) {
	s := NewStore(0, 0)
	l, _ := s.CreateList("並行アイテム追加", "")
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
	s := NewStore(0, 0)

	listCount, itemCount := s.Stats()
	if listCount != 0 || itemCount != 0 {
		t.Fatalf("expected 0/0, got %d/%d", listCount, itemCount)
	}

	l1, _ := s.CreateList("リスト1", "")
	s.AddItem(l1.ShareToken, "アイテム1", "", true)
	s.AddItem(l1.ShareToken, "アイテム2", "", false)

	l2, _ := s.CreateList("リスト2", "")
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

func TestMaxListsLimit(t *testing.T) {
	s := NewStore(3, 0)

	for i := 0; i < 3; i++ {
		_, err := s.CreateList("リスト", "")
		if err != nil {
			t.Fatalf("list %d should be allowed, got error: %v", i+1, err)
		}
	}

	_, err := s.CreateList("超過リスト", "")
	if !errors.Is(err, errMaxListsReached) {
		t.Fatalf("expected errMaxListsReached, got %v", err)
	}

	listCount, _ := s.Stats()
	if listCount != 3 {
		t.Fatalf("expected 3 lists, got %d", listCount)
	}
}

func TestMaxListsLimitAfterDelete(t *testing.T) {
	s := NewStore(2, 0)

	l1, _ := s.CreateList("リスト1", "")
	s.CreateList("リスト2", "")

	_, err := s.CreateList("超過リスト", "")
	if err == nil {
		t.Fatal("expected error when exceeding max lists")
	}

	s.DeleteList(l1.ShareToken)
	_, err = s.CreateList("削除後のリスト", "")
	if err != nil {
		t.Fatalf("expected success after delete, got error: %v", err)
	}
}

func TestMaxItemsPerListLimit(t *testing.T) {
	s := NewStore(0, 3)
	l, _ := s.CreateList("リスト", "")

	for i := 0; i < 3; i++ {
		_, err := s.AddItem(l.ShareToken, "アイテム", "", true)
		if err != nil {
			t.Fatalf("item %d should be allowed, got error: %v", i+1, err)
		}
	}

	_, err := s.AddItem(l.ShareToken, "超過アイテム", "", true)
	if !errors.Is(err, errMaxItemsReached) {
		t.Fatalf("expected errMaxItemsReached, got %v", err)
	}

	items := s.GetList(l.ShareToken).Items
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
}

func TestMaxItemsLimitPerListIndependent(t *testing.T) {
	s := NewStore(0, 2)
	l1, _ := s.CreateList("リスト1", "")
	l2, _ := s.CreateList("リスト2", "")

	s.AddItem(l1.ShareToken, "A1", "", true)
	s.AddItem(l1.ShareToken, "A2", "", true)

	_, err := s.AddItem(l2.ShareToken, "B1", "", true)
	if err != nil {
		t.Fatalf("list2 should allow items independently, got error: %v", err)
	}
}

func TestZeroLimitsAreUnlimited(t *testing.T) {
	s := NewStore(0, 0)

	for i := 0; i < 10; i++ {
		l, err := s.CreateList("リスト", "")
		if err != nil {
			t.Fatalf("unlimited store should allow any number of lists, got error: %v", err)
		}
		for j := 0; j < 10; j++ {
			_, err := s.AddItem(l.ShareToken, "アイテム", "", true)
			if err != nil {
				t.Fatalf("unlimited store should allow any number of items, got error: %v", err)
			}
		}
	}
}

func TestGetEnv(t *testing.T) {
	os.Setenv("TEST_VAR", "hello")
	defer os.Unsetenv("TEST_VAR")
	if v := getEnv("TEST_VAR", "default"); v != "hello" {
		t.Fatalf("expected hello, got %s", v)
	}

	os.Unsetenv("TEST_VAR")
	if v := getEnv("TEST_VAR", "default"); v != "default" {
		t.Fatalf("expected default, got %s", v)
	}
}

func TestGetEnvInt(t *testing.T) {
	const key = "TEST_INT_VAR"

	os.Setenv(key, "42")
	if v := getEnvInt(key, 10); v != 42 {
		t.Fatalf("expected 42, got %d", v)
	}

	os.Setenv(key, "abc")
	if v := getEnvInt(key, 10); v != 10 {
		t.Fatalf("expected default 10 for invalid value, got %d", v)
	}

	os.Setenv(key, "")
	if v := getEnvInt(key, 99); v != 99 {
		t.Fatalf("expected default 99 for empty value, got %d", v)
	}

	os.Unsetenv(key)
	if v := getEnvInt(key, 50); v != 50 {
		t.Fatalf("expected default 50 for unset var, got %d", v)
	}

	os.Setenv(key, "0")
	if v := getEnvInt(key, 10); v != 0 {
		t.Fatalf("expected 0, got %d", v)
	}

	os.Setenv(key, "-5")
	if v := getEnvInt(key, 10); v != -5 {
		t.Fatalf("expected -5, got %d", v)
	}
	os.Unsetenv(key)
}
