// Package access models how devices join a self-hosted platform over mTLS:
// one-time registration codes, and the whitelist of devices that redeemed one.
package access

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

var (
	// ErrCodeInvalid: the registration code is unknown, used, revoked or expired.
	ErrCodeInvalid = errors.New("registration code is invalid, used, revoked or expired")
	// ErrDevidTaken: the device ID already belongs to another device.
	ErrDevidTaken = errors.New("device ID belongs to another device")
)

// WhitelistEntry is a device allowed on the mTLS port.
type WhitelistEntry struct {
	MAC       string // colon-less, lowercase; the factory certificate CN
	Devid     string
	CodeID    int64 // registration code it joined with (0 = none)
	CreatedAt int64
}

// HashCode returns the stored form of a registration code.
func HashCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}
