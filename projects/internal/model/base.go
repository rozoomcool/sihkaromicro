package model

import (
	"time"
)

// base contains common columns for all tables.
type Base struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;not null" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime;not null" json:"updatedAt"`
	// DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt"`
}

type baseID struct {
	ID int64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
}
