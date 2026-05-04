package main

import (
	"database/sql"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
	"whale-tracker/internal/db"
	"whale-tracker/internal/service"
)

type Config struct {
	RPCURLs        []string      `yaml:"rpc_urls"`
	UpdateInterval int           `yaml:"update_interval"`
	Whales         []WhaleConfig `yaml:"whales"`
	Tokens         []TokenConfig `yaml:"tokens"`
}

type WhaleConfig struct {
	Address        string `yaml:"address"`
	Label          string `yaml:"label"`
	TelegramChatID string `yaml:"telegram_chat_id"`
	Active         bool   `yaml:"active"`
}

type TokenConfig struct {
	Address     string `yaml:"address"`
	Symbol      string `yaml:"symbol"`
	Decimals    int    `yaml:"decimals"`
	Category    string `yaml:"category"`
	CoingeckoID string `yaml:"coingecko_id"`
}

type RPCRotator struct {
	urls   []string
	index  int
	mu     sync.Mutex
}

func NewRPCRotator(urls []string) *RPCRotator {
	if len(urls) == 0 {
		urls = []string{"https://ethereum.publicnode.com"}
	}
	return &RPCRotator{urls: urls}
}

func (r *RPCRotator) Next() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	url := r.urls[r.index]
	r.index = (r.index + 1) % len(r.urls)
	return url
}

func (r *RPCRotator) Current() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.urls[r.index]
}

func main() {
	// Load config
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Set defaults
	if cfg.UpdateInterval == 0 {
		cfg.UpdateInterval = 30
	}

	// Initialize RPC rotator
	rpcRotator := NewRPCRotator(cfg.RPCURLs)

	// Open database
	database, err := db.Open("data/whale_tracker.duckdb")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	svc := service.New(database)

	log.Println("Starting Whale Tracker...")
	log.Printf("Loaded %d RPC endpoints", len(cfg.RPCURLs))
	log.Printf("Loaded %d whales, %d tokens", len(cfg.Whales), len(cfg.Tokens))

	// Initial data load from config
	if err := loadWhalesFromConfig(svc, cfg.Whales); err != nil {
		log.Fatalf("Error loading whales: %v", err)
	}
	if err := loadTokensFromConfig(svc, cfg.Tokens); err != nil {
		log.Fatalf("Error loading tokens: %v", err)
	}

	// Setup graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start update loop
	interval := time.Duration(cfg.UpdateInterval) * time.Second
	updateTicker := time.NewTicker(interval)
	defer updateTicker.Stop()

	// Initial update
	runUpdate(svc, rpcRotator)

	// Main loop
	for {
		select {
		case <-updateTicker.C:
			runUpdate(svc, rpcRotator)
		case <-stop:
			log.Println("Shutting down...")
			return
		}
	}
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func loadWhalesFromConfig(svc *service.Service, whales []WhaleConfig) error {
	for _, w := range whales {
		whale := service.Whale{
			Address:        w.Address,
			Label:          w.Label,
			TelegramChatID: sql.NullString{String: w.TelegramChatID, Valid: w.TelegramChatID != ""},
			Active:         w.Active,
		}
		if err := svc.InsertWhale(whale); err != nil {
			log.Printf("Warning: %v", err)
		}
	}
	return nil
}

func loadTokensFromConfig(svc *service.Service, tokens []TokenConfig) error {
	for _, t := range tokens {
		token := service.Token{
			Address:     t.Address,
			Symbol:      t.Symbol,
			Decimals:    t.Decimals,
			Category:    sql.NullString{String: t.Category, Valid: t.Category != ""},
			CoingeckoID: sql.NullString{String: t.CoingeckoID, Valid: t.CoingeckoID != ""},
		}
		if err := svc.InsertToken(token); err != nil {
			log.Printf("Warning: %v", err)
		}
	}
	return nil
}

func runUpdate(svc *service.Service, rotator *RPCRotator) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	rpcURL := rotator.Next()

	log.Printf("[%s] Starting update... (RPC: %s)", timestamp, shortURL(rpcURL))

	// Update prices
	if err := svc.UpdateAllPrices(); err != nil {
		log.Printf("Error updating prices: %v", err)
	}

	// Update balances from RPC with retry logic
	err := svc.FetchWhaleBalances(rpcURL)
	if err != nil {
		log.Printf("Error with RPC %s: %v", shortURL(rpcURL), err)

		// Try next RPC
		nextRPC := rotator.Next()
		log.Printf("Retrying with next RPC: %s", shortURL(nextRPC))
		if err := svc.FetchWhaleBalances(nextRPC); err != nil {
			log.Printf("Error updating balances: %v", err)
			return
		}
	}

	// Get results and log
	holdings, err := svc.GetLatestHoldings()
	if err != nil {
		log.Printf("Error getting holdings: %v", err)
		return
	}

	for _, h := range holdings {
		log.Printf("  %s: %s = %.2f ($%.2f)", h.Whale, h.Token, h.Balance, h.USDValue)
	}

	// Check for alerts
	changes, err := svc.GetBigChanges()
	if err != nil {
		log.Printf("Error getting changes: %v", err)
		return
	}

	if len(changes) > 0 {
		log.Printf("  ALERTS: %d big changes detected", len(changes))
		for _, c := range changes {
			change := 0.0
			if c.ChangePercent.Valid {
				change = c.ChangePercent.Float64
			}
			log.Printf("    %s %s: %.2f%% ($%.2f)", c.Whale, c.Token, change, c.USDValue)
		}
	}
}

func shortURL(url string) string {
	if len(url) > 30 {
		return url[:27] + "..."
	}
	return url
}
