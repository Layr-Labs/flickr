package signer

import (
	"fmt"

	"github.com/yourorg/flickr/internal/config"
)

// FromContext creates a signer from the context configuration
func FromContext(ctx *config.Context) (Signer, error) {
	// Check for ECDSA private key
	if ctx.ECDSAPrivateKey != "" {
		return NewECDSASignerFromHex(ctx.ECDSAPrivateKey)
	}

	// Check for keystore
	if ctx.KeystorePath != "" {
		if ctx.KeystorePassword == "" {
			return nil, fmt.Errorf("keystore password is required")
		}
		return NewKeystoreSigner(ctx.KeystorePath, ctx.KeystorePassword)
	}

	return nil, fmt.Errorf("no signer configured in context")
}