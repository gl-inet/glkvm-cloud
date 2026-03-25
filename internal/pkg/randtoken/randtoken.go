package randtoken

import (
	"crypto/rand"
	"encoding/base64"
)

// New returns a high-entropy random token (URL-safe).
// 32 bytes => 256-bit.
func New() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
