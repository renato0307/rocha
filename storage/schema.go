package storage

import (
	"time"
)

// Session represents a tmux session (both top-level and nested)
type Session struct {
	BranchName   string    `gorm:"default:''"`
	ClaudeDir    string    `gorm:"default:''"`
	CreatedAt    time.Time
	DisplayName  string    `gorm:"not null;default:''"`
	ExecutionID  string    `gorm:"not null;index:idx_execution_id"`
	LastUpdated  time.Time `gorm:"not null;index:idx_last_updated"`
	Name         string    `gorm:"primaryKey"`
	ParentName   *string   `gorm:"index:idx_parent;default:null"` // NULL = top-level, set = nested session
	Position     int       `gorm:"not null;default:0;index:idx_position"` // Only used for top-level sessions
	RepoInfo     string    `gorm:"default:''"`
	RepoPath     string    `gorm:"default:''"`
	RepoSource   string    `gorm:"default:''"`
	State        string    `gorm:"not null;default:'idle';check:state IN ('waiting','working','idle','exited')"`
	UpdatedAt    time.Time
	WorktreePath string    `gorm:"default:''"`

	// Transient fields
	GitStats interface{} `gorm:"-" json:"-"`
}

// SessionFlag represents a flag/marker for a session (extension table)
type SessionFlag struct {
	CreatedAt   time.Time
	FlaggedAt   *time.Time `gorm:"default:null"`
	IsFlagged   bool       `gorm:"not null;default:false"`
	SessionName string     `gorm:"primaryKey"`
	UpdatedAt   time.Time
}

// SessionStatus represents the implementation status for a session (1-to-1 with Session)
type SessionStatus struct {
	CreatedAt   time.Time
	SessionName string `gorm:"primaryKey;index"` // Foreign key to Session.Name
	Status      string `gorm:"not null"`         // Implementation status (spec, plan, implement, review, done)
	UpdatedAt   time.Time
}

// SessionComment represents a comment/note for a session (1-to-1 with Session)
type SessionComment struct {
	Comment     string `gorm:"not null;default:''"`
	CreatedAt   time.Time
	SessionName string `gorm:"primaryKey"`
	UpdatedAt   time.Time
}

// SessionArchive represents archive status for a session (extension table)
type SessionArchive struct {
	ArchivedAt  *time.Time `gorm:"default:null"`
	CreatedAt   time.Time
	IsArchived  bool   `gorm:"not null;default:false"`
	SessionName string `gorm:"primaryKey"`
	UpdatedAt   time.Time
}

// SessionAgentCLIFlags represents CLI flags for the agent (Claude) for a session (extension table)
type SessionAgentCLIFlags struct {
	AllowDangerouslySkipPermissions bool   `gorm:"not null;default:false"`
	CreatedAt                       time.Time
	SessionName                     string `gorm:"primaryKey"`
	UpdatedAt                       time.Time
}
