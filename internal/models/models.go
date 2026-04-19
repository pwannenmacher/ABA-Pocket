package models

import "time"

type User struct {
	ID           int64
	Username     string
	Email        string
	PasswordHash string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Session struct {
	ID        string
	UserID    int64
	CreatedAt time.Time
	ExpiresAt time.Time
}

type CardEntry struct {
	ID        int64
	LeftCol   string
	RightCol  string
	SortOrder int
}

type Symptom struct {
	ID          int64
	Title       string
	Description string
	Source      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Entries     []CardEntry
	Medications []*Medication
}

type Medication struct {
	ID          int64
	Name        string
	Description string
	Source      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Entries     []CardEntry
	Symptoms    []*Symptom
}

// SearchResult holds a unified search result
type SearchResult struct {
	Type  string // "symptom" or "medication"
	ID    int64
	Title string
}
