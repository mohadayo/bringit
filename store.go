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
	mu     sync.RWMutex
	lists  map[string]*List // keyed by ShareToken
	nextID int
}

func NewStore() *Store {
	return &Store{lists: make(map[string]*List)}
}

func (s *Store) genID() string {
	s.nextID++
	return fmt.Sprintf("%d", s.nextID)
}

func generateToken() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Store) CreateList(title, description string) *List {
	s.mu.Lock()
	defer s.mu.Unlock()

	l := &List{
		ID:          s.genID(),
		Title:       title,
		Description: description,
		ShareToken:  generateToken(),
		Items:       []*Item{},
		CreatedAt:   time.Now(),
	}
	s.lists[l.ShareToken] = l
	return l
}

func (s *Store) GetList(token string) *List {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lists[token]
}

func (s *Store) AddItem(token, name, assignee string, required bool) *Item {
	s.mu.Lock()
	defer s.mu.Unlock()

	l := s.lists[token]
	if l == nil {
		return nil
	}
	item := &Item{
		ID:       s.genID(),
		Name:     name,
		Assignee: assignee,
		Required: required,
	}
	l.Items = append(l.Items, item)
	return item
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
	}
}

func (s *Store) ToggleRequired(token, itemID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if it := s.findItem(token, itemID); it != nil {
		it.Required = !it.Required
	}
}

func (s *Store) UpdateAssignee(token, itemID, assignee string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if it := s.findItem(token, itemID); it != nil {
		it.Assignee = assignee
	}
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
