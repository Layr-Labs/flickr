package ref

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDigest32ToSha256String(t *testing.T) {
	tests := []struct {
		name     string
		input    [32]byte
		expected string
	}{
		{
			name:     "all zeros",
			input:    [32]byte{},
			expected: "sha256:" + strings.Repeat("00", 32),
		},
		{
			name: "all 0xaa",
			input: func() [32]byte {
				var d [32]byte
				for i := range d {
					d[i] = 0xaa
				}
				return d
			}(),
			expected: "sha256:" + strings.Repeat("aa", 32),
		},
		{
			name: "mixed values",
			input: func() [32]byte {
				var d [32]byte
				for i := 0; i < 16; i++ {
					d[i] = 0xff
				}
				for i := 16; i < 32; i++ {
					d[i] = 0x11
				}
				return d
			}(),
			expected: "sha256:" + strings.Repeat("ff", 16) + strings.Repeat("11", 16),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Digest32ToSha256String(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestBuildReference(t *testing.T) {
	validDigest := "sha256:" + strings.Repeat("11", 32)
	
	tests := []struct {
		name        string
		registry    string
		digest      string
		expected    string
		shouldError bool
	}{
		{
			name:     "standard reference",
			registry: "ghcr.io/org/image",
			digest:   validDigest,
			expected: "ghcr.io/org/image@" + validDigest,
		},
		{
			name:     "registry with port",
			registry: "localhost:5000/myimage",
			digest:   validDigest,
			expected: "localhost:5000/myimage@" + validDigest,
		},
		{
			name:     "passthrough - registry already has digest",
			registry: "ghcr.io/org/image@sha256:" + strings.Repeat("22", 32),
			digest:   "ignored",
			expected: "ghcr.io/org/image@sha256:" + strings.Repeat("22", 32),
		},
		{
			name:        "empty registry",
			registry:    "",
			digest:      validDigest,
			shouldError: true,
		},
		{
			name:        "invalid digest format - missing prefix",
			registry:    "ghcr.io/org/image",
			digest:      strings.Repeat("11", 32),
			shouldError: true,
		},
		{
			name:        "invalid digest format - wrong prefix",
			registry:    "ghcr.io/org/image",
			digest:      "md5:" + strings.Repeat("11", 16),
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildReference(tt.registry, tt.digest)
			
			if tt.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestBuildReference_Passthrough(t *testing.T) {
	// Special test for passthrough behavior
	keep := "ghcr.io/org/image@sha256:" + strings.Repeat("22", 32)
	got, err := BuildReference(keep, "this-should-be-ignored")
	require.NoError(t, err)
	assert.Equal(t, keep, got)
}

func TestDigestFormat(t *testing.T) {
	// Test that our digest format matches expected OCI/ORAS format
	var d [32]byte
	for i := range d {
		d[i] = byte(i)
	}
	
	digest := Digest32ToSha256String(d)
	
	// Should start with sha256:
	assert.True(t, strings.HasPrefix(digest, "sha256:"))
	
	// Should be exactly 7 (prefix) + 64 (hex) characters
	assert.Equal(t, 71, len(digest))
	
	// Should be lowercase hex after prefix
	hexPart := strings.TrimPrefix(digest, "sha256:")
	assert.Equal(t, strings.ToLower(hexPart), hexPart)
	
	// Should be valid hex
	for _, c := range hexPart {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'))
	}
}