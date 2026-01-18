package storage

import (
	"time"
)

// Session represents a tmux session (both top-level and nested)
type Session struct {
	Name         string    `gorm:"primaryKey"`
	ParentName   *string   `gorm:"index:idx_parent;default:null"` // NULL = top-level, set = nested session
	DisplayName  string    `gorm:"not null;default:''"`
	State        string    `gorm:"not null;default:'idle';check:state IN ('waiting','working','idle','exited')"`
	ExecutionID  string    `gorm:"not null;index:idx_execution_id"`
	LastUpdated  time.Time `gorm:"not null;index:idx_last_updated"`
	RepoPath     string    `gorm:"default:''"`
	RepoInfo     string    `gorm:"default:''"`
	BranchName   string    `gorm:"default:''"`
	WorktreePath string    `gorm:"default:''"`
	Position     int       `gorm:"not null;default:0;index:idx_position"` // Only used for top-level sessions
	CreatedAt    time.Time
	UpdatedAt    time.Time

	// Transient fields
	GitStats interface{} `gorm:"-" json:"-"`
}
