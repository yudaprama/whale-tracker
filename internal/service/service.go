package service

import (
	"whale-tracker/internal/db"
)

type Service struct {
	db *db.DB
}

// New creates a new service
func New(database *db.DB) *Service {
	return &Service{db: database.DB}
}
