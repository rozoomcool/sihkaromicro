package model

type Project struct {
	Base
	OwnerID string `gorm:"column:owner_id;type:char(36);not null;index"`
	Title   string `gorm:"column:title;not null;size:255" json:"title"`
}

func (Project) TableName() string {
	return "projects"
}
