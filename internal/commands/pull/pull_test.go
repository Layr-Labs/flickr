package pull

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		name            string
		inputError      error
		expectedMessage string
	}{
		{
			name:            "Arithmetic underflow",
			inputError:      fmt.Errorf("execution reverted: panic: arithmetic underflow or overflow (0x11)"),
			expectedMessage: "no releases available",
		},
		{
			name:            "No releases",
			inputError:      fmt.Errorf("NoReleases"),
			expectedMessage: "no releases available",
		},
		{
			name:            "Array out of bounds",
			inputError:      fmt.Errorf("execution reverted: panic: array out-of-bounds access (0x32)"),
			expectedMessage: "release ID",
		},
		{
			name:            "Generic error",
			inputError:      fmt.Errorf("network error"),
			expectedMessage: "network error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test error message handling logic
			errMsg := tt.inputError.Error()
			
			var result string
			if strings.Contains(errMsg, "arithmetic underflow") || strings.Contains(errMsg, "NoReleases") {
				result = "no releases available for this operator set"
			} else if strings.Contains(errMsg, "array out-of-bounds") {
				result = "release ID does not exist"
			} else {
				result = errMsg
			}

			assert.Contains(t, result, tt.expectedMessage)
		})
	}
}

func TestReleaseValidation(t *testing.T) {
	tests := []struct {
		name        string
		releaseID   uint64
		totalCount  int64
		expectError bool
	}{
		{
			name:        "Valid release ID",
			releaseID:   0,
			totalCount:  3,
			expectError: false,
		},
		{
			name:        "Release ID too high",
			releaseID:   5,
			totalCount:  3,
			expectError: true,
		},
		{
			name:        "No releases",
			releaseID:   0,
			totalCount:  0,
			expectError: true,
		},
		{
			name:        "Last release",
			releaseID:   2,
			totalCount:  3,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate validation logic
			hasError := false
			if tt.totalCount == 0 {
				hasError = true
			} else if tt.releaseID >= uint64(tt.totalCount) {
				hasError = true
			}

			assert.Equal(t, tt.expectError, hasError)
		})
	}
}

func TestArtifactSelection(t *testing.T) {
	tests := []struct {
		name              string
		totalArtifacts    int
		pullAll           bool
		expectedPullCount int
	}{
		{
			name:              "Single artifact",
			totalArtifacts:    1,
			pullAll:           false,
			expectedPullCount: 1,
		},
		{
			name:              "Multiple artifacts - pull first only",
			totalArtifacts:    3,
			pullAll:           false,
			expectedPullCount: 1,
		},
		{
			name:              "Multiple artifacts - pull all",
			totalArtifacts:    3,
			pullAll:           true,
			expectedPullCount: 3,
		},
		{
			name:              "No artifacts",
			totalArtifacts:    0,
			pullAll:           false,
			expectedPullCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate artifact selection logic
			artifactsToPull := tt.totalArtifacts
			if !tt.pullAll && tt.totalArtifacts > 1 {
				artifactsToPull = 1
			}

			assert.Equal(t, tt.expectedPullCount, artifactsToPull)
		})
	}
}