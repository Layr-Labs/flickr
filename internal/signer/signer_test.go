package signer_test

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourorg/flickr/internal/config"
	"github.com/yourorg/flickr/internal/signer"
)

func TestECDSASigner(t *testing.T) {
	// Test private key from Anvil
	privateKeyHex := "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	expectedAddress := "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"

	t.Run("Create from hex", func(t *testing.T) {
		sig, err := signer.NewECDSASignerFromHex(privateKeyHex)
		require.NoError(t, err)
		assert.Equal(t, expectedAddress, sig.Address().Hex())
	})

	t.Run("Create from hex without 0x prefix", func(t *testing.T) {
		sig, err := signer.NewECDSASignerFromHex(privateKeyHex[2:])
		require.NoError(t, err)
		assert.Equal(t, expectedAddress, sig.Address().Hex())
	})

	t.Run("Invalid hex", func(t *testing.T) {
		_, err := signer.NewECDSASignerFromHex("invalid")
		assert.Error(t, err)
	})

	t.Run("Sign transaction", func(t *testing.T) {
		sig, err := signer.NewECDSASignerFromHex(privateKeyHex)
		require.NoError(t, err)

		// Create a test transaction
		tx := types.NewTransaction(
			0,                                      // nonce
			common.HexToAddress("0x0000000000000000000000000000000000000000"), // to
			big.NewInt(1000),                       // value
			21000,                                  // gas limit
			big.NewInt(20000000000),                // gas price
			nil,                                    // data
		)

		chainID := big.NewInt(1)
		signedTx, err := sig.SignTransaction(tx, chainID)
		require.NoError(t, err)
		assert.NotNil(t, signedTx)
	})

	t.Run("Sign message", func(t *testing.T) {
		sig, err := signer.NewECDSASignerFromHex(privateKeyHex)
		require.NoError(t, err)

		message := []byte("Hello, Flickr!")
		signature, err := sig.SignMessage(message)
		require.NoError(t, err)
		assert.Len(t, signature, 65) // r(32) + s(32) + v(1)
	})

	t.Run("Public key", func(t *testing.T) {
		sig, err := signer.NewECDSASignerFromHex(privateKeyHex)
		require.NoError(t, err)

		pubKey := sig.PublicKey()
		assert.NotNil(t, pubKey)
		assert.IsType(t, &ecdsa.PublicKey{}, pubKey)
	})
}

func TestFromContext(t *testing.T) {
	tests := []struct {
		name        string
		context     *config.Context
		expectError bool
		expectedAddr string
	}{
		{
			name: "With ECDSA private key",
			context: &config.Context{
				ECDSAPrivateKey: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
			},
			expectError:  false,
			expectedAddr: "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
		},
		{
			name: "No signer configured",
			context: &config.Context{},
			expectError: true,
		},
		{
			name: "Invalid private key",
			context: &config.Context{
				ECDSAPrivateKey: "invalid",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig, err := signer.FromContext(tt.context)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, sig)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, sig)
				assert.Equal(t, tt.expectedAddr, sig.Address().Hex())
			}
		})
	}
}

func TestSignerMutualExclusivity(t *testing.T) {
	// Test that setting one signer type should clear the other
	ctx := &config.Context{}

	// Set ECDSA private key
	ctx.ECDSAPrivateKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	ctx.KeystorePath = ""
	ctx.KeystorePassword = ""

	sig, err := signer.FromContext(ctx)
	require.NoError(t, err)
	assert.Equal(t, "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", sig.Address().Hex())

	// Now set keystore (simulating context set command logic)
	ctx.KeystorePath = "/path/to/keystore"
	ctx.KeystorePassword = "password"
	ctx.ECDSAPrivateKey = "" // Should be cleared

	// This will fail because the keystore doesn't exist, but it shows the logic
	_, err = signer.FromContext(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "keystore")
}