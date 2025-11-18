package models

import "time"

type Device struct {
	ID          uint      `gorm:"primaryKey" json:"-"`
	DeviceID    string    `gorm:"uniqueIndex;size:128" json:"device_id"`
	DisplayName string    `gorm:"size:255" json:"display_name"`
	Token       string    `gorm:"size:255;uniqueIndex" json:"device_token"`
	CreatedAt   time.Time `json:"created_at"`
}

type Change struct {
	ID        uint      `gorm:"primaryKey" json:"-"`
	ChangeID  string    `gorm:"uniqueIndex;size:128" json:"change_id"`
	Repo      string    `gorm:"size:255;index" json:"repo"`
	FileUID   string    `gorm:"size:255;index" json:"file_uid"`
	Op        string    `gorm:"size:32" json:"op"`
	Path      string    `gorm:"size:1024" json:"path"`
	BaseSHA   string    `gorm:"size:128" json:"base_sha"`
	NewSHA    string    `gorm:"size:128" json:"new_sha"`
	DeviceID  string    `gorm:"size:128;index" json:"device_id"`
	Timestamp time.Time `json:"timestamp"`
	// Attachments stored as JSON string for MVP
	Attachments string `gorm:"type:text" json:"attachments"`
}

// TimeNow is a helper to return current time; kept for handler compatibility
func TimeNow() time.Time { return time.Now() }
