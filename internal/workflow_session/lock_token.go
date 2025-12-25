package workflow_session

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func newLockToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to generate lock token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
