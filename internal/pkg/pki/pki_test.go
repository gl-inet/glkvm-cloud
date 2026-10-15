package pki

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGLRootPool(t *testing.T) {
	if _, err := GLRootPool(); err != nil {
		t.Fatalf("GLRootPool: %v", err)
	}

	// A different certificate in place of the embedded root must be refused.
	orig := glRootPEM
	defer func() { glRootPEM = orig }()
	d, err := LoadOrCreate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	glRootPEM = d.CAPEM()
	if _, err := GLRootPool(); err == nil {
		t.Fatal("GLRootPool accepted a certificate that does not match the pinned fingerprint")
	}
}

// issue signs a certificate for tests with the given template tweaks.
func issue(t *testing.T, d *Deployment, cn string, eku []x509.ExtKeyUsage, notAfter time.Time) *x509.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: randomSerial(),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     notAfter,
		ExtKeyUsage:  eku,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, d.ca, key.Public(), d.caKey)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return cert
}

func TestDeviceMAC(t *testing.T) {
	d, err := LoadOrCreate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Hour)
	client := []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}

	cases := []struct {
		name    string
		cert    *x509.Certificate
		wantMAC string // "" = must be rejected
	}{
		{"factory certificate", issue(t, d, "9483c4b71fb1", client, later), "9483c4b71fb1"},
		{"upper-case CN", issue(t, d, "9483C4B71FB1", client, later), "9483c4b71fb1"},
		{"no extended key usage", issue(t, d, "9483c4b71fb1", nil, later), ""},
		{"server auth only", issue(t, d, "9483c4b71fb1", []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, later), ""},
		{"CN is a host name", issue(t, d, "kvm-server", client, later), ""},
		{"CN with colons", issue(t, d, "94:83:c4:b7:1f:b1", client, later), ""},
		{"CN too short", issue(t, d, "9483c4b71f", client, later), ""},
	}
	for _, tc := range cases {
		mac, err := DeviceMAC(tc.cert)
		if tc.wantMAC == "" {
			if err == nil {
				t.Errorf("%s: accepted with MAC %q, want rejection", tc.name, mac)
			}
			continue
		}
		if err != nil || mac != tc.wantMAC {
			t.Errorf("%s: got (%q, %v), want %q", tc.name, mac, err, tc.wantMAC)
		}
	}
}

func TestLoadOrCreateKeepsCA(t *testing.T) {
	dir := t.TempDir()
	d1, err := LoadOrCreate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(filepath.Join(dir, "ca.key")); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("ca.key mode = %v (err %v), want 0600", fi.Mode().Perm(), err)
	}
	if !d1.ca.IsCA || !d1.ca.MaxPathLenZero {
		t.Error("deployment CA must be a CA that only signs leaves")
	}

	d2, err := LoadOrCreate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if d1.ca.SerialNumber.Cmp(d2.ca.SerialNumber) != 0 {
		t.Fatal("second start replaced the deployment CA")
	}

	// Losing only the key must not silently mint a new CA.
	if err := os.Remove(filepath.Join(dir, "ca.key")); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrCreate(dir); err == nil {
		t.Fatal("LoadOrCreate accepted a CA certificate without its key")
	}
}

func readLeaf(t *testing.T, path string) *x509.Certificate {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	blk, _ := pem.Decode(b)
	cert, err := x509.ParseCertificate(blk.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	return cert
}

func TestEnsureLeaf(t *testing.T) {
	d, err := LoadOrCreate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sans := []string{"192.0.2.10", "KVM.example.com"}

	certFile, keyFile, err := d.EnsureLeaf("device", sans)
	if err != nil {
		t.Fatal(err)
	}
	if fi, _ := os.Stat(keyFile); fi.Mode().Perm() != 0o600 {
		t.Errorf("leaf key mode = %v, want 0600", fi.Mode().Perm())
	}
	leaf := readLeaf(t, certFile)

	roots := x509.NewCertPool()
	roots.AddCert(d.ca)
	if _, err := leaf.Verify(x509.VerifyOptions{Roots: roots, DNSName: "kvm.example.com"}); err != nil {
		t.Errorf("leaf does not verify for its DNS name: %v", err)
	}
	if _, err := leaf.Verify(x509.VerifyOptions{Roots: roots}); err != nil {
		t.Errorf("leaf does not chain to the deployment CA: %v", err)
	}
	if len(leaf.IPAddresses) != 1 || !leaf.IPAddresses[0].Equal(net.ParseIP("192.0.2.10")) {
		t.Errorf("leaf IPs = %v", leaf.IPAddresses)
	}

	// Same SANs (any order / case): kept.
	if _, _, err := d.EnsureLeaf("device", []string{"kvm.example.com", "192.0.2.10"}); err != nil {
		t.Fatal(err)
	}
	if readLeaf(t, certFile).SerialNumber.Cmp(leaf.SerialNumber) != 0 {
		t.Error("unchanged SANs reissued the leaf")
	}

	// New address: reissued, same CA.
	if _, _, err := d.EnsureLeaf("device", []string{"198.51.100.7"}); err != nil {
		t.Fatal(err)
	}
	moved := readLeaf(t, certFile)
	if moved.SerialNumber.Cmp(leaf.SerialNumber) == 0 || !moved.IPAddresses[0].Equal(net.ParseIP("198.51.100.7")) {
		t.Error("changed SANs did not reissue the leaf")
	}

	// Close to expiry: reissued.
	soon := issue(t, d, "GLKVM Cloud device", []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, time.Now().Add(24*time.Hour))
	if soon.CheckSignatureFrom(d.ca) != nil {
		t.Fatal("test leaf not signed by CA")
	}
	if d.reusable(soon, nil, nil) {
		t.Error("a leaf expiring tomorrow was considered reusable")
	}
}

func TestIsBundledSample(t *testing.T) {
	d, err := LoadOrCreate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if IsBundledSample(d.CAPEM()) {
		t.Error("a freshly generated certificate was flagged as the bundled sample")
	}
	if IsBundledSample([]byte("not pem")) {
		t.Error("garbage was flagged as the bundled sample")
	}
}
