package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Store is a thread-safe in-memory store for lists.
type Store struct {
	mu              sync.RWMutex
	lists           map[string]*List // keyed by ShareToken
	nextID          int
	maxLists        int
	maxItemsPerList int
}

func NewStore(maxLists, maxItemsPerList int) *Store {
	return &Store{
		lists:           make(map[string]*List),
		maxLists:        maxLists,
		maxItemsPerList: maxItemsPerList,
	}
}

func (s *Store) genID() string {
	s.nextID++
	return fmt.Sprintf("%d", s.nextID)
}

func generateToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("乱数生成に失敗しました: " + err.Error())
	}
	return hex.EncodeToString(b)
}

var errMaxListsReached = fmt.Errorf("リスト数が上限に達しています")
var errMaxItemsReached = fmt.Errorf("アイテム数が上限に達しています")

func (s *Store) CreateList(title, description string) (*List, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.maxLists > 0 && len(s.lists) >= s.maxLists {
		return nil, errMaxListsReached
	}

	l := &List{
		ID:          s.genID(),
		Title:       title,
		Description: description,
		ShareToken:  generateToken(),
		Items:       []*Item{},
		CreatedAt:   time.Now(),
	}
	s.lists[l.ShareToken] = l
	return l, nil
}

func (s *Store) GetList(token string) *List {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lists[token]
}

func (s *Store) AddItem(token, name, assignee string, required bool) (*Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	l := s.lists[token]
	if l == nil {
		return nil, nil
	}
	if s.maxItemsPerList > 0 && len(l.Items) >= s.maxItemsPerList {
		return nil, errMaxItemsReached
	}
	item := &Item{
		ID:        s.genID(),
		Name:      name,
		Assignee:  assignee,
		Required:  required,
		UpdatedAt: time.Now(),
	}
	l.Items = append(l.Items, item)
	return item, nil
}

func (s *Store) findItem(token, itemID string) *Item {
	l := s.lists[token]
	if l == nil {
		return nil
	}
	for _, it := range l.Items {
		if it.ID == itemID {
			return it
		}
	}
	return nil
}

func (s *Store) TogglePrepared(token, itemID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if it := s.findItem(token, itemID); it != nil {
		it.Prepared = !it.Prepared
		it.UpdatedAt = time.Now()
	}
}

func (s *Store) ToggleRequired(token, itemID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if it := s.findItem(token, itemID); it != nil {
		it.Required = !it.Required
		it.UpdatedAt = time.Now()
	}
}

func (s *Store) UpdateAssignee(token, itemID, assignee string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if it := s.findItem(token, itemID); it != nil {
		it.Assignee = assignee
		it.UpdatedAt = time.Now()
	}
}

// DeleteList はトークンに対応するリストを削除する。削除に成功した場合は true を返す。
func (s *Store) DeleteList(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.lists[token]; !ok {
		return false
	}
	delete(s.lists, token)
	return true
}

func (s *Store) Stats() (listCount int, itemCount int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	listCount = len(s.lists)
	for _, l := range s.lists {
		itemCount += len(l.Items)
	}
	return
}

func (s *Store) DeleteItem(token, itemID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	l := s.lists[token]
	if l == nil {
		return
	}
	for i, it := range l.Items {
		if it.ID == itemID {
			l.Items = append(l.Items[:i], l.Items[i+1:]...)
			return
		}
	}
}
