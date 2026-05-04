package service

import (
	"database/sql"
	"fmt"
)

type Token struct {
	Address     string
	Symbol      string
	Decimals    int
	Category    sql.NullString
	CoingeckoID sql.NullString
}

// InsertToken inserts a new token
func (s *Service) InsertToken(token Token) error {
	_, err := s.db.Exec(`
		INSERT INTO tokens (address, symbol, decimals, category, coingecko_id)
		VALUES (?, ?, ?, ?, ?)
	`, token.Address, token.Symbol, token.Decimals, token.Category, token.CoingeckoID)
	return err
}

// GetTokenBySymbol gets a token by symbol
func (s *Service) GetTokenBySymbol(symbol string) (*Token, error) {
	var token Token
	err := s.db.QueryRow(`
		SELECT address, symbol, decimals, category, coingecko_id
		FROM tokens WHERE symbol = ?
	`, symbol).Scan(&token.Address, &token.Symbol, &token.Decimals, &token.Category, &token.CoingeckoID)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("token not found")
	}
	return &token, err
}

// ListTokens returns all tokens
func (s *Service) ListTokens() ([]Token, error) {
	rows, err := s.db.Query(`SELECT address, symbol, decimals, category, coingecko_id FROM tokens`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []Token
	for rows.Next() {
		var t Token
		if err := rows.Scan(&t.Address, &t.Symbol, &t.Decimals, &t.Category, &t.CoingeckoID); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, nil
}
