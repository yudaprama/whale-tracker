package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"
)

type Query struct {
	Name        string
	Description string
	SQL         string
}

var queries = map[string]Query{
	"holdings": {
		Name:        "holdings",
		Description: "All current holdings with USD values",
		SQL:         "SELECT * FROM latest_holdings ORDER BY usd_value DESC",
	},
	"portfolio": {
		Name:        "portfolio",
		Description: "Total portfolio per whale",
		SQL: `
			SELECT
			  w.label as whale,
			  SUM(b.balance_decimal * tp.price_usd) as total_usd,
			  COUNT(*) as token_count
			FROM balances b
			JOIN whales w ON b.whale_address = w.address
			JOIN token_prices tp ON b.token_address = tp.token_address
			WHERE b.balance_decimal > 0
			GROUP BY w.label
			ORDER BY total_usd DESC
		`,
	},
	"changes": {
		Name:        "changes",
		Description: "Big changes (>10%)",
		SQL:         "SELECT * FROM big_changes ORDER BY ABS(change_percent) DESC",
	},
	"tokens": {
		Name:        "tokens",
		Description: "Token distribution across whales",
		SQL: `
			SELECT
			  t.symbol as token,
			  SUM(b.balance_decimal * tp.price_usd) as total_usd,
			  COUNT(DISTINCT b.whale_address) as holder_count
			FROM balances b
			JOIN tokens t ON b.token_address = t.address
			JOIN token_prices tp ON b.token_address = tp.token_address
			WHERE b.balance_decimal > 0
			GROUP BY t.symbol
			ORDER BY total_usd DESC
		`,
	},
	"prices": {
		Name:        "prices",
		Description: "Current token prices",
		SQL: `
			SELECT t.symbol, tp.price_usd, tp.updated_at
			FROM token_prices tp
			JOIN tokens t ON tp.token_address = t.address
			ORDER BY tp.price_usd DESC
		`,
	},
	"whales": {
		Name:        "whales",
		Description: "List all tracked whales",
		SQL:         "SELECT * FROM whales WHERE active = true ORDER BY label",
	},
}

type OutputFormat string

const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatCSV   OutputFormat = "csv"
)

func main() {
	if len(os.Args) < 2 {
		listQueries()
		os.Exit(0)
	}

	queryName := os.Args[1]
	outputFormat := FormatTable
	outputFile := ""

	// Parse flags
	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--json", "-j":
			outputFormat = FormatJSON
		case "--csv", "-c":
			outputFormat = FormatCSV
		case "--out", "-o":
			if i+1 < len(os.Args) {
				outputFile = os.Args[i+1]
				i++
			}
		}
	}

	// Open database
	db, err := sql.Open("duckdb", "data/whale_tracker.duckdb")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Find query
	query, ok := queries[queryName]
	if !ok {
		log.Fatalf("Unknown query: %s\nAvailable: %s", queryName, getQueryNames())
	}

	// Run query
	rows, err := db.Query(query.SQL)
	if err != nil {
		log.Fatalf("Query error: %v", err)
	}
	defer rows.Close()

	// Get results
	results, err := parseRows(rows)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	if len(results.Rows) == 0 {
		fmt.Println("No results")
		return
	}

	// Output
	var output string
	switch outputFormat {
	case FormatJSON:
		output = formatJSON(results)
	case FormatCSV:
		output = formatCSV(results)
	default:
		output = formatTable(results)
	}

	// Write to file or stdout
	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
			log.Fatalf("Write error: %v", err)
		}
		fmt.Printf("✅ Output written to %s\n", outputFile)
	} else {
		fmt.Print(output)
	}
}

type QueryResult struct {
	Columns []string
	Rows    []map[string]interface{}
}

func parseRows(rows *sql.Rows) (*QueryResult, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results QueryResult
	results.Columns = columns

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results.Rows = append(results.Rows, row)
	}

	return &results, nil
}

func formatTable(results *QueryResult) string {
	var sb strings.Builder

	// Calculate column widths
	widths := make([]int, len(results.Columns))
	for i, col := range results.Columns {
		widths[i] = len(col)
	}
	for _, row := range results.Rows {
		for i, col := range results.Columns {
			val := fmt.Sprintf("%v", row[col])
			if len(val) > widths[i] {
				widths[i] = len(val)
			}
		}
	}

	// Header
	for i, col := range results.Columns {
		sb.WriteString(fmt.Sprintf("%-*s", widths[i]+2, col))
	}
	sb.WriteString("\n")

	// Separator
	for _, w := range widths {
		sb.WriteString(strings.Repeat("-", w+2))
	}
	sb.WriteString("\n")

	// Rows
	for _, row := range results.Rows {
		for i, col := range results.Columns {
			val := fmt.Sprintf("%v", row[col])
			sb.WriteString(fmt.Sprintf("%-*s", widths[i]+2, val))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func formatJSON(results *QueryResult) string {
	data, _ := json.MarshalIndent(results.Rows, "", "  ")
	return string(data) + "\n"
}

func formatCSV(results *QueryResult) string {
	var sb strings.Builder
	writer := csv.NewWriter(&sb)

	// Header
	writer.Write(results.Columns)

	// Rows
	for _, row := range results.Rows {
		record := make([]string, len(results.Columns))
		for i, col := range results.Columns {
			record[i] = fmt.Sprintf("%v", row[col])
		}
		writer.Write(record)
	}

	writer.Flush()
	return sb.String()
}

func listQueries() {
	fmt.Println("🐋 Whale Tracker Queries")
	fmt.Println("\nUsage: make query <name> [flags]")
	fmt.Println("\nAvailable queries:")

	names := make([]string, 0, len(queries))
	for name := range queries {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		q := queries[name]
		fmt.Printf("  %-12s %s\n", name, q.Description)
	}

	fmt.Println("\nFlags:")
	fmt.Println("  --json, -j    Output as JSON")
	fmt.Println("  --csv, -c     Output as CSV")
	fmt.Println("  --out FILE    Write to file")
}

func getQueryNames() string {
	names := make([]string, 0, len(queries))
	for name := range queries {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
