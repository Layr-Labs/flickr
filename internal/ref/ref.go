package ref

import (
	"encoding/hex"
	"fmt"
	"strings"
)

func Digest32ToSha256String(d [32]byte) string {
	return "sha256:" + strings.ToLower(hex.EncodeToString(d[:]))
}

func BuildReference(registry string, digest string) (string, error) {
	if registry == "" {
		return "", fmt.Errorf("empty registry")
	}
	if strings.Contains(registry, "@sha256:") {
		return registry, nil
	}
	if !strings.HasPrefix(digest, "sha256:") {
		return "", fmt.Errorf("invalid digest format")
	}
	return registry + "@" + digest, nil
}