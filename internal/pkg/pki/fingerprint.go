package pki

import (
	"encoding/pem"
	"errors"
	"strings"
)

// DisplayFingerprint is the SHA-256 fingerprint of the first certificate in
// certPEM as colon-separated uppercase hex, the way
// `openssl x509 -noout -fingerprint -sha256` prints it, so an admin can
// compare the two by eye.
func DisplayFingerprint(certPEM []byte) (string, error) {
	blk, _ := pem.Decode(certPEM)
	if blk == nil || blk.Type != "CERTIFICATE" {
		return "", errors.New("pki: no certificate in PEM data")
	}
	h := strings.ToUpper(Fingerprint(blk.Bytes))
	parts := make([]string, 0, len(h)/2)
	for i := 0; i+1 < len(h); i += 2 {
		parts = append(parts, h[i:i+2])
	}
	return strings.Join(parts, ":"), nil
}
