package model

import "time"

type RefreshToken struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	UserID    int64     `gorm:"not null;index"`
	Token     string    `gorm:"uniqueIndex;not null;size:500"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}
