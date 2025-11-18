package store

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"obsync/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type SQLiteStore struct {
	DB *gorm.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&models.Device{}, &models.Change{}); err != nil {
		return nil, err
	}

	return &SQLiteStore{DB: db}, nil
}

func genToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *SQLiteStore) CreateDevice(deviceID, displayName string) (*models.Device, error) {
	token, err := genToken(16)
	if err != nil {
		return nil, err
	}
	d := &models.Device{
		DeviceID:    deviceID,
		DisplayName: displayName,
		Token:       token,
		CreatedAt:   time.Now(),
	}
	if err := s.DB.Create(d).Error; err != nil {
		return nil, err
	}
	return d, nil
}

func (s *SQLiteStore) GetDeviceByToken(token string) (*models.Device, error) {
	var d models.Device
	if err := s.DB.Where("token = ?", token).First(&d).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *SQLiteStore) SaveChanges(changes []models.Change) error {
	if len(changes) == 0 {
		return nil
	}
	tx := s.DB.Begin()
	for _, c := range changes {
		if err := tx.Create(&c).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

func (s *SQLiteStore) ListChangesSince(repo string, sinceID uint) ([]models.Change, error) {
	var out []models.Change
	if repo == "" {
		return nil, errors.New("repo required")
	}
	if err := s.DB.Where("repo = ? AND id > ?", repo, sinceID).Order("id asc").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}
