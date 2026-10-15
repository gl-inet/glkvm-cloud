package pki

import (
	"crypto/sha256"
	"crypto/x509"
	_ "embed"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
)

// glRootPEM is the GL.iNet Root CA. It signs the factory certificate every GL
// KVM ships with, so the mTLS device port trusts it to recognize genuine devices.
//
//go:embed gl-root-ca.pem
var glRootPEM []byte

// GLRootFingerprint is the SHA-256 of the embedded root (DER). Pinned so a
// swapped file is rejected instead of silently trusting another CA.
const GLRootFingerprint = "83dbc6f561b7a786a230316f4fe2b258b4f71adb3b8d78cf6b0cabbae167d9b2"

// GLRootPool returns a pool holding only the GL.iNet Root CA.
func GLRootPool() (*x509.CertPool, error) {
	block, _ := pem.Decode(glRootPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("pki: embedded GL root CA is not a PEM certificate")
	}
	if fp := Fingerprint(block.Bytes); fp != GLRootFingerprint {
		return nil, fmt.Errorf("pki: embedded GL root CA fingerprint %s does not match the pinned value", fp)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("pki: parse GL root CA: %w", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return pool, nil
}

// Fingerprint returns the lowercase hex SHA-256 of a DER certificate.
func Fingerprint(der []byte) string {
	sum := sha256.Sum256(der)
	return hex.EncodeToString(sum[:])
}

// DeviceMAC checks that a client certificate, already verified to chain to the
// GL root, is a factory device certificate and returns its MAC (the CN,
// colon-less, lowercase).
//
// Chaining to the GL root is not enough on its own: the root also signs
// certificates that are not device certificates. A factory certificate always
// names client authentication explicitly and uses the MAC as its CN.
func DeviceMAC(cert *x509.Certificate) (string, error) {
	clientAuth := false
	for _, u := range cert.ExtKeyUsage {
		if u == x509.ExtKeyUsageClientAuth {
			clientAuth = true
			break
		}
	}
	if !clientAuth {
		return "", errors.New("certificate is not issued for client authentication")
	}
	mac := strings.ToLower(cert.Subject.CommonName)
	if !isMAC(mac) {
		return "", fmt.Errorf("certificate CN %q is not a MAC address", cert.Subject.CommonName)
	}
	return mac, nil
}

// isMAC reports whether s is a colon-less 12-digit hex MAC.
func isMAC(s string) bool {
	if len(s) != 12 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
