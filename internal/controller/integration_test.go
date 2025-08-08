package controller_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourorg/flickr/internal/config"
	"github.com/yourorg/flickr/internal/controller"
	"github.com/yourorg/flickr/internal/docker"
	"github.com/yourorg/flickr/internal/eth"
	"github.com/yourorg/flickr/internal/ref"
	"github.com/yourorg/flickr/internal/signer"
)

// Integration tests for the complete workflow
// These tests require Docker to be running

func TestIntegration_CompleteWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test complete workflow with mock data
	ctx := context.Background()

	// Create test configuration
	testAVS := parseAddress("0x0000000000000000000000000000000000000001")
	testOpSetID := uint32(0)
	testKey := fmt.Sprintf("%s-%d", testAVS.Hex(), testOpSetID)
	
	// Create mock release manager client
	mockRM := &MockReleaseManagerClient{
		releases: map[string][]eth.Release{
			testKey: {
				{
					Artifacts: []eth.Artifact{
						{
							Registry: "alpine",
							Digest32: parseDigest("sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1"),
						},
					},
					UpgradeByTime: 1757281070,
				},
			},
		},
		metadataURIs: map[string]string{
			testKey: "https://example.com/metadata.json",
		},
	}

	// Create real Docker runner
	dockerRunner := docker.New()

	// Create controller
	ctrl := controller.New(mockRM, dockerRunner)

	// Test configuration
	cfg := controller.RunConfig{
		AVS:           testAVS,
		OperatorSetID: testOpSetID,
		ReleaseID:     uint64Ptr(0),
		Name:          "test-flickr-integration",
		Detached:      false,
		Env:           map[string]string{"TEST_ENV": "integration"},
		Cmd:           []string{"echo", "Integration test successful"},
	}

	// Execute the workflow
	err := ctrl.Execute(ctx, cfg)
	assert.NoError(t, err, "Should execute successfully with valid release")
}

func TestIntegration_Signer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("ECDSA Private Key Signer", func(t *testing.T) {
		// Test private key from Anvil
		privateKey := "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
		expectedAddress := "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"

		sig, err := signer.NewECDSASignerFromHex(privateKey)
		require.NoError(t, err)

		assert.Equal(t, expectedAddress, sig.Address().Hex())
	})

	t.Run("Context with Signer", func(t *testing.T) {
		ctx := &config.Context{
			ECDSAPrivateKey: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
		}

		sig, err := signer.FromContext(ctx)
		require.NoError(t, err)

		assert.Equal(t, "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", sig.Address().Hex())
	})
}

func TestIntegration_MetadataCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test that metadata URI is required before pushing
	mockRM := &MockReleaseManagerClient{
		metadataURIs: map[string]string{}, // No metadata set
	}

	// This should fail when trying to push without metadata
	err := mockRM.CheckMetadataRequired(context.Background(), 
		parseAddress("0x0000000000000000000000000000000000000001"), 0)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "metadata URI required")
}

func TestIntegration_DigestConversion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Alpine digest",
			input:    "4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1",
			expected: "sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1",
		},
		{
			name:     "Zero digest",
			input:    strings.Repeat("00", 32),
			expected: "sha256:" + strings.Repeat("00", 32),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			digest32 := parseDigest("sha256:" + tt.input)
			result := ref.Digest32ToSha256String(digest32)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIntegration_ReferenceBuilding(t *testing.T) {
	tests := []struct {
		name     string
		registry string
		digest   string
		expected string
	}{
		{
			name:     "Alpine registry",
			registry: "alpine",
			digest:   "sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1",
			expected: "alpine@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1",
		},
		{
			name:     "Docker Hub",
			registry: "docker.io/library",
			digest:   "sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1",
			expected: "docker.io/library@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1",
		},
		{
			name:     "Private registry",
			registry: "myregistry.io/myorg/myimage",
			digest:   "sha256:abcd1234",
			expected: "myregistry.io/myorg/myimage@sha256:abcd1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ref.BuildReference(tt.registry, tt.digest)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Mock implementation for testing
type MockReleaseManagerClient struct {
	releases     map[string][]eth.Release
	metadataURIs map[string]string
}

func (m *MockReleaseManagerClient) GetLatestRelease(ctx context.Context, avs common.Address, opSetID uint32) (eth.Release, uint64, error) {
	key := fmt.Sprintf("%s-%d", avs.Hex(), opSetID)
	releases, ok := m.releases[key]
	if !ok || len(releases) == 0 {
		return eth.Release{}, 0, fmt.Errorf("no releases found")
	}
	return releases[len(releases)-1], uint64(len(releases) - 1), nil
}

func (m *MockReleaseManagerClient) GetRelease(ctx context.Context, avs common.Address, opSetID uint32, releaseID uint64) (eth.Release, error) {
	key := fmt.Sprintf("%s-%d", avs.Hex(), opSetID)
	releases, ok := m.releases[key]
	if !ok || int(releaseID) >= len(releases) {
		return eth.Release{}, fmt.Errorf("release not found")
	}
	return releases[releaseID], nil
}

func (m *MockReleaseManagerClient) CheckMetadataRequired(ctx context.Context, avs common.Address, opSetID uint32) error {
	key := fmt.Sprintf("%s-%d", avs.Hex(), opSetID)
	if _, ok := m.metadataURIs[key]; !ok {
		return fmt.Errorf("metadata URI required")
	}
	return nil
}

// Helper functions
func parseAddress(addr string) common.Address {
	return common.HexToAddress(addr)
}

func parseDigest(digestStr string) [32]byte {
	// Remove sha256: prefix if present
	digestStr = strings.TrimPrefix(digestStr, "sha256:")
	
	var digest [32]byte
	for i := 0; i < 32 && i*2 < len(digestStr); i++ {
		fmt.Sscanf(digestStr[i*2:i*2+2], "%02x", &digest[i])
	}
	return digest
}

func uint64Ptr(v uint64) *uint64 {
	return &v
}