package db

import (
	"database/sql"
	"fmt"

	_ "github.com/marcboeker/go-duckdb"
)

type DB struct {
	*sql.DB
}

// Open opens a DuckDB database connection
func Open(path string) (*DB, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Initialize schema
	if err := initSchema(db); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &DB{db}, nil
}

func initSchema(db *sql.DB) error {
	// Read and execute schema
	schema := `
	-- WHALES
	CREATE OR REPLACE TABLE IF NOT EXISTS whales (
		address VARCHAR(42) PRIMARY KEY,
		label VARCHAR(255) NOT NULL,
		telegram_chat_id VARCHAR(50),
		active BOOLEAN DEFAULT true,
		created_at TIMESTAMP DEFAULT NOW()
	);

	-- TOKENS
	CREATE OR REPLACE TABLE IF NOT EXISTS tokens (
		address VARCHAR(42) PRIMARY KEY,
		symbol VARCHAR(32) NOT NULL,
		decimals INT NOT NULL,
		category VARCHAR(50),
		coingecko_id VARCHAR(50)
	);

	-- TOKEN_PRICES
	CREATE OR REPLACE TABLE IF NOT EXISTS token_prices (
		token_address VARCHAR(42) PRIMARY KEY,
		price_usd DOUBLE NOT NULL,
		source VARCHAR(50),
		updated_at TIMESTAMP DEFAULT NOW()
	);

	-- BALANCES
	CREATE OR REPLACE TABLE IF NOT EXISTS balances (
		whale_address VARCHAR(42) NOT NULL,
		token_address VARCHAR(42) NOT NULL,
		balance_decimal DOUBLE NOT NULL,
		prev_balance_decimal DOUBLE,
		change_percent DOUBLE,
		last_updated TIMESTAMP NOT NULL,
		PRIMARY KEY (whale_address, token_address)
	);

	-- VIEW: latest_holdings
	CREATE OR REPLACE VIEW latest_holdings AS
	SELECT
		w.label as whale,
		t.symbol as token,
		t.category,
		b.balance_decimal,
		tp.price_usd as token_price,
		(b.balance_decimal * tp.price_usd) as usd_value,
		b.change_percent,
		b.last_updated
	FROM balances b
	JOIN whales w ON b.whale_address = w.address
	JOIN tokens t ON b.token_address = t.address
	JOIN token_prices tp ON b.token_address = tp.token_address
	WHERE b.balance_decimal > 0
	ORDER BY usd_value DESC;

	-- VIEW: big_changes
	CREATE OR REPLACE VIEW big_changes AS
	SELECT
		w.label as whale,
		w.telegram_chat_id,
		t.symbol as token,
		b.balance_decimal,
		b.prev_balance_decimal,
		b.change_percent,
		(b.balance_decimal * tp.price_usd) as usd_value,
		b.last_updated
	FROM balances b
	JOIN whales w ON b.whale_address = w.address
	JOIN tokens t ON b.token_address = t.address
	JOIN token_prices tp ON b.token_address = tp.token_address
	WHERE b.change_percent IS NOT NULL
		AND ABS(b.change_percent) >= 10
	ORDER BY ABS(b.change_percent) DESC;
	`

	_, err := db.Exec(schema)
	return err
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}
