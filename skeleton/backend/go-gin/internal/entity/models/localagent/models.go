package localagent

import (
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/entity/models"
	"time"
)

// Local definitions are separate from the PowerX publication registry.
type Agent struct {
	UUID        string `gorm:"type:uuid;primaryKey"`
	TenantUUID  string `gorm:"type:uuid;not null;uniqueIndex:local_agent_key,priority:1"`
	Key         string `gorm:"size:128;not null;uniqueIndex:local_agent_key,priority:2"`
	Name        string
	Description string
	Status      string
	ModelKey    string
	Persona     string
	Prompt      string
	SkillUUIDs  []string `gorm:"serializer:json;type:jsonb"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Agent) TableName() string { return models.S("local_agent_definitions") }

type Skill struct {
	UUID        string `gorm:"type:uuid;primaryKey"`
	TenantUUID  string `gorm:"type:uuid;not null;uniqueIndex:local_skill_key,priority:1"`
	Key         string `gorm:"size:128;not null;uniqueIndex:local_skill_key,priority:2"`
	Version     string `gorm:"size:64;not null;uniqueIndex:local_skill_key,priority:3"`
	Name        string
	Description string
	Status      string
	Executor    string
	ModelKey    string
	Prompt      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Skill) TableName() string { return models.S("local_skill_definitions") }
