package models

import (
	"time"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    Email     string    `gorm:"uniqueIndex;not null" json:"email"`
    Password  string    `gorm:"" json:"-"`
    GoogleID  string    `gorm:"uniqueIndex" json:"-"`
    AppleID   string    `gorm:"uniqueIndex" json:"-"`
    FacebookID string   `gorm:"uniqueIndex" json:"-"`
    TikTokID  string    `gorm:"uniqueIndex" json:"-"`
    CreatedAt time.Time `json:"created_at"`
}
func (u *User) HashPassword(password string) error {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return err
	}
	u.Password = string(bytes)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}
