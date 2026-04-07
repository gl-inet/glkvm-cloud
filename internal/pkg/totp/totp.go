// Package totp wraps github.com/pquerna/otp/totp for the cloud server.
//
// We use TOTP (RFC 6238) for two-factor authentication. Secrets are stored
// base32-encoded in the database and verified with ±1 step (30s) skew to
// tolerate clock drift between server and client.
package totp

import (
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// GenerateSecret creates a fresh TOTP secret for the given account.
// Returns the base32 secret and the otpauth:// URL ready for QR encoding.
func GenerateSecret(issuer, accountName string) (secret string, otpauthURL string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return "", "", err
	}
	return key.Secret(), key.URL(), nil
}

// Verify checks a 6-digit code against the secret with ±1 step skew.
func Verify(secret, code string) bool {
	if secret == "" || code == "" {
		return false
	}
	valid, err := totp.ValidateCustom(code, secret, time.Now(), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		return false
	}
	return valid
}
