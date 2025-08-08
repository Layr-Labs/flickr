package controller

import (
	"context"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourorg/flickr/internal/docker"
	"github.com/yourorg/flickr/internal/eth"
)

// TestRealDocker_AlpineWithDigest tests with a real Alpine image using digest conversion
func TestRealDocker_AlpineWithDigest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker integration test in short mode")
	}

	// Check Docker availability
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not available")
	}

	// Alpine digest (first 32 bytes of the sha256)
	// Full digest: 4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1
	alpineDigestHex := "4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1"
	
	// Convert to [32]byte (taking first 32 bytes)
	digestBytes, err := hex.DecodeString(alpineDigestHex[:64]) // First 64 hex chars = 32 bytes
	require.NoError(t, err)
	
	var digest32 [32]byte
	copy(digest32[:], digestBytes)

	// Container name for cleanup
	containerName := fmt.Sprintf("flickr-alpine-test-%d", time.Now().UnixNano())
	
	// Cleanup
	cleanup := func() {
		exec.Command("docker", "rm", "-f", containerName).Run()
	}
	defer cleanup()
	cleanup() // Pre-cleanup

	// Mock release with alpine digest
	mockRelease := eth.Release{
		Artifacts: []eth.Artifact{
			{
				Registry: "docker.io/library/alpine",
				Digest32: digest32,
			},
		},
		UpgradeByTime: 1234567890,
	}
	
	rm := &mockRM{
		latest:   mockRelease,
		latestID: 100,
	}
	
	// Custom Docker runner that adds a sleep command to alpine
	type alpineRunner struct {
		*docker.Runner
	}
	
	runner := &alpineRunner{docker.New()}
	
	// Wrapper to add sleep command for alpine
	dockerWithSleep := &dockerWithSleepWrapper{
		inner: runner,
	}
	
	ctrl := New(rm, dockerWithSleep)
	
	cfg := RunConfig{
		AVS:            common.HexToAddress("0x1234567890123456789012345678901234567890"),
		OperatorSetID:  1,
		ReleaseManager: common.HexToAddress("0xABCDEF1234567890ABCDEF1234567890ABCDEF12"),
		RPCURL:         "https://eth.example.com",
		Name:           containerName,
		Detached:       true,
		Env: map[string]string{
			"TEST_ENV": "test_value",
		},
	}
	
	// Execute
	ctx := context.Background()
	err = ctrl.Execute(ctx, cfg)
	require.NoError(t, err, "Execution should succeed")
	
	// Wait for container to start
	time.Sleep(2 * time.Second)
	
	// Verify container is running
	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", containerName), "--format", "{{.Names}}")
	output, err := cmd.Output()
	require.NoError(t, err)
	assert.Contains(t, string(output), containerName, "Container should be running")
	
	// Verify environment variables
	cmd = exec.Command("docker", "inspect", containerName, "--format", "{{range .Config.Env}}{{println .}}{{end}}")
	envOutput, err := cmd.Output()
	require.NoError(t, err)
	
	envStr := string(envOutput)
	assert.Contains(t, envStr, "AVS_ADDRESS=0x1234567890123456789012345678901234567890")
	assert.Contains(t, envStr, "OPERATOR_SET_ID=1")
	assert.Contains(t, envStr, "RELEASE_ID=100")
	assert.Contains(t, envStr, "TEST_ENV=test_value")
	
	// Stop and verify
	exec.Command("docker", "stop", containerName).Run()
}

// dockerWithSleepWrapper adds sleep command to keep containers running for testing
type dockerWithSleepWrapper struct {
	inner docker.Docker
}

func (d *dockerWithSleepWrapper) Pull(ctx context.Context, ref string) error {
	return d.inner.Pull(ctx, ref)
}

func (d *dockerWithSleepWrapper) Run(ctx context.Context, ref string, opts docker.RunOptions) error {
	// Add sleep command for alpine/busybox to keep them running
	if strings.Contains(ref, "alpine") || strings.Contains(ref, "busybox") {
		opts.Cmd = []string{"sleep", "3600"} // Sleep for 1 hour
	}
	return d.inner.Run(ctx, ref, opts)
}

// TestRealDocker_HelloWorld tests with hello-world which exits immediately  
func TestRealDocker_HelloWorld(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker integration test in short mode")
	}

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not available")
	}

	// Use hello-world with digest directly in registry (passthrough mode)
	helloWorldWithDigest := "docker.io/library/hello-world@sha256:d211f485f2dd1dee407a80973c8f129f00d54604d2c90732e8e320e5038a0348"
	
	mockRelease := eth.Release{
		Artifacts: []eth.Artifact{
			{
				Registry: helloWorldWithDigest,
				Digest32: [32]byte{}, // Ignored in passthrough mode
			},
		},
		UpgradeByTime: 987654321,
	}
	
	rm := &mockRM{
		latest:   mockRelease,
		latestID: 200,
	}
	
	dockerRunner := docker.New()
	ctrl := New(rm, dockerRunner)
	
	cfg := RunConfig{
		AVS:            common.HexToAddress("0xAAAABBBBCCCCDDDDEEEEFFFF111122223333444"),
		OperatorSetID:  5,
		ReleaseManager: common.HexToAddress("0x5555666677778888999900001111222233334444"),
		RPCURL:         "https://eth.example.com",
		Name:           "", // Let Docker assign
		Detached:       false, // hello-world exits immediately
		Env: map[string]string{
			"HELLO_ENV": "world",
		},
	}
	
	ctx := context.Background()
	err := ctrl.Execute(ctx, cfg)
	
	// hello-world runs and exits with 0, which is success
	require.NoError(t, err, "hello-world should execute successfully")
}

// TestRealDocker_CleanupMultiple tests cleanup of multiple containers
func TestRealDocker_CleanupMultiple(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker integration test in short mode")  
	}

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not available")
	}

	// Create multiple containers
	containerNames := []string{
		fmt.Sprintf("flickr-cleanup-1-%d", time.Now().UnixNano()),
		fmt.Sprintf("flickr-cleanup-2-%d", time.Now().UnixNano()),
		fmt.Sprintf("flickr-cleanup-3-%d", time.Now().UnixNano()),
	}
	
	// Cleanup all containers
	cleanup := func() {
		for _, name := range containerNames {
			exec.Command("docker", "rm", "-f", name).Run()
		}
	}
	defer cleanup()
	cleanup() // Pre-cleanup
	
	// Use alpine with passthrough digest
	alpineWithDigest := "docker.io/library/alpine@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1"
	
	mockRelease := eth.Release{
		Artifacts: []eth.Artifact{
			{
				Registry: alpineWithDigest,
				Digest32: [32]byte{},
			},
		},
		UpgradeByTime: 555555555,
	}
	
	rm := &mockRM{
		latest:   mockRelease,
		latestID: 300,
	}
	
	// Use wrapper to add sleep
	dockerRunner := &dockerWithSleepWrapper{inner: docker.New()}
	ctrl := New(rm, dockerRunner)
	
	// Start multiple containers
	for i, containerName := range containerNames {
		cfg := RunConfig{
			AVS:            common.HexToAddress(fmt.Sprintf("0x%040d", i)),
			OperatorSetID:  uint32(i),
			ReleaseManager: common.HexToAddress("0x1111222233334444555566667777888899990000"),
			RPCURL:         "https://eth.example.com",
			Name:           containerName,
			Detached:       true,
			Env:            map[string]string{},
		}
		
		ctx := context.Background()
		err := ctrl.Execute(ctx, cfg)
		require.NoError(t, err, "Container %d should start", i)
	}
	
	// Wait for containers to start
	time.Sleep(2 * time.Second)
	
	// Verify all are running
	for _, name := range containerNames {
		cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", name), "--format", "{{.Names}}")
		output, err := cmd.Output()
		require.NoError(t, err)
		assert.Contains(t, string(output), name, "Container %s should be running", name)
	}
	
	// Stop all containers
	for _, name := range containerNames {
		exec.Command("docker", "stop", name).Run()
	}
	
	// Verify all containers are removed (--rm flag automatically removes them)
	for _, name := range containerNames {
		cmd := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", name), "--format", "{{.Names}}")
		output, err := cmd.Output()
		require.NoError(t, err)
		// With --rm flag, containers should be removed after stopping
		assert.Empty(t, strings.TrimSpace(string(output)), "Container %s should be removed", name)
	}
}