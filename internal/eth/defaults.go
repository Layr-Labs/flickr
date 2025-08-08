package eth

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// DefaultContractAddresses contains default contract addresses for a chain
type DefaultContractAddresses struct {
	ReleaseManager string
}

// GetDefaultContractAddresses returns the default contract addresses for a given chain ID
func GetDefaultContractAddresses(chainID uint64) (*DefaultContractAddresses, error) {
	switch chainID {
	case 11155111: // Sepolia
		return &DefaultContractAddresses{
			ReleaseManager: "0xd9Cb89F1993292dEC2F973934bC63B0f2A702776",
		}, nil
	case 31337: // Local/Hardhat
		return &DefaultContractAddresses{
			ReleaseManager: "0xd9Cb89F1993292dEC2F973934bC63B0f2A702776",
		}, nil
	case 1: // Mainnet
		return &DefaultContractAddresses{
			ReleaseManager: "0x0000000000000000000000000000000000000000", // To be updated
		}, nil
	default:
		return nil, fmt.Errorf("default contract addresses not found for chain ID %d", chainID)
	}
}

// GetChainID retrieves the chain ID from an RPC endpoint
func GetChainID(rpcURL string) (uint64, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to Ethereum client: %w", err)
	}
	defer client.Close()

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return 0, fmt.Errorf("failed to get chain ID: %w", err)
	}

	return chainID.Uint64(), nil
}

// GetReleaseManagerAddress returns the ReleaseManager address
// It uses the provided address if non-empty, otherwise uses chain defaults
func GetReleaseManagerAddress(rpcURL string, providedAddress string) (common.Address, error) {
	// If address is provided, use it
	if providedAddress != "" && providedAddress != "0x0000000000000000000000000000000000000000" {
		return common.HexToAddress(providedAddress), nil
	}

	// Otherwise, get from chain defaults
	chainID, err := GetChainID(rpcURL)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to get chain ID: %w", err)
	}

	defaults, err := GetDefaultContractAddresses(chainID)
	if err != nil {
		return common.Address{}, fmt.Errorf("no default ReleaseManager for chain %d: %w", chainID, err)
	}

	return common.HexToAddress(defaults.ReleaseManager), nil
}

// NetworkInfo contains information about the connected network
type NetworkInfo struct {
	ChainID        *big.Int
	ChainName      string
	ReleaseManager common.Address
}

// GetNetworkInfo retrieves information about the connected network
func GetNetworkInfo(rpcURL string, releaseManager string) (*NetworkInfo, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum client: %w", err)
	}
	defer client.Close()

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	chainName := getChainName(chainID.Uint64())
	
	rmAddr, err := GetReleaseManagerAddress(rpcURL, releaseManager)
	if err != nil {
		return nil, err
	}

	return &NetworkInfo{
		ChainID:        chainID,
		ChainName:      chainName,
		ReleaseManager: rmAddr,
	}, nil
}

func getChainName(chainID uint64) string {
	switch chainID {
	case 1:
		return "Ethereum Mainnet"
	case 11155111:
		return "Sepolia Testnet"
	case 31337:
		return "Local Network"
	case 8453:
		return "Base Mainnet"
	case 84532:
		return "Base Sepolia"
	default:
		return fmt.Sprintf("Chain %d", chainID)
	}
}