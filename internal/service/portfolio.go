package service

type PortfolioTotal struct {
	Whale    string
	TotalUSD float64
}

// GetTotalPortfolio returns total portfolio value per whale
func (s *Service) GetTotalPortfolio() ([]PortfolioTotal, error) {
	rows, err := s.db.Query(`
		SELECT whale, SUM(usd_value) as total_usd
		FROM latest_holdings
		GROUP BY whale
		ORDER BY total_usd DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var portfolios []PortfolioTotal
	for rows.Next() {
		var p PortfolioTotal
		if err := rows.Scan(&p.Whale, &p.TotalUSD); err != nil {
			return nil, err
		}
		portfolios = append(portfolios, p)
	}
	return portfolios, nil
}
