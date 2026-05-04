package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

func main() {
	dbPath := "data/whale_tracker.duckdb"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	// Open database
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		fmt.Printf("❌ Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Printf("🚀 Starting DuckDB UI...\n")
	fmt.Printf("📁 Database: %s\n", dbPath)

	// Install and load UI extension
	if _, err := db.Exec("INSTALL ui; LOAD ui;"); err != nil {
		fmt.Printf("❌ Error loading UI extension: %v\n", err)
		os.Exit(1)
	}

	// Start UI
	if _, err := db.Exec("CALL start_ui();"); err != nil {
		fmt.Printf("❌ Error starting UI: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ UI running at: http://localhost:4213\n")
	fmt.Printf("\nPress Ctrl+C to stop\n")

	// Keep connection alive
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := db.Ping(); err != nil {
			fmt.Printf("❌ Connection lost: %v\n", err)
			break
		}
	}
}
