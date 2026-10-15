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
	// ErrCodeNotFound: no registration code with that ID.
	ErrCodeNotFound = errors.New("registration code not found")
	// ErrCodeNotRevocable: the code was already used, revoked or has expired.
	ErrCodeNotRevocable = errors.New("registration code is already used, revoked or expired")
)

// CodeStatus is the state of a registration code.
type CodeStatus string

const (
	CodeUnused  CodeStatus = "unused"
	CodeUsed    CodeStatus = "used"
	CodeExpired CodeStatus = "expired"
	CodeRevoked CodeStatus = "revoked"
)

// RegistrationCode is a stored registration code. The code itself is never
// stored, only its hash.
type RegistrationCode struct {
	ID        int64
	CreatedBy int64
	CreatedAt int64
	ExpiresAt int64
	UsedAt    int64  // 0 = not used
	UsedMAC   string // device that redeemed it
	Devid     string // ID that device joined under
	RevokedAt int64  // 0 = not revoked
}

// Status reports the code's state at time now.
func (c RegistrationCode) Status(now int64) CodeStatus {
	switch {
	case c.RevokedAt != 0:
		return CodeRevoked
	case c.UsedAt != 0:
		return CodeUsed
	case c.ExpiresAt <= now:
		return CodeExpired
	default:
		return CodeUnused
	}
}

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
