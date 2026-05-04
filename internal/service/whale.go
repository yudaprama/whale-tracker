package service

import (
	"database/sql"
	"fmt"
)

type Whale struct {
	Address        string
	Label          string
	TelegramChatID sql.NullString
	Active         bool
}

// InsertWhale inserts a new whale
func (s *Service) InsertWhale(whale Whale) error {
	_, err := s.db.Exec(`
		INSERT INTO whales (address, label, telegram_chat_id)
		VALUES (?, ?, ?)
	`, whale.Address, whale.Label, whale.TelegramChatID)
	return err
}

// GetWhaleByAddress gets a whale by address
func (s *Service) GetWhaleByAddress(address string) (*Whale, error) {
	var whale Whale
	err := s.db.QueryRow(`
		SELECT address, label, telegram_chat_id, active
		FROM whales WHERE address = ?
	`, address).Scan(&whale.Address, &whale.Label, &whale.TelegramChatID, &whale.Active)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("whale not found")
	}
	return &whale, err
}

// ListWhales returns all whales
func (s *Service) ListWhales() ([]Whale, error) {
	rows, err := s.db.Query(`SELECT address, label, telegram_chat_id, active FROM whales`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var whales []Whale
	for rows.Next() {
		var w Whale
		if err := rows.Scan(&w.Address, &w.Label, &w.TelegramChatID, &w.Active); err != nil {
			return nil, err
		}
		whales = append(whales, w)
	}
	return whales, nil
}
