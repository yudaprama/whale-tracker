package main

import (
	"fmt"
	"log"
	"whale-tracker/internal/db"
	"whale-tracker/internal/service"
)

func main() {
	// Open database
	database, err := db.Open("whale_tracker.duckdb")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	// Create service
	svc := service.New(database)

	// Example: Insert sample data
	fmt.Println("=== Whale Tracker ===")

	// Insert whales
	whales := []service.Whale{
		{Address: "0x3f5CE5FBFe3E9af3971dD833D26bA9b5C936f0bE", Label: "Binance Cold Wallet"},
		{Address: "0x2B8f1b796eE9Ad21Ce014197E58dAeeB8666D091", Label: "Bitfinex Cold Wallet"},
	}
	for _, w := range whales {
		if err := svc.InsertWhale(w); err != nil {
			log.Printf("Error inserting whale: %v", err)
		}
	}

	// Insert tokens
	tokens := []service.Token{
		{Address: "0x0000000000000000000000000000000000000000", Symbol: "ETH", Decimals: 18},
		{Address: "0xdac17f958d2ee523a2206206994597c13d831ec7", Symbol: "USDT", Decimals: 6},
	}
	for _, t := range tokens {
		if err := svc.InsertToken(t); err != nil {
			log.Printf("Error inserting token: %v", err)
		}
	}

	// Update prices
	prices := []service.TokenPrice{
		{TokenAddress: "0x0000000000000000000000000000000000000000", PriceUSD: 3000.0, Source: "manual"},
		{TokenAddress: "0xdac17f958d2ee523a2206206994597c13d831ec7", PriceUSD: 1.0, Source: "manual"},
	}
	for _, p := range prices {
		if err := svc.UpdateTokenPrice(p); err != nil {
			log.Printf("Error updating price: %v", err)
		}
	}

	// Update balances
	balances := []service.Balance{
		{WhaleAddress: "0x3f5CE5FBFe3E9af3971dD833D26bA9b5C936f0bE", TokenAddress: "0x0000000000000000000000000000000000000000", BalanceDecimal: 50000.0},
		{WhaleAddress: "0x3f5CE5FBFe3E9af3971dD833D26bA9b5C936f0bE", TokenAddress: "0xdac17f958d2ee523a2206206994597c13d831ec7", BalanceDecimal: 10000.0},
	}
	for _, b := range balances {
		if err := svc.UpdateBalance(b); err != nil {
			log.Printf("Error updating balance: %v", err)
		}
	}

	// Query latest holdings
	fmt.Println("\n=== Latest Holdings ===")
	holdings, err := svc.GetLatestHoldings()
	if err != nil {
		log.Fatal(err)
	}
	for _, h := range holdings {
		fmt.Printf("%s: %s = %.2f ($%.2f)\n", h.Whale, h.Token, h.Balance, h.USDValue)
	}

	// Query Binance holdings
	fmt.Println("\n=== Binance Holdings ===")
	binanceHoldings, err := svc.GetHoldingsByWhale("Binance Cold Wallet")
	if err != nil {
		log.Fatal(err)
	}
	for _, h := range binanceHoldings {
		fmt.Printf("%s: %.2f ($%.2f)\n", h.Token, h.Balance, h.USDValue)
	}
}
