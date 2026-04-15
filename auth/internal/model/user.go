package model

type User struct {
	Base
	Username string `gorm:"uniqueIndex;not null;size:255" json:"username"`
	Password string `gorm:"not null;size:255" json:"-"`

	RefreshTokens []RefreshToken `gorm:"foreignKey:UserID" json:"-"`
}

func (User) TableName() string {
	return "users"
}
