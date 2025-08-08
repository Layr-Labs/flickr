package controller

import (
	"context"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourorg/flickr/internal/docker"
	"github.com/yourorg/flickr/internal/eth"
)

// Mock ReleaseManager client
type mockRM struct {
	latest    eth.Release
	latestID  uint64
	rel       eth.Release
	err       error
	latestErr error
}

func (m *mockRM) GetLatestRelease(ctx context.Context, avs common.Address, opSetID uint32) (eth.Release, uint64, error) {
	if m.latestErr != nil {
		return eth.Release{}, 0, m.latestErr
	}
	return m.latest, m.latestID, nil
}

func (m *mockRM) GetRelease(ctx context.Context, avs common.Address, opSetID uint32, releaseID uint64) (eth.Release, error) {
	if m.err != nil {
		return eth.Release{}, m.err
	}
	return m.rel, nil
}

// Mock Docker client
type captureDocker struct {
	pulled  string
	ran     string
	env     map[string]string
	runOpts docker.RunOptions
	pullErr error
	runErr  error
}

func (d *captureDocker) Pull(ctx context.Context, ref string) error {
	d.pulled = ref
	return d.pullErr
}

func (d *captureDocker) Run(ctx context.Context, ref string, opts docker.RunOptions) error {
	d.ran = ref
	d.env = opts.Env
	d.runOpts = opts
	return d.runErr
}

func TestController_Execute_LatestRelease(t *testing.T) {
	// Setup mock release manager
	mockRelease := eth.Release{
		Artifacts: []eth.Artifact{
			{
				Registry: "ghcr.io/org/image",
				Digest32: func() [32]byte {
					var d [32]byte
					for i := range d {
						d[i] = 0xaa
					}
					return d
				}(),
			},
		},
		UpgradeByTime: 123456,
	}
	
	rm := &mockRM{
		latest:   mockRelease,
		latestID: 7,
	}
	
	// Setup mock docker
	dockerMock := &captureDocker{}
	
	// Create controller
	ctrl := New(rm, dockerMock)
	
	// Create config (no release ID = use latest)
	cfg := RunConfig{
		AVS:            common.HexToAddress("0x1234567890123456789012345678901234567890"),
		OperatorSetID:  1,
		ReleaseID:      nil, // Use latest
		ReleaseManager: common.HexToAddress("0xabcdef1234567890abcdef1234567890abcdef12"),
		RPCURL:         "https://eth.example.com",
		Name:           "test-container",
		Detached:       true,
		Env: map[string]string{
			"CUSTOM_VAR": "custom_value",
		},
	}
	
	// Execute
	err := ctrl.Execute(context.Background(), cfg)
	require.NoError(t, err)
	
	// Verify docker pull was called with correct reference
	expectedRef := "ghcr.io/org/image@sha256:" + strings.Repeat("aa", 32)
	assert.Equal(t, expectedRef, dockerMock.pulled)
	
	// Verify docker run was called with same reference
	assert.Equal(t, expectedRef, dockerMock.ran)
	
	// Verify environment variables
	assert.Equal(t, "0x1234567890123456789012345678901234567890", dockerMock.env["AVS_ADDRESS"])
	assert.Equal(t, "1", dockerMock.env["OPERATOR_SET_ID"])
	assert.Equal(t, "7", dockerMock.env["RELEASE_ID"])
	assert.Equal(t, "123456", dockerMock.env["UPGRADE_BY_TIME"])
	assert.Equal(t, "custom_value", dockerMock.env["CUSTOM_VAR"])
	
	// Verify run options
	assert.Equal(t, "test-container", dockerMock.runOpts.Name)
	assert.True(t, dockerMock.runOpts.Detached)
}

func TestController_Execute_SpecificRelease(t *testing.T) {
	// Setup mock release manager
	mockRelease := eth.Release{
		Artifacts: []eth.Artifact{
			{
				Registry: "docker.io/library/busybox",
				Digest32: func() [32]byte {
					var d [32]byte
					for i := range d {
						d[i] = 0xbb
					}
					return d
				}(),
			},
		},
		UpgradeByTime: 654321,
	}
	
	rm := &mockRM{
		rel: mockRelease,
	}
	
	// Setup mock docker
	dockerMock := &captureDocker{}
	
	// Create controller
	ctrl := New(rm, dockerMock)
	
	// Create config with specific release ID
	releaseID := uint64(42)
	cfg := RunConfig{
		AVS:            common.HexToAddress("0x1234567890123456789012345678901234567890"),
		OperatorSetID:  2,
		ReleaseID:      &releaseID,
		ReleaseManager: common.HexToAddress("0xabcdef1234567890abcdef1234567890abcdef12"),
		RPCURL:         "https://eth.example.com",
		Name:           "",
		Detached:       false,
		Env:            map[string]string{},
	}
	
	// Execute
	err := ctrl.Execute(context.Background(), cfg)
	require.NoError(t, err)
	
	// Verify docker pull was called with correct reference
	expectedRef := "docker.io/library/busybox@sha256:" + strings.Repeat("bb", 32)
	assert.Equal(t, expectedRef, dockerMock.pulled)
	
	// Verify docker run was called
	assert.Equal(t, expectedRef, dockerMock.ran)
	
	// Verify release ID in env
	assert.Equal(t, "42", dockerMock.env["RELEASE_ID"])
	assert.Equal(t, "654321", dockerMock.env["UPGRADE_BY_TIME"])
	
	// Verify run options
	assert.Equal(t, "", dockerMock.runOpts.Name)
	assert.False(t, dockerMock.runOpts.Detached)
}

func TestController_Execute_NoArtifacts(t *testing.T) {
	// Setup mock release manager with empty artifacts
	rm := &mockRM{
		latest: eth.Release{
			Artifacts:     []eth.Artifact{}, // No artifacts
			UpgradeByTime: 123,
		},
		latestID: 1,
	}
	
	dockerMock := &captureDocker{}
	ctrl := New(rm, dockerMock)
	
	cfg := RunConfig{
		AVS:            common.HexToAddress("0x1234567890123456789012345678901234567890"),
		OperatorSetID:  1,
		ReleaseManager: common.HexToAddress("0xabcdef1234567890abcdef1234567890abcdef12"),
		RPCURL:         "https://eth.example.com",
	}
	
	// Execute should fail
	err := ctrl.Execute(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no artifacts in release")
}

func TestController_Execute_RegistryWithDigest(t *testing.T) {
	// Test passthrough when registry already contains digest
	mockRelease := eth.Release{
		Artifacts: []eth.Artifact{
			{
				Registry: "ghcr.io/org/image@sha256:" + strings.Repeat("cc", 32),
				Digest32: [32]byte{}, // Should be ignored
			},
		},
		UpgradeByTime: 100,
	}
	
	rm := &mockRM{
		latest:   mockRelease,
		latestID: 5,
	}
	
	dockerMock := &captureDocker{}
	ctrl := New(rm, dockerMock)
	
	cfg := RunConfig{
		AVS:            common.HexToAddress("0x1234567890123456789012345678901234567890"),
		OperatorSetID:  1,
		ReleaseManager: common.HexToAddress("0xabcdef1234567890abcdef1234567890abcdef12"),
		RPCURL:         "https://eth.example.com",
	}
	
	err := ctrl.Execute(context.Background(), cfg)
	require.NoError(t, err)
	
	// Should use registry as-is
	expectedRef := "ghcr.io/org/image@sha256:" + strings.Repeat("cc", 32)
	assert.Equal(t, expectedRef, dockerMock.pulled)
	assert.Equal(t, expectedRef, dockerMock.ran)
}

func TestController_Execute_MultipleArtifacts(t *testing.T) {
	// MVP only uses first artifact
	mockRelease := eth.Release{
		Artifacts: []eth.Artifact{
			{
				Registry: "first.io/image",
				Digest32: func() [32]byte {
					var d [32]byte
					d[0] = 0x11
					return d
				}(),
			},
			{
				Registry: "second.io/image",
				Digest32: func() [32]byte {
					var d [32]byte
					d[0] = 0x22
					return d
				}(),
			},
		},
		UpgradeByTime: 200,
	}
	
	rm := &mockRM{
		latest:   mockRelease,
		latestID: 9,
	}
	
	dockerMock := &captureDocker{}
	ctrl := New(rm, dockerMock)
	
	cfg := RunConfig{
		AVS:            common.HexToAddress("0x1234567890123456789012345678901234567890"),
		OperatorSetID:  1,
		ReleaseManager: common.HexToAddress("0xabcdef1234567890abcdef1234567890abcdef12"),
		RPCURL:         "https://eth.example.com",
	}
	
	err := ctrl.Execute(context.Background(), cfg)
	require.NoError(t, err)
	
	// Should only use first artifact
	assert.Contains(t, dockerMock.pulled, "first.io/image")
	assert.NotContains(t, dockerMock.pulled, "second.io/image")
}