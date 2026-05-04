# 🐋 Whale Tracker

Cryptocurrency whale wallet tracker with live blockchain data, powered by DuckDB and Go.

## Features

- 🔄 **Live balances** from Ethereum blockchain via RPC
- 💰 **Price tracking** via Coingecko API
- 📊 **Portfolio analytics** with SQL queries
- 🚨 **Change detection** (>10% threshold)
- 🔁 **RPC rotation** to avoid rate limits
- 🌐 **Web UI** (DuckDB)

## Project Structure

```
whale-tracker/
├── cmd/
│   ├── server/main.go    # Main tracker service
│   ├── ui/main.go        # DuckDB web UI
│   └── query/main.go     # Query CLI
├── internal/
│   ├── db/db.go          # Database connection & schema
│   └── service/          # Business logic
│       ├── balance.go    # Balance operations
│       ├── coingecko.go  # Price fetching
│       ├── portfolio.go  # Portfolio queries
│       ├── price.go      # Price operations
│       ├── rpc.go        # Blockchain RPC
│       ├── whale.go      # Whale operations
│       └── token.go      # Token operations
├── docs/queries.sql      # Reference queries
├── config.yaml           # Configuration (whales, tokens, RPC)
├── config.example.yaml   # Config template
├── Makefile              # Commands
└── go.mod
```

## Quick Start

### Prerequisites

```bash
# Install Go
brew install go

# Install DuckDB CLI (optional, for UI)
brew install duckdb
```

### Setup

```bash
# Clone and install dependencies
cd whale-tracker
go mod tidy

# Copy and edit config
cp config.example.yaml config.yaml
vim config.yaml  # Add your whales and tokens
```

### Run

```bash
# Build binaries
make build

# Run tracker (live data)
make run
# or: ./bin/whale-tracker

# Start DuckDB web UI
make ui
# or: ./bin/whale-ui
```

## Commands

| Command | Description |
|---------|-------------|
| `make run` | Run tracker with live blockchain data |
| `make ui` | Start DuckDB web UI |
| `./bin/whale-query` | List available queries |
| `./bin/whale-query portfolio` | Run portfolio query |
| `./bin/whale-query holdings --json --out report.json` | Export to JSON |
| `make build` | Build all binaries |
| `make clean` | Clean artifacts and database |
| `make unlock` | Release database locks |

## Configuration

Edit `config.yaml`:

```yaml
# RPC endpoints (rotated automatically)
rpc_urls:
  - "https://ethereum.publicnode.com"
  - "https://rpc.ankr.com/eth"
  - "https://eth.llamarpc.com"

# Update interval (seconds)
update_interval: 30

# Whales to track
whales:
  - address: "0x3f5CE5FBFe3E9af3971dD833D26bA9b5C936f0bE"
    label: "Binance Cold Wallet"
    active: true

# Tokens to track
tokens:
  - address: "0xdac17f958d2ee523a2206206994597c13d831ec7"
    symbol: "USDT"
    decimals: 6
    category: "stablecoin"
    coingecko_id: "tether"
```

## Database Schema

| Table | Purpose |
|-------|---------|
| `whales` | Whale addresses + metadata |
| `tokens` | Token addresses + Coingecko IDs |
| `token_prices` | Latest prices (single source of truth) |
| `balances` | Current balances + change tracking |

## Queries

### Via CLI

```bash
# Portfolio totals
./bin/whale-query portfolio

# All holdings
./bin/whale-query holdings

# Big changes
./bin/whale-query changes

# Export to CSV
./bin/whale-query tokens --csv --out tokens.csv
```

### Via Web UI

```bash
make ui
# Open http://localhost:4213
# Run any SQL query
```

## Available Queries

| Query | Description |
|-------|-------------|
| `holdings` | All holdings with USD values |
| `portfolio` | Total portfolio per whale |
| `changes` | Big changes (>10%) |
| `tokens` | Token distribution |
| `prices` | Current token prices |
| `whales` | Tracked whales |

## Output Formats

```bash
# Table (default)
./bin/whale-query portfolio

# JSON
./bin/whale-query portfolio --json

# CSV
./bin/whale-query portfolio --csv

# Export to file
./bin/whale-query portfolio --json --out report.json
```

## Tech Stack

- **Go** - Application logic
- **DuckDB** - Database (single file, embedded)
- **go-ethereum** - Ethereum RPC client
- **Coingecko API** - Price data
- **YAML** - Configuration

## License

MIT
