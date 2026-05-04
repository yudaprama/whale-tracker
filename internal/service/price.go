package service

import (
	"database/sql"
	"time"
)

type TokenPrice struct {
	TokenAddress string
	PriceUSD     float64
	Source       string
	UpdatedAt    time.Time
}

// UpdateTokenPrice updates or inserts a token price
func (s *Service) UpdateTokenPrice(price TokenPrice) error {
	_, err := s.db.Exec(`
		INSERT INTO token_prices (token_address, price_usd, source, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (token_address) DO UPDATE SET
			price_usd = excluded.price_usd,
			source = excluded.source,
			updated_at = excluded.updated_at
	`, price.TokenAddress, price.PriceUSD, price.Source, price.UpdatedAt)
	return err
}

// GetLatestPrice gets the latest price for a token
func (s *Service) GetLatestPrice(tokenAddress string) (*TokenPrice, error) {
	var price TokenPrice
	err := s.db.QueryRow(`
		SELECT token_address, price_usd, source, updated_at
		FROM token_prices WHERE token_address = ?
	`, tokenAddress).Scan(&price.TokenAddress, &price.PriceUSD, &price.Source, &price.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, err
	}
	return &price, err
}

// ListLatestPrices returns all latest prices
func (s *Service) ListLatestPrices() ([]TokenPrice, error) {
	rows, err := s.db.Query(`
		SELECT tp.token_address, tp.price_usd, tp.source, tp.updated_at, t.symbol
		FROM token_prices tp
		JOIN tokens t ON tp.token_address = t.address
		ORDER BY tp.price_usd DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []TokenPrice
	for rows.Next() {
		var p TokenPrice
		var symbol string
		if err := rows.Scan(&p.TokenAddress, &p.PriceUSD, &p.Source, &p.UpdatedAt, &symbol); err != nil {
			return nil, err
		}
		prices = append(prices, p)
	}
	return prices, nil
}
