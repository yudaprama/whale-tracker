package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CoingeckoSimplePrice represents the simple price API response
type CoingeckoSimplePrice map[string]map[string]float64

// CoingeckoService handles Coingecko API calls
type CoingeckoService struct {
	client  *http.Client
	baseURL string
}

// NewCoingeckoService creates a new Coingecko service
func NewCoingeckoService() *CoingeckoService {
	return &CoingeckoService{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.coingecko.com/api/v3",
	}
}

// GetSimplePrice gets the current price for tokens
// ids: comma-separated coingecko ids (e.g., "ethereum,bitcoin,tether")
// vs_currencies: target currency (default "usd")
func (c *CoingeckoService) GetSimplePrice(ids string, vsCurrencies string) (CoingeckoSimplePrice, error) {
	url := fmt.Sprintf("%s/simple/price?ids=%s&vs_currencies=%s", c.baseURL, ids, vsCurrencies)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch price: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var prices CoingeckoSimplePrice
	if err := json.NewDecoder(resp.Body).Decode(&prices); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return prices, nil
}

// GetPriceForToken gets the price for a single token by coingecko id
func (c *CoingeckoService) GetPriceForToken(coingeckoID string) (float64, error) {
	prices, err := c.GetSimplePrice(coingeckoID, "usd")
	if err != nil {
		return 0, err
	}

	if tokenData, ok := prices[coingeckoID]; ok {
		if usdPrice, ok := tokenData["usd"]; ok {
			return usdPrice, nil
		}
	}

	return 0, fmt.Errorf("price not found for %s", coingeckoID)
}

// FetchAllPrices fetches prices for all tokens in the database
func (s *Service) FetchAllPrices() (map[string]float64, error) {
	// Get all tokens with coingecko_id
	rows, err := s.db.Query(`
		SELECT address, symbol, coingecko_id
		FROM tokens
		WHERE coingecko_id IS NOT NULL
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	var addressMap = make(map[string]string) // coingecko_id -> address

	for rows.Next() {
		var address, symbol, coingeckoID string
		if err := rows.Scan(&address, &symbol, &coingeckoID); err != nil {
			return nil, err
		}
		ids = append(ids, coingeckoID)
		addressMap[coingeckoID] = address
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("no tokens with coingecko_id found")
	}

	// Build comma-separated list
	idsStr := ""
	for i, id := range ids {
		if i > 0 {
			idsStr += ","
		}
		idsStr += id
	}

	// Fetch prices from Coingecko
	coingecko := NewCoingeckoService()
	prices, err := coingecko.GetSimplePrice(idsStr, "usd")
	if err != nil {
		return nil, err
	}

	// Map token addresses to prices
	result := make(map[string]float64)
	for coingeckoID, tokenData := range prices {
		if usdPrice, ok := tokenData["usd"]; ok {
			if address, ok := addressMap[coingeckoID]; ok {
				result[address] = usdPrice
			}
		}
	}

	return result, nil
}

// UpdateAllPrices fetches and updates all token prices in the database
func (s *Service) UpdateAllPrices() error {
	prices, err := s.FetchAllPrices()
	if err != nil {
		return err
	}

	// Update each token price
	now := time.Now()
	for tokenAddress, priceUSD := range prices {
		price := TokenPrice{
			TokenAddress: tokenAddress,
			PriceUSD:     priceUSD,
			Source:       "coingecko",
			UpdatedAt:    now,
		}

		if err := s.UpdateTokenPrice(price); err != nil {
			// Log error but continue with other tokens
			fmt.Printf("Error updating price for %s: %v\n", tokenAddress, err)
		}
	}

	fmt.Printf("✅ Updated %d token prices from Coingecko\n", len(prices))
	return nil
}
