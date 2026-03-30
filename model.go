package main

import "time"

// List represents a packing list for a trip/event.
type List struct {
	ID          string
	Title       string
	Description string
	ShareToken  string
	Items       []*Item
	CreatedAt   time.Time
}

// Item represents a single item in a packing list.
type Item struct {
	ID       string
	Name     string
	Assignee string
	Required bool
	Prepared bool
}
