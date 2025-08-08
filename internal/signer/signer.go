package signer

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Signer interface for signing transactions and messages
type Signer interface {
	// Address returns the Ethereum address of the signer
	Address() common.Address

	// SignTransaction signs a transaction
	SignTransaction(tx *types.Transaction, chainID *big.Int) (*types.Transaction, error)

	// SignMessage signs a message using EIP-191
	SignMessage(msg []byte) ([]byte, error)

	// PublicKey returns the public key
	PublicKey() *ecdsa.PublicKey
}