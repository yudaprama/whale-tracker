package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/aquasecurity/table"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

const (
	configFile = "config.yaml"
)

// ERC20 Transfer event signature
var transferEventID = common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

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

type HolderInfo struct {
	Address     string
	Balance     *big.Int
	Percent     float64
	TokenSymbol string
	Decimals    int
}

// Known exchange addresses
var knownExchanges = map[string]string{
	"0x28c6c06298d514db089934071355e5743bf21d60": "Binance",
	"0x3f5ce5fbfe3e9af3971dd833d26ba9b5c936f0be": "Binance",
	"0x2b8f1b796ee9ad21ce014197e58daeeb8666d091": "Bitfinex",
	"0x21a31ee1afc51d94c2effcaa1ad2df12b723a08e": "Coinbase",
	"0x88e6a0c2ddd26feeb64f039a2c41296fcb3f5640": "Circle",
}

func main() {
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	if os.Args[1] == "--help" || os.Args[1] == "-h" {
		printUsage()
		os.Exit(0)
	}

	tokenAddress := os.Args[1]
	limit := 100
	blocks := 10000 // Number of recent blocks to scan
	upsert := false
	updateLabels := false
	tokenSymbol := "TOKEN"
	decimals := 18

	// Parse flags
	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--upsert", "-u":
			upsert = true
		case "--update-labels", "-U":
			updateLabels = true
		case "--symbol", "-s":
			if i+1 < len(os.Args) {
				tokenSymbol = strings.ToUpper(os.Args[i+1])
				i++
			}
		case "--blocks", "-b":
			if i+1 < len(os.Args) {
				if b, err := strconv.Atoi(os.Args[i+1]); err == nil && b > 0 {
					blocks = b
					i++
				}
			}
		case "--help", "-h":
			printUsage()
			os.Exit(0)
		default:
			if l, err := strconv.Atoi(os.Args[i]); err == nil && l > 0 {
				limit = l
			}
		}
	}

	fmt.Printf("🔍 Scaning last %d blocks for top holders of %s\n\n", blocks, tokenAddress)

	holders, err := scanTokenHolders(tokenAddress, blocks, limit)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	if len(holders) == 0 {
		fmt.Println("No holders found")
		return
	}

	displayHolders(holders, decimals)

	if upsert {
		upsertToConfig(holders, tokenSymbol, decimals, true)
	} else if updateLabels {
		upsertToConfig(holders, tokenSymbol, decimals, false)
	} else {
		displayConfig(holders, tokenSymbol)
	}
}

func printUsage() {
	fmt.Println("Usage: whale-rpc-holders <token_address> [options]")
	fmt.Println("\nExamples:")
	fmt.Println("  whale-rpc-holders 0x6982508145454Ce325dDbE47a25d4ec3d2311933")
	fmt.Println("  whale-rpc-holders 0x6982508145454Ce325dDbE47a25d4ec3d2311933 --symbol PEPE")
	fmt.Println("  whale-rpc-holders 0x6982508145454Ce325dDbE47a25d4ec3d2311933 50 --blocks 5000")
	fmt.Println("  whale-rpc-holders 0x6982508145454Ce325dDbE47a25d4ec3d2311933 --upsert")
	fmt.Println("  whale-rpc-holders 0x6982508145454Ce325dDbE47a25d4ec3d2311933 --update-labels")
	fmt.Println("\nOptions:")
	fmt.Println("  --symbol, -s       Token symbol (default: TOKEN)")
	fmt.Println("  --blocks, -b       Number of blocks to scan (default: 10000)")
	fmt.Println("  --upsert, -u      Add new holders to config.yaml")
	fmt.Println("  --update-labels, -U  Update labels of existing whales only")
	fmt.Println("  --help, -h        Show this help")
	fmt.Println("\nNote: Scans Transfer events from recent blocks")
	fmt.Println("      More blocks = more accurate but slower")
}

func generateLabel(address, tokenSymbol string, rank int, percent float64) string {
	if name, ok := knownExchanges[strings.ToLower(address)]; ok {
		return fmt.Sprintf("%s %s", tokenSymbol, name)
	}

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

func scanTokenHolders(tokenAddress string, blockCount, limit int) ([]HolderInfo, error) {
	// Get RPC URL from config or use default
	rpcURL := "https://ethereum.publicnode.com"
	if cfg, err := readConfig(); err == nil && len(cfg.RPCUrls) > 0 {
		rpcURL = cfg.RPCUrls[0]
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %w", err)
	}
	defer client.Close()

	// Get current block
	latestBlock, err := client.BlockNumber(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get latest block: %w", err)
	}

	fromBlock := latestBlock - uint64(blockCount)
	if fromBlock < 0 {
		fromBlock = 0
	}

	fmt.Printf("Scanning blocks %d to %d...\n", fromBlock, latestBlock)

	tokenContract := common.HexToAddress(tokenAddress)
	holders := make(map[string]*big.Int)

	// Scan for Transfer events
	query := ethereum.FilterQuery{
		Addresses: []common.Address{tokenContract},
		FromBlock: big.NewInt(int64(fromBlock)),
		ToBlock:   big.NewInt(int64(latestBlock)),
		Topics:    [][]common.Hash{{transferEventID}},
	}

	logs, err := client.FilterLogs(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to filter logs: %w", err)
	}

	fmt.Printf("Found %d transfer events, processing...\n", len(logs))

	for _, vLog := range logs {
		// Transfer event: indexed from, indexed to, value
		if len(vLog.Topics) < 3 {
			continue
		}

		from := common.HexToAddress(vLog.Topics[1].Hex())
		to := common.HexToAddress(vLog.Topics[2].Hex())
		value := new(big.Int).SetBytes(vLog.Data)

		// Skip zero address (mint/burn)
		if from == (common.Address{}) || to == (common.Address{}) {
			continue
		}

		// Subtract from sender
		if holders[from.Hex()] != nil {
			holders[from.Hex()] = new(big.Int).Sub(holders[from.Hex()], value)
		}

		// Add to receiver
		if holders[to.Hex()] == nil {
			holders[to.Hex()] = big.NewInt(0)
		}
		holders[to.Hex()] = new(big.Int).Add(holders[to.Hex()], value)
	}

	// Convert to slice and sort by balance
	var holderList []HolderInfo
	for addr, balance := range holders {
		if balance.Sign() > 0 { // Only positive balances
			holderList = append(holderList, HolderInfo{
				Address:  addr,
				Balance:  balance,
				Decimals: 18,
			})
		}
	}

	// Sort by balance (descending)
	for i := 0; i < len(holderList); i++ {
		for j := i + 1; j < len(holderList); j++ {
			if holderList[j].Balance.Cmp(holderList[i].Balance) > 0 {
				holderList[i], holderList[j] = holderList[j], holderList[i]
			}
		}
	}

	// Limit results
	if len(holderList) > limit {
		holderList = holderList[:limit]
	}

	// Calculate percentages
	totalBalance := big.NewInt(0)
	for _, h := range holderList {
		totalBalance.Add(totalBalance, h.Balance)
	}

	for i := range holderList {
		if totalBalance.Sign() > 0 {
			ratio := new(big.Float).Quo(
				new(big.Float).SetInt(holderList[i].Balance),
				new(big.Float).SetInt(totalBalance),
			)
			pct, _ := ratio.Float64()
			holderList[i].Percent = pct * 100
		}
	}

	return holderList, nil
}

func displayHolders(holders []HolderInfo, decimals int) {
	t := table.New(os.Stdout)
	t.SetRowLines(false)
	t.SetHeaders("Rank", "Address", "Balance", "Holding %")

	for i, h := range holders {
		rank := fmt.Sprintf("#%d", i+1)
		address := shortenAddress(h.Address)
		balance := formatBalance(h.Balance, decimals)
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
		if i >= 10 {
			break
		}
		label := generateLabel(h.Address, tokenSymbol, i+1, h.Percent)
		fmt.Printf("  - address: \"%s\"\n", h.Address)
		fmt.Printf("    label: \"%s\"\n", label)
		fmt.Printf("    active: true\n")
	}
}

func upsertToConfig(holders []HolderInfo, tokenSymbol string, decimals int, addNew bool) {
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
		label := generateLabel(h.Address, tokenSymbol, i+1, h.Percent)

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

func formatBalance(balance *big.Int, decimals int) string {
	if balance == nil || balance.Sign() == 0 {
		return "0"
	}

	// Convert to float for formatting
	floatBalance := new(big.Float).SetInt(balance)
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	adjusted := new(big.Float).Quo(floatBalance, divisor)

	f, _ := adjusted.Float64()
	if f > 1000000 {
		return fmt.Sprintf("%.2fM", f/1000000)
	} else if f > 1000 {
		return fmt.Sprintf("%.2fK", f/1000)
	}
	return fmt.Sprintf("%.2f", f)
}
