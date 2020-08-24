package model

import (
	"time"
)

// StarterKit represents an installed starter kit.
type StarterKit struct {
	ID          int    `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Website     string `json:"website,omitempty"`

	NumStars  int `json:"num_stars,omitempty"`
	NumForks  int `json:"num_forks,omitempty"`
	NumIssues int `json:"num_issues,omitempty"`
	NumPulls  int `json:"num_pulls,omitempty"`

	IsPrivate bool `json:"is_private,omitempty"`
	IsMirror  bool `json:"is_mirror,omitempty"`
	IsFork    bool `json:"is_fork,omitempty"`

	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
	DeletedOn time.Time `json:"deleted_on"`
}

// NewStarterKit istantiates a new starter kit
func NewStarterKit(id int, repoName, author string) StarterKit {
	return StarterKit{
		ID:          id,
		Name:        repoName,
		Description: "Description of this repository",
		Website:     "www.ribice.ba/glice",
		NumStars:    20,
		NumForks:    3,
		NumIssues:   15,
		NumPulls:    2,
	}
}

// NewStarterKits returns slice of starter kits
func NewStarterKits(author string) []StarterKit {
	return []StarterKit{NewStarterKit(1, "glice", author), NewStarterKit(2, "kiss", author)}
}
