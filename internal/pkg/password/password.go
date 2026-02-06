package password

import "golang.org/x/crypto/bcrypt"

const bcryptCost = 12

// HashPassword hashes a plaintext password using bcrypt.
func HashPassword(pw string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword checks a plaintext password against a stored bcrypt hash.
func VerifyPassword(pw, hash string) bool {
	if hash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}
