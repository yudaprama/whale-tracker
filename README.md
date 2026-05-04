# 🐋 Whale Tracker

Simple whale balance tracking with DuckDB and Go.

## 📁 Project Structure

```
whale-tracker/
├── schema/
│   └── 001_create_tables.sql    # Database schema
├── sql/
│   ├── 002_sample_data.sql      # Sample data
│   └── queries.sql              # Useful queries
├── cmd/server/
│   └── main.go                  # Go application
├── internal/
│   ├── db/
│   │   └── db.go                # Database connection & schema
│   └── service/
│       ├── service.go           # Service struct
│       ├── whale.go             # Whale operations
│       ├── token.go             # Token operations
│       ├── price.go             # Price operations
│       └── balance.go           # Balance operations
├── data/                        # Database & exports
├── go.mod
└── README.md
```

## 🗂️ Database Schema

| Table | Purpose |
|-------|---------|
| `whales` | Master list whale + telegram chat_id |
| `tokens` | Master list token + coingecko_id |
| `token_prices` | **Latest price** (single source of truth) |
| `balances` | Current balance decimal (no price/USD) |

## 🚀 Quick Start

### Option A: Use SQL directly

```bash
# Install DuckDB
brew install duckdb

# Create database
duckdb whale_tracker.duckdb < schema/001_create_tables.sql

# Load sample data
duckdb whale_tracker.duckdb < sql/002_sample_data.sql

# Query
duckdb whale_tracker.duckdb
D SELECT * FROM latest_holdings;
```

### Option B: Use Go application

```bash
# Install dependencies
go mod tidy
go get github.com/marcboeker/go-duckdb

# Run
go run cmd/server/main.go

# Build
go build -o whale-tracker cmd/server/main.go
./whale-tracker
```

## 📊 Common Queries (SQL)

```sql
-- All holdings with USD (auto-calculated)
SELECT * FROM latest_holdings;

-- Total portfolio per whale
SELECT whale, SUM(usd_value) as total_usd
FROM latest_holdings
GROUP BY whale;

-- Big changes (alert candidates)
SELECT * FROM big_changes;
```

## 💻 Go Usage

```go
// Open database
database, _ := db.Open("whale_tracker.duckdb")
defer database.Close()

// Create service
svc := service.New(database)

// Insert whale
svc.InsertWhale(service.Whale{
    Address: "0x...",
    Label: "Binance Cold Wallet",
})

// Update price
svc.UpdateTokenPrice(service.TokenPrice{
    TokenAddress: "0x...",
    PriceUSD: 3000.0,
    Source: "coingecko",
})

// Update balance
svc.UpdateBalance(service.Balance{
    WhaleAddress: "0x...",
    TokenAddress: "0x...",
    BalanceDecimal: 50000.0,
})

// Get latest holdings
holdings, _ := svc.GetLatestHoldings()
```

## 🔄 Workflow

```
┌─────────────────────────────────────────────────────────────┐
│                    Price Update Loop                         │
│                    (setiap 5-10 menit)                       │
├─────────────────────────────────────────────────────────────┤
│  1. Fetch price dari Coingecko API                          │
│  2. UPSERT token_prices (replace existing)                   │
│  → USD value di latest_holdings auto-update!                │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                    Balance Update Loop                       │
│                    (setiap 30 menit)                         │
├─────────────────────────────────────────────────────────────┤
│  1. Cek balance whale × token via RPC                      │
│  2. UPDATE balances (set prev_balance, change_percent)      │
│  3. USD value auto-update via VIEW (join token_prices)      │
│  4. Kirim Telegram alert jika change > 10%                  │
└─────────────────────────────────────────────────────────────┘
```

## 🔗 Resources

- DuckDB Docs: https://duckdb.org/docs/
- DuckDB Go: https://github.com/marcboeker/go-duckdb
- Coingecko API: https://www.coingecko.com/en/api
- Whale Addresses: https://www.whale-alert.io/
