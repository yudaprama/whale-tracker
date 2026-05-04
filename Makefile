.PHONY: run ui build clean test help unlock

# Default target
help:
	@echo "🐋 Whale Tracker - Available commands:"
	@echo ""
	@echo "  make run              - Run with live RPC data from blockchain"
	@echo "  ./bin/whale-query     - List all available queries"
	@echo "  ./bin/whale-query Q   - Run query Q (holdings, portfolio, changes, etc)"
	@echo "                         FLAGS: --json, --csv, --out FILE"
	@echo "                         Example: ./bin/whale-query portfolio --json --out report.json"
	@echo "  make ui               - Start DuckDB web UI"
	@echo "  make build            - Build binaries"
	@echo "  make clean            - Clean build artifacts and database"
	@echo "  make test             - Run tests"
	@echo "  make unlock           - Force release database locks"
	@echo ""

# Run with RPC mode
run:
	@echo "🚀 Starting Whale Tracker (live blockchain data)..."
	go run cmd/server/main.go

# Start DuckDB UI
ui:
	@echo "🌐 Starting DuckDB UI..."
	go run cmd/ui/main.go

# Build binaries
build:
	@echo "🔨 Building binaries..."
	@mkdir -p bin
	go build -o bin/whale-tracker cmd/server/main.go
	go build -o bin/whale-ui cmd/ui/main.go
	go build -o bin/whale-query cmd/query/main.go
	@echo "✅ Built: bin/whale-tracker, bin/whale-ui, bin/whale-query"

# Clean
clean:
	@echo "🧹 Cleaning..."
	rm -rf bin/
	rm -f data/*.duckdb data/*.duckdb.wal
	rm -f report.json report.csv
	@echo "✅ Cleaned"

# Run tests
test:
	@echo "🧪 Running tests..."
	go test ./...

# Install dependencies
deps:
	@echo "📦 Installing dependencies..."
	go mod download
	go mod tidy
	@echo "✅ Dependencies installed"

# Force release database lock (use with caution)
unlock:
	@echo "🔓 Releasing database locks..."
	@pkill -9 -f "whale_tracker.duckdb" 2>/dev/null || true
	@rm -f data/whale_tracker.duckdb.wal 2>/dev/null || true
	@echo "✅ Locks released"
