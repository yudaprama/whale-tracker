package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/aquasecurity/table"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

const (
	bitqueryAPI = "https://graphql.bitquery.io/ethereum"
	configFile  = "config.yaml"
)

type BitqueryResponse struct {
	Data struct {
		Ethereum struct {
			Transfers []struct {
				Receiver  struct {
					Address string `json:"address"`
				} `json:"receiver"`
				Count    int     `json:"count"`
				Sum      float64 `json:"sum"`
				Currency struct {
					Symbol   string `json:"symbol"`
					Address  string `json:"address"`
					Decimals int    `json:"decimals"`
				} `json:"currency"`
			} `json:"transfers"`
		} `json:"ethereum"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// Config structures matching config.yaml
type Config struct {
	RPCUrls        []string `yaml:"rpc_urls"`
	UpdateInterval int      `yaml:"update_interval"`
	Whales         []Whale  `yaml:"whales"`
	Tokens         []Token  `yaml:"tokens"`
}

type Whale struct {
	Address        string `yaml:"address"`
	Label          string `yaml:"label"`
	TelegramChatID string `yaml:"telegram_chat_id,omitempty"`
	Active         bool   `yaml:"active"`
}

type Token struct {
	Address     string `yaml:"address"`
	Symbol      string `yaml:"symbol"`
	Decimals    int    `yaml:"decimals"`
	Category    string `yaml:"category"`
	CoingeckoID string `yaml:"coingecko_id"`
}

func main() {
	// Load .env file
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Check for help flag
	if os.Args[1] == "--help" || os.Args[1] == "-h" {
		printUsage()
		os.Exit(0)
	}

	tokenAddress := os.Args[1]
	apiKey := ""
	limit := 100
	upsert := false
	updateLabels := false

	// Parse flags
	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--upsert", "-u":
			upsert = true
		case "--update-labels", "-U":
			updateLabels = true
		case "--help", "-h":
			printUsage()
			os.Exit(0)
		default:
			// Check if it's a number (limit)
			if l, err := strconv.Atoi(os.Args[i]); err == nil && l > 0 {
				if i == 2 || (i == 3 && apiKey != "") {
					limit = l
				}
			} else if apiKey == "" {
				apiKey = os.Args[i]
			}
		}
	}

	// Fallback to env variable if API key not provided
	if apiKey == "" {
		apiKey = os.Getenv("BITQUERY_API_KEY")
	}

	if apiKey == "" {
		log.Fatal("Error: BITQUERY_API_KEY not found. Provide it as argument or set in .env file")
	}

	fmt.Printf("🔍 Fetching top %d holders for %s\n\n", limit, tokenAddress)

	holders, tokenSymbol, err := fetchTokenHolders(tokenAddress, apiKey, limit)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	if len(holders) == 0 {
		fmt.Println("No holders found")
		return
	}

	displayHolders(holders)

	if upsert {
		upsertToConfig(holders, tokenSymbol, true)
	} else if updateLabels {
		upsertToConfig(holders, tokenSymbol, false)
	} else {
		displayConfig(holders, tokenSymbol)
	}
}

func printUsage() {
	fmt.Println("Usage: whale-bitquery <token_address> [options]")
	fmt.Println("\nExamples:")
	fmt.Println("  whale-bitquery 0x6982508145454Ce325dDbE47a25d4ec3d2311933")
	fmt.Println("  whale-bitquery 0x6982508145454Ce325dDbE47a25d4ec3d2311933 50")
	fmt.Println("  whale-bitquery 0x6982508145454Ce325dDbE47a25d4ec3d2311933 --upsert")
	fmt.Println("  whale-bitquery 0x6982508145454Ce325dDbE47a25d4ec3d2311933 --update-labels")
	fmt.Println("\nOptions:")
	fmt.Println("  --upsert, -u      Add new holders to config.yaml")
	fmt.Println("  --update-labels, -U  Update labels of existing whales only")
	fmt.Println("  --help, -h        Show this help")
	fmt.Println("\nAPI Key sources (in order):")
	fmt.Println("  1. BITQUERY_API_KEY environment variable")
	fmt.Println("  2. .env file")
	fmt.Println("\nGet API key: https://bitquery.io/")
}

type HolderInfo struct {
	Address     string
	Balance     string
	Percent     float64
	TokenSymbol string
}

// Known exchange addresses for better labels
var knownExchanges = map[string]string{
	"0x28c6c06298d514db089934071355e5743bf21d60": "Binance",
	"0x3f5ce5fbfe3e9af3971dd833d26ba9b5c936f0be": "Binance",
	"0x2b8f1b796ee9ad21ce014197e58daeeb8666d091": "Bitfinex",
	"0x21a31ee1afc51d94c2effcaa1ad2df12b723a08e": "Coinbase",
	"0x88e6a0c2ddd26feeb64f039a2c41296fcb3f5640": "Circle",
	"0x742d35cc6634c0532925a3b844bc9e7595f0beeb": "PEPE Whale",
}

func generateLabel(address, tokenSymbol string, rank int, percent float64) string {
	// Check if it's a known exchange
	if name, ok := knownExchanges[strings.ToLower(address)]; ok {
		return fmt.Sprintf("%s %s", tokenSymbol, name)
	}

	// Generate tier-based label
	suffix := ""
	if percent >= 30 {
		suffix = "Mega Whale"
	} else if percent >= 10 {
		suffix = "Whale"
	} else if percent >= 5 {
		suffix = "Dolphin"
	} else if percent >= 1 {
		suffix = "Fish"
	} else if rank <= 3 {
		suffix = "Top"
	} else {
		suffix = "Holder"
	}

	if rank <= 3 && (suffix == "Top" || suffix == "Fish") {
		return fmt.Sprintf("%s %s #%d", tokenSymbol, suffix, rank)
	}

	if suffix == "Top" {
		return fmt.Sprintf("%s %s", tokenSymbol, suffix)
	}

	return fmt.Sprintf("%s %s #%d", tokenSymbol, suffix, rank)
}

func fetchTokenHolders(tokenAddress, apiKey string, limit int) ([]HolderInfo, string, error) {
	// Use transfers aggregation to find top holders
	// Groups by receiver and sums incoming amounts
	query := fmt.Sprintf(`{
  ethereum(network: ethereum) {
    transfers(
      currency: {is: "%s"}
      options: {limit: %d, desc: "count"}
    ) {
      receiver {
        address
      }
      count
      sum: amount(calculate: sum)
      currency {
        symbol
        decimals
      }
    }
  }
}`, tokenAddress, limit)

	reqBody := map[string]string{
		"query": query,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", bitqueryAPI, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", apiKey)
	req.Header.Set("User-Agent", "Whale-Tracker/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	// Check for non-JSON response (API errors)
	if !bytes.HasPrefix(body, []byte("{")) {
		return nil, "", fmt.Errorf("API error: %s", string(body))
	}

	// Parse response
	var result BitqueryResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, "", fmt.Errorf("parse error: %w", err)
	}

	// Check for errors
	if len(result.Data.Ethereum.Transfers) == 0 && len(result.Errors) > 0 {
		return nil, "", fmt.Errorf("API error: %s", result.Errors[0].Message)
	}

	// Get token symbol
	tokenSymbol := "UNKNOWN"
	if len(result.Data.Ethereum.Transfers) > 0 {
		tokenSymbol = result.Data.Ethereum.Transfers[0].Currency.Symbol
	}

	// Convert to HolderInfo
	var holders []HolderInfo
	totalSum := 0.0

	for _, t := range result.Data.Ethereum.Transfers {
		totalSum += t.Sum
	}

	for _, t := range result.Data.Ethereum.Transfers {
		percent := 0.0
		if totalSum > 0 {
			percent = (t.Sum / totalSum) * 100
		}

		holders = append(holders, HolderInfo{
			Address:     t.Receiver.Address,
			Balance:     fmt.Sprintf("%.0f", t.Sum),
			Percent:     percent,
			TokenSymbol: tokenSymbol,
		})
	}

	return holders, tokenSymbol, nil
}

func displayHolders(holders []HolderInfo) {
	t := table.New(os.Stdout)
	t.SetRowLines(false)
	t.SetHeaders("Rank", "Address", "Balance", "Holding %")

	for i, h := range holders {
		rank := fmt.Sprintf("#%d", i+1)
		address := shortenAddress(h.Address)
		balance := formatBalance(h.Balance)
		percent := fmt.Sprintf("%.4f%%", h.Percent)

		t.AddRow(rank, address, balance, percent)
	}
	t.Render()
}

func displayConfig(holders []HolderInfo, tokenSymbol string) {
	fmt.Println("\n" + strings.Repeat("─", 70))
	fmt.Println("📋 Config to add to config.yaml:")
	fmt.Println(strings.Repeat("─", 70))

	fmt.Printf("\n# Top %d holders for %s\n", len(holders), tokenSymbol)
	fmt.Println("whales:")

	for i, h := range holders {
		if i >= 10 { // Only show top 10
			break
		}
		label := generateLabel(h.Address, h.TokenSymbol, i+1, h.Percent)
		fmt.Printf("  - address: \"%s\"\n", h.Address)
		fmt.Printf("    label: \"%s\"\n", label)
		fmt.Printf("    active: true\n")
	}
}

func upsertToConfig(holders []HolderInfo, tokenSymbol string, addNew bool) {
	// Read existing config
	cfg, err := readConfig()
	if err != nil {
		log.Fatalf("Error reading config: %v", err)
	}

	// Build map of existing addresses and their indices
	existingAddresses := make(map[string]int)
	for i, w := range cfg.Whales {
		existingAddresses[strings.ToLower(w.Address)] = i
	}

	added := 0
	updated := 0

	// Process holders
	for i, h := range holders {
		if i >= 10 {
			break
		}
		addrLower := strings.ToLower(h.Address)
		label := generateLabel(h.Address, h.TokenSymbol, i+1, h.Percent)

		if idx, exists := existingAddresses[addrLower]; exists {
			// Update existing label
			if cfg.Whales[idx].Label != label {
				cfg.Whales[idx].Label = label
				updated++
			}
		} else if addNew {
			// Add new holder
			cfg.Whales = append(cfg.Whales, Whale{
				Address: h.Address,
				Label:   label,
				Active:  true,
			})
			existingAddresses[addrLower] = len(cfg.Whales) - 1
			added++
		}
	}

	// Write back to config
	if err := writeConfig(cfg); err != nil {
		log.Fatalf("Error writing config: %v", err)
	}

	fmt.Println("\n" + strings.Repeat("─", 70))
	if added > 0 || updated > 0 {
		if addNew {
			fmt.Printf("✅ Added %d new holders, updated %d labels in %s\n", added, updated, configFile)
		} else {
			fmt.Printf("✅ Updated %d labels in %s\n", updated, configFile)
		}
	} else {
		fmt.Printf("ℹ️  No changes needed in %s\n", configFile)
	}
	fmt.Println(strings.Repeat("─", 70))
}

func readConfig() (*Config, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func writeConfig(cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0644)
}

func shortenAddress(addr string) string {
	if len(addr) <= 12 {
		return addr
	}
	return addr[:6] + "..." + addr[len(addr)-4:]
}

func formatBalance(balance string) string {
	return balance
}
