package storage

import "time"

// SessionModel is the GORM model for sessions table
type SessionModel struct {
	BranchName   string    `gorm:"default:''"`
	ClaudeDir    string    `gorm:"default:''"`
	CreatedAt    time.Time
	DisplayName  string    `gorm:"not null;default:''"`
	ExecutionID  string    `gorm:"not null;index:idx_execution_id"`
	GitStats     any       `gorm:"-" json:"-"`
	LastUpdated  time.Time `gorm:"not null;index:idx_last_updated"`
	Name         string    `gorm:"primaryKey"`
	ParentName   *string   `gorm:"index:idx_parent;default:null"`
	Position     int       `gorm:"not null;default:0;index:idx_position"`
	RepoInfo     string    `gorm:"default:''"`
	RepoPath     string    `gorm:"default:''"`
	RepoSource   string    `gorm:"default:''"`
	State        string    `gorm:"not null;default:'idle';check:state IN ('waiting','working','idle','exited')"`
	UpdatedAt    time.Time
	WorktreePath string    `gorm:"default:''"`
}

// TableName specifies the table name for GORM
func (SessionModel) TableName() string { return "sessions" }

// SessionFlagModel is the GORM model for session flags
type SessionFlagModel struct {
	CreatedAt   time.Time
	FlaggedAt   *time.Time `gorm:"default:null"`
	IsFlagged   bool       `gorm:"not null;default:false"`
	SessionName string     `gorm:"primaryKey"`
	UpdatedAt   time.Time
}

// TableName specifies the table name for GORM
func (SessionFlagModel) TableName() string { return "session_flags" }

// SessionStatusModel is the GORM model for session status
type SessionStatusModel struct {
	CreatedAt   time.Time
	SessionName string `gorm:"primaryKey;index"`
	Status      string `gorm:"not null"`
	UpdatedAt   time.Time
}

// TableName specifies the table name for GORM
func (SessionStatusModel) TableName() string { return "session_statuses" }

// SessionCommentModel is the GORM model for session comments
type SessionCommentModel struct {
	Comment     string `gorm:"not null;default:''"`
	CreatedAt   time.Time
	SessionName string `gorm:"primaryKey"`
	UpdatedAt   time.Time
}

// TableName specifies the table name for GORM
func (SessionCommentModel) TableName() string { return "session_comments" }

// SessionArchiveModel is the GORM model for session archive status
type SessionArchiveModel struct {
	ArchivedAt  *time.Time `gorm:"default:null"`
	CreatedAt   time.Time
	IsArchived  bool   `gorm:"not null;default:false"`
	SessionName string `gorm:"primaryKey"`
	UpdatedAt   time.Time
}

// TableName specifies the table name for GORM
func (SessionArchiveModel) TableName() string { return "session_archives" }

// SessionAgentCLIFlagsModel is the GORM model for agent CLI flags
type SessionAgentCLIFlagsModel struct {
	AllowDangerouslySkipPermissions bool   `gorm:"not null;default:false"`
	CreatedAt                       time.Time
	SessionName                     string `gorm:"primaryKey"`
	UpdatedAt                       time.Time
}

// TableName specifies the table name for GORM
func (SessionAgentCLIFlagsModel) TableName() string { return "session_agent_cli_flags" }
