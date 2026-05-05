# 🐋 Whale Tracker

Cryptocurrency whale wallet tracker with live blockchain data, powered by DuckDB and Go.

## Features

- 🔄 **Live balances** from Ethereum blockchain via RPC
- 💰 **Price tracking** via Coingecko API
- 📊 **Portfolio analytics** with SQL queries
- 🚨 **Change detection** (>10% threshold)
- 🔁 **RPC rotation** to avoid rate limits
- 🔍 **Top holders discovery** via RPC scan (free) or Bitquery API
- 🌐 **Web UI** (DuckDB)

## Project Structure

```
whale-tracker/
├── cmd/
│   ├── server/main.go     # Main tracker service
│   ├── ui/main.go         # DuckDB web UI
│   ├── query/main.go      # Query CLI
│   ├── bitquery/main.go   # Bitquery token holders CLI
│   └── rpc-holders/main.go # RPC token holders scanner (free)
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

# Copy config and env templates
cp config.example.yaml config.yaml
cp .env.example .env

# Edit files
vim config.yaml  # Add your whales and tokens
vim .env         # Add BITQUERY_API_KEY
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
| `./bin/whale-rpc-holders <token_address> [options]` | Scan token holders via RPC (free) |
| `./bin/whale-bitquery <token_address> [limit]` | Fetch top token holders (uses .env) |
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

## Token Holders Discovery

Two options available:

### 1. RPC Scan (Free, No API Key) ⭐ Recommended

```bash
# Display holders only
./bin/whale-rpc-holders 0x6982508145454Ce325dDbE47a25d4ec3d2311933 --symbol PEPE

# Add new holders to config
./bin/whale-rpc-holders 0x6982508145454Ce325dDbE47a25d4ec3d2311933 --symbol PEPE --upsert

# Update labels of existing whales (no new entries)
./bin/whale-rpc-holders 0x6982508145454Ce325dDbE47a25d4ec3d2311933 --symbol PEPE --update-labels

# Scan more blocks for better accuracy
./bin/whale-rpc-holders 0x6982508145454Ce325dDbE47a25d4ec3d2311933 --blocks 50000 --symbol PEPE
```

**Options:**
- `--symbol, -s` - Token symbol (default: TOKEN)
- `--blocks, -b` - Number of blocks to scan (default: 10000)
- `--upsert, -u` - Add new holders + update existing labels
- `--update-labels, -U` - Update labels only (no new entries)

**Note:** RPC scan captures holders active in recent blocks only. For complete historical data, use Bitquery API.

### 2. Bitquery API (Requires API Key)

```bash
# 1. Add API key to .env file:
echo "BITQUERY_API_KEY=your_api_key_here" > .env

# 2. Display holders only:
./bin/whale-bitquery 0x6982508145454Ce325dDbE47a25d4ec3d2311933 50

# 3. Add new holders:
./bin/whale-bitquery 0x6982508145454Ce325dDbE47a25d4ec3d2311933 50 --upsert

# 4. Update existing labels only:
./bin/whale-bitquery 0x6982508145454Ce325dDbE47a25d4ec3d2311933 --update-labels
```

**Options:**
- `--upsert, -u` - Add new holders + update existing labels
- `--update-labels, -U` - Update labels only (no new entries)

### Label Format

Both methods use smart labeling based on holding percentage:

| Percentage | Label | Example |
|------------|-------|---------|
| ≥30% | Mega Whale | `PEPE Mega Whale #1` |
| ≥10% | Whale | `PEPE Whale #2` |
| ≥5% | Dolphin | `PEPE Dolphin #3` |
| ≥1% | Fish | `PEPE Fish #4` |
| <1% | Holder | `PEPE Holder #5` |

**Exchange Detection:**
Known exchanges are automatically labeled:
- `PEPE Binance` (not `0x... Binance`)
- `PEPE Bitfinex`
- `PEPE Circle`
- etc.

## Tech Stack

- **Go** - Application logic
- **DuckDB** - Database (single file, embedded)
- **go-ethereum** - Ethereum RPC client
- **Coingecko API** - Price data
- **Bitquery API** - Token holders discovery
- **YAML** - Configuration

## License

MIT
