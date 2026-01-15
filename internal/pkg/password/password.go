package password

import (
	"crypto/sha256"
	"encoding/hex"
)

// Demo only; production should use bcrypt/argon2.
func HashDemoSHA256(pw string) string {
	h := sha256.Sum256([]byte(pw))
	return hex.EncodeToString(h[:])
}

func VerifyDemoSHA256(pw, hash string) bool {
	return HashDemoSHA256(pw) == hash
}
