package signer

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// ECDSASigner implements Signer using an ECDSA private key
type ECDSASigner struct {
	privateKey *ecdsa.PrivateKey
	address    common.Address
}

// NewECDSASigner creates a new ECDSA signer from a private key
func NewECDSASigner(privateKey *ecdsa.PrivateKey) *ECDSASigner {
	address := crypto.PubkeyToAddress(privateKey.PublicKey)
	return &ECDSASigner{
		privateKey: privateKey,
		address:    address,
	}
}

// NewECDSASignerFromHex creates a new ECDSA signer from a hex-encoded private key
func NewECDSASignerFromHex(hexKey string) (*ECDSASigner, error) {
	// Remove 0x prefix if present
	if len(hexKey) >= 2 && hexKey[0:2] == "0x" {
		hexKey = hexKey[2:]
	}

	privateKey, err := crypto.HexToECDSA(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	return NewECDSASigner(privateKey), nil
}

// Address returns the Ethereum address of the signer
func (s *ECDSASigner) Address() common.Address {
	return s.address
}

// SignTransaction signs a transaction
func (s *ECDSASigner) SignTransaction(tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	signer := types.NewLondonSigner(chainID)
	signedTx, err := types.SignTx(tx, signer, s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}
	return signedTx, nil
}

// SignMessage signs a message using EIP-191
func (s *ECDSASigner) SignMessage(msg []byte) ([]byte, error) {
	// Add Ethereum message prefix
	prefixedMsg := accounts.TextHash(msg)
	
	// Sign the hash
	sig, err := crypto.Sign(prefixedMsg, s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}

	// Transform V from 0/1 to 27/28 according to Ethereum yellow paper
	if sig[64] < 27 {
		sig[64] += 27
	}

	return sig, nil
}

// PublicKey returns the public key
func (s *ECDSASigner) PublicKey() *ecdsa.PublicKey {
	return &s.privateKey.PublicKey
}