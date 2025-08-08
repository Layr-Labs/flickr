package eth

import (
	"context"
	"fmt"
	"math/big"

	"github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/ReleaseManager"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/yourorg/flickr/internal/signer"
)

// Artifact represents a release artifact with registry and digest
type Artifact struct {
	Registry string
	Digest32 [32]byte
}

// Release represents a release with artifacts and upgrade deadline
type Release struct {
	Artifacts     []Artifact
	UpgradeByTime uint32
}

// ReleaseManagerClient interface for interacting with the ReleaseManager contract
type ReleaseManagerClient interface {
	GetLatestRelease(ctx context.Context, avs common.Address, opSetID uint32) (Release, uint64, error)
	GetRelease(ctx context.Context, avs common.Address, opSetID uint32, releaseID uint64) (Release, error)
}

// Client implements ReleaseManagerClient using the actual contract bindings
type Client struct {
	ethClient    *ethclient.Client
	rmContract   *ReleaseManager.ReleaseManager
	contractAddr common.Address
	rpcURL       string
	signer       signer.Signer // Optional signer for transactions
}

// NewClient creates a new ReleaseManager client
func NewClient(rpcURL string, contractAddr common.Address) (*Client, error) {
	ethClient, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum client: %w", err)
	}

	// Create contract instance
	rmContract, err := ReleaseManager.NewReleaseManager(contractAddr, ethClient)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate ReleaseManager contract: %w", err)
	}

	return &Client{
		ethClient:    ethClient,
		rmContract:   rmContract,
		contractAddr: contractAddr,
		rpcURL:       rpcURL,
	}, nil
}

// GetLatestRelease fetches the latest release for an AVS and operator set
func (c *Client) GetLatestRelease(ctx context.Context, avs common.Address, opSetID uint32) (Release, uint64, error) {
	opts := &bind.CallOpts{Context: ctx}

	// Create OperatorSet struct
	operatorSet := ReleaseManager.OperatorSet{
		Avs: avs,
		Id:  opSetID,
	}

	// Call the contract method to get latest release
	releaseID, contractRelease, err := c.rmContract.GetLatestRelease(opts, operatorSet)
	if err != nil {
		return Release{}, 0, fmt.Errorf("failed to get latest release: %w", err)
	}

	// Convert contract release to our internal format
	release := convertRelease(contractRelease)

	return release, releaseID.Uint64(), nil
}

// GetRelease fetches a specific release for an AVS and operator set
func (c *Client) GetRelease(ctx context.Context, avs common.Address, opSetID uint32, releaseID uint64) (Release, error) {
	opts := &bind.CallOpts{Context: ctx}

	// Create OperatorSet struct
	operatorSet := ReleaseManager.OperatorSet{
		Avs: avs,
		Id:  opSetID,
	}

	// Convert releaseID to big.Int
	releaseIDBig := new(big.Int).SetUint64(releaseID)

	// Call the contract to get the release
	contractRelease, err := c.rmContract.GetRelease(opts, operatorSet, releaseIDBig)
	if err != nil {
		return Release{}, fmt.Errorf("failed to get release from contract: %w", err)
	}

	// Convert contract release to our internal format
	release := convertRelease(contractRelease)

	return release, nil
}

// GetMetadataURI gets the metadata URI for an operator set
func (c *Client) GetMetadataURI(ctx context.Context, avs common.Address, opSetID uint32) (string, error) {
	opts := &bind.CallOpts{Context: ctx}

	// Create OperatorSet struct
	operatorSet := ReleaseManager.OperatorSet{
		Avs: avs,
		Id:  opSetID,
	}

	uri, err := c.rmContract.GetMetadataURI(opts, operatorSet)
	if err != nil {
		return "", fmt.Errorf("failed to get metadata URI: %w", err)
	}

	return uri, nil
}

// GetTotalReleases gets the total number of releases for an AVS and operator set
func (c *Client) GetTotalReleases(ctx context.Context, avs common.Address, opSetID uint32) (*big.Int, error) {
	opts := &bind.CallOpts{Context: ctx}

	// Create OperatorSet struct
	operatorSet := ReleaseManager.OperatorSet{
		Avs: avs,
		Id:  opSetID,
	}

	count, err := c.rmContract.GetTotalReleases(opts, operatorSet)
	if err != nil {
		return nil, fmt.Errorf("failed to get total releases: %w", err)
	}

	return count, nil
}

// GetLatestUpgradeByTime gets the upgrade deadline for the latest release
func (c *Client) GetLatestUpgradeByTime(ctx context.Context, avs common.Address, opSetID uint32) (uint32, error) {
	opts := &bind.CallOpts{Context: ctx}

	// Create OperatorSet struct
	operatorSet := ReleaseManager.OperatorSet{
		Avs: avs,
		Id:  opSetID,
	}

	upgradeByTime, err := c.rmContract.GetLatestUpgradeByTime(opts, operatorSet)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest upgrade by time: %w", err)
	}

	return upgradeByTime, nil
}

// NewClientWithSigner creates a new ReleaseManager client with a signer for transactions
func NewClientWithSigner(rpcURL string, contractAddr common.Address, sig signer.Signer) (*Client, error) {
	client, err := NewClient(rpcURL, contractAddr)
	if err != nil {
		return nil, err
	}
	client.signer = sig
	return client, nil
}

// PublishMetadataURI publishes a metadata URI for an operator set
func (c *Client) PublishMetadataURI(ctx context.Context, avs common.Address, opSetID uint32, uri string, gasLimit uint64) (*types.Transaction, error) {
	if c.signer == nil {
		return nil, fmt.Errorf("signer required for publishing metadata URI")
	}

	// Get chain ID
	chainID, err := c.ethClient.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	// Get nonce
	nonce, err := c.ethClient.PendingNonceAt(ctx, c.signer.Address())
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get gas price
	gasPrice, err := c.ethClient.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	// Create transaction options
	opts := &bind.TransactOpts{
		From:     c.signer.Address(),
		Nonce:    big.NewInt(int64(nonce)),
		GasLimit: gasLimit,
		GasPrice: gasPrice,
		Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			if address != c.signer.Address() {
				return nil, fmt.Errorf("unexpected signer address")
			}
			return c.signer.SignTransaction(tx, chainID)
		},
		Context: ctx,
	}

	// Create OperatorSet struct
	operatorSet := ReleaseManager.OperatorSet{
		Avs: avs,
		Id:  opSetID,
	}

	// Call the contract to publish metadata URI
	tx, err := c.rmContract.PublishMetadataURI(opts, operatorSet, uri)
	if err != nil {
		return nil, fmt.Errorf("failed to publish metadata URI: %w", err)
	}

	return tx, nil
}

// PushRelease pushes a new release on-chain
func (c *Client) PushRelease(ctx context.Context, avs common.Address, opSetID uint32, artifacts []Artifact, upgradeByTime uint32, gasLimit uint64) (*types.Transaction, error) {
	if c.signer == nil {
		return nil, fmt.Errorf("signer required for pushing releases")
	}

	// Get chain ID
	chainID, err := c.ethClient.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	// Get nonce
	nonce, err := c.ethClient.PendingNonceAt(ctx, c.signer.Address())
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get gas price
	gasPrice, err := c.ethClient.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	// Create transaction options
	opts := &bind.TransactOpts{
		From:     c.signer.Address(),
		Nonce:    big.NewInt(int64(nonce)),
		GasLimit: gasLimit,
		GasPrice: gasPrice,
		Signer: func(address common.Address, tx *types.Transaction) (*types.Transaction, error) {
			if address != c.signer.Address() {
				return nil, fmt.Errorf("unexpected signer address")
			}
			return c.signer.SignTransaction(tx, chainID)
		},
		Context: ctx,
	}

	// Create OperatorSet struct
	operatorSet := ReleaseManager.OperatorSet{
		Avs: avs,
		Id:  opSetID,
	}

	// Convert artifacts to contract format
	contractArtifacts := make([]ReleaseManager.IReleaseManagerTypesArtifact, len(artifacts))
	for i, artifact := range artifacts {
		contractArtifacts[i] = ReleaseManager.IReleaseManagerTypesArtifact{
			Registry: artifact.Registry,
			Digest:   artifact.Digest32,
		}
	}

	// Create release
	release := ReleaseManager.IReleaseManagerTypesRelease{
		Artifacts:     contractArtifacts,
		UpgradeByTime: upgradeByTime,
	}

	// Call the contract to publish the release
	tx, err := c.rmContract.PublishRelease(opts, operatorSet, release)
	if err != nil {
		return nil, fmt.Errorf("failed to publish release: %w", err)
	}

	return tx, nil
}

// Close closes the Ethereum client connection
func (c *Client) Close() {
	if c.ethClient != nil {
		c.ethClient.Close()
	}
}

// Helper function to convert contract release to our internal format
func convertRelease(contractRelease ReleaseManager.IReleaseManagerTypesRelease) Release {
	release := Release{
		UpgradeByTime: contractRelease.UpgradeByTime,
		Artifacts:     make([]Artifact, len(contractRelease.Artifacts)),
	}

	for i, artifact := range contractRelease.Artifacts {
		release.Artifacts[i] = Artifact{
			Registry: artifact.Registry,
			Digest32: artifact.Digest,
		}
	}

	return release
}