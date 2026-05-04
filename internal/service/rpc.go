package service

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ERC20ABI contains the ABI for ERC20 balanceOf function
const ERC20ABI = `[{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"type":"function"}]`

// RPCConfig holds RPC connection configuration
type RPCConfig struct {
	RPCURL string
}

// RPCService handles Ethereum RPC calls
type RPCService struct {
	client *ethclient.Client
	config *RPCConfig
}

// NewRPCService creates a new RPC service
func NewRPCService(rpcURL string) (*RPCService, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %w", err)
	}

	return &RPCService{
		client: client,
		config: &RPCConfig{RPCURL: rpcURL},
	}, nil
}

// Close closes the RPC connection
func (r *RPCService) Close() {
	r.client.Close()
}

// GetETHBalance gets the native ETH balance for an address
func (r *RPCService) GetETHBalance(address string) (*big.Int, error) {
	account := common.HexToAddress(address)
	balance, err := r.client.BalanceAt(context.Background(), account, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get ETH balance: %w", err)
	}
	return balance, nil
}

// GetTokenBalance gets the ERC20 token balance for an address
func (r *RPCService) GetTokenBalance(tokenAddress, holderAddress string, decimals int) (float64, error) {
	tokenAddr := common.HexToAddress(tokenAddress)
	holderAddr := common.HexToAddress(holderAddress)

	// Parse ERC20 ABI
	parsedABI, err := abi.JSON(strings.NewReader(ERC20ABI))
	if err != nil {
		return 0, fmt.Errorf("failed to parse ABI: %w", err)
	}

	// Pack the balanceOf function call data
	data, err := parsedABI.Pack("balanceOf", holderAddr)
	if err != nil {
		return 0, fmt.Errorf("failed to pack data: %w", err)
	}

	// Create call message
	msg := ethereum.CallMsg{
		To:   &tokenAddr,
		Data: data,
	}

	// Call the contract
	result, err := r.client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to call contract: %w", err)
	}

	// Unpack the result
	var balance *big.Int
	err = parsedABI.UnpackIntoInterface(&balance, "balanceOf", result)
	if err != nil {
		return 0, fmt.Errorf("failed to unpack result: %w", err)
	}

	// Convert to decimal based on token decimals
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	balanceDecimal := new(big.Float).Quo(new(big.Float).SetInt(balance), new(big.Float).SetInt(divisor))

	resultFloat, _ := balanceDecimal.Float64()
	return resultFloat, nil
}

// FetchWhaleBalances fetches balances for all whales and tokens from RPC
func (s *Service) FetchWhaleBalances(rpcURL string) error {
	rpc, err := NewRPCService(rpcURL)
	if err != nil {
		return err
	}
	defer rpc.Close()

	// Get all whales
	whales, err := s.ListWhales()
	if err != nil {
		return err
	}

	// Get all tokens
	tokens, err := s.ListTokens()
	if err != nil {
		return err
	}

	// Fetch balances
	count := 0
	for _, whale := range whales {
		for _, token := range tokens {
			var balanceDecimal float64
			var err error

			if token.Symbol == "ETH" || token.Address == "0x0000000000000000000000000000000000000000" {
				// Native ETH
				balance, err := rpc.GetETHBalance(whale.Address)
				if err != nil {
					continue // Skip if error
				}
				balanceDecimal, _ = new(big.Float).SetInt(balance).Float64()
			} else {
				// ERC20 Token
				if token.Decimals == 0 {
					continue
				}
				balanceDecimal, err = rpc.GetTokenBalance(token.Address, whale.Address, token.Decimals)
				if err != nil {
					continue // Skip if error
				}
			}

			// Skip zero balances
			if balanceDecimal == 0 {
				continue
			}

			// Update balance in database
			balance := Balance{
				WhaleAddress:   whale.Address,
				TokenAddress:   token.Address,
				BalanceDecimal: balanceDecimal,
			}

			if err := s.UpdateBalance(balance); err != nil {
				fmt.Printf("Error updating balance: %v\n", err)
			} else {
				count++
			}
		}
	}

	fmt.Printf("✅ Fetched and updated %d whale balances from RPC\n", count)
	return nil
}
