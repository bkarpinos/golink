package link

import (
	"time"
)

// Link represents a go link with alias and target URL
type Link struct {
	Alias       string    `json:"alias"`
	URL         string    `json:"url"`
	Description string    `json:"description,omitempty"`
	Category    string    `json:"category,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewLink creates a new link with current timestamp
func NewLink(alias, url, description, category string) *Link {
	now := time.Now()
	return &Link{
		Alias:       alias,
		URL:         url,
		Description: description,
		Category:    category,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
