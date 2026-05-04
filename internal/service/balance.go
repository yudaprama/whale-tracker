package service

import (
	"database/sql"
	"time"
)

type Balance struct {
	WhaleAddress       string
	TokenAddress       string
	BalanceDecimal     float64
	PrevBalanceDecimal sql.NullFloat64
	ChangePercent      sql.NullFloat64
	LastUpdated        time.Time
}

type Holding struct {
	Whale        string
	Token        string
	Category     string
	Balance      float64
	TokenPrice   float64
	USDValue     float64
	ChangePercent sql.NullFloat64
	LastUpdated  time.Time
}

// UpdateBalance updates or inserts a balance
func (s *Service) UpdateBalance(balance Balance) error {
	// First, get the old balance if exists
	var oldBalance sql.NullFloat64
	err := s.db.QueryRow(`
		SELECT balance_decimal FROM balances
		WHERE whale_address = ? AND token_address = ?
	`, balance.WhaleAddress, balance.TokenAddress).Scan(&oldBalance)

	if err != nil && err != sql.ErrNoRows {
		return err
	}

	// Calculate change percent if old balance exists
	var changePercent sql.NullFloat64
	if oldBalance.Valid && oldBalance.Float64 > 0 {
		change := ((balance.BalanceDecimal - oldBalance.Float64) / oldBalance.Float64) * 100
		changePercent.Float64 = change
		changePercent.Valid = true
	}

	// UPSERT
	_, err = s.db.Exec(`
		INSERT INTO balances (whale_address, token_address, balance_decimal, prev_balance_decimal, change_percent, last_updated)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (whale_address, token_address) DO UPDATE SET
			balance_decimal = excluded.balance_decimal,
			prev_balance_decimal = excluded.prev_balance_decimal,
			change_percent = excluded.change_percent,
			last_updated = excluded.last_updated
	`, balance.WhaleAddress, balance.TokenAddress, balance.BalanceDecimal, oldBalance, changePercent, time.Now())

	return err
}

// GetLatestHoldings returns all current holdings
func (s *Service) GetLatestHoldings() ([]Holding, error) {
	rows, err := s.db.Query(`
		SELECT whale, token, balance_decimal, token_price, usd_value, change_percent, last_updated
		FROM latest_holdings
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var holdings []Holding
	for rows.Next() {
		var h Holding
		if err := rows.Scan(&h.Whale, &h.Token, &h.Balance, &h.TokenPrice, &h.USDValue, &h.ChangePercent, &h.LastUpdated); err != nil {
			return nil, err
		}
		holdings = append(holdings, h)
	}
	return holdings, nil
}

// GetHoldingsByWhale returns holdings for a specific whale
func (s *Service) GetHoldingsByWhale(whaleLabel string) ([]Holding, error) {
	rows, err := s.db.Query(`
		SELECT whale, token, balance_decimal, token_price, usd_value, change_percent, last_updated
		FROM latest_holdings WHERE whale = ?
	`, whaleLabel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var holdings []Holding
	for rows.Next() {
		var h Holding
		if err := rows.Scan(&h.Whale, &h.Token, &h.Balance, &h.TokenPrice, &h.USDValue, &h.ChangePercent, &h.LastUpdated); err != nil {
			return nil, err
		}
		holdings = append(holdings, h)
	}
	return holdings, nil
}

// GetBigChanges returns whales with >10% change
func (s *Service) GetBigChanges() ([]Holding, error) {
	rows, err := s.db.Query(`
		SELECT whale, token, balance_decimal, token_price, usd_value, change_percent, last_updated
		FROM big_changes
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var holdings []Holding
	for rows.Next() {
		var h Holding
		if err := rows.Scan(&h.Whale, &h.Token, &h.Balance, &h.TokenPrice, &h.USDValue, &h.ChangePercent, &h.LastUpdated); err != nil {
			return nil, err
		}
		holdings = append(holdings, h)
	}
	return holdings, nil
}
