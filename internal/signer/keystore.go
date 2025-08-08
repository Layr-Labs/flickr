package signer

import (
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// KeystoreSigner implements Signer using a keystore file
type KeystoreSigner struct {
	privateKey *ecdsa.PrivateKey
	address    common.Address
}

// NewKeystoreSigner creates a new signer from a keystore file
func NewKeystoreSigner(keystorePath, password string) (*KeystoreSigner, error) {
	// Read the keystore file
	keyjson, err := ioutil.ReadFile(keystorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read keystore file: %w", err)
	}

	// Decrypt the key
	key, err := keystore.DecryptKey(keyjson, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt keystore: %w", err)
	}

	return &KeystoreSigner{
		privateKey: key.PrivateKey,
		address:    key.Address,
	}, nil
}

// Address returns the Ethereum address of the signer
func (s *KeystoreSigner) Address() common.Address {
	return s.address
}

// SignTransaction signs a transaction
func (s *KeystoreSigner) SignTransaction(tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	signer := types.NewLondonSigner(chainID)
	signedTx, err := types.SignTx(tx, signer, s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}
	return signedTx, nil
}

// SignMessage signs a message using EIP-191
func (s *KeystoreSigner) SignMessage(msg []byte) ([]byte, error) {
	// Reuse ECDSASigner's implementation
	ecdsaSigner := &ECDSASigner{
		privateKey: s.privateKey,
		address:    s.address,
	}
	return ecdsaSigner.SignMessage(msg)
}

// PublicKey returns the public key
func (s *KeystoreSigner) PublicKey() *ecdsa.PublicKey {
	return &s.privateKey.PublicKey
}