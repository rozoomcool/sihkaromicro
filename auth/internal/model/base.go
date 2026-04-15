package model

import (
	"time"
)

// Base contains common columns for all tables.
type Base struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time `gorm:"autoCreateTime;not null" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;not null" json:"updatedAt"`
	// DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt"`
}

type BaseID struct {
	ID int64 `gorm:"primaryKey;autoIncrement" json:"id"`
}
