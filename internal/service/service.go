package service

import (
	"database/sql"
	"whale-tracker/internal/db"
)

type Service struct {
	db *sql.DB
}

// New creates a new service
func New(database *db.DB) *Service {
	return &Service{db: database.DB}
}
