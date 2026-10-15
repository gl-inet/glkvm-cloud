package pki

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	caValidity   = 30 * 365 * 24 * time.Hour
	leafValidity = 10 * 365 * 24 * time.Hour
	// A leaf this close to expiry is reissued at startup.
	renewBefore = 365 * 24 * time.Hour
)

// Deployment is the PKI a self-hosted platform generates for itself: a
// long-lived CA that devices trust (it travels inside access codes) and the
// leaf certificates it signs. The CA never changes once created, so devices
// keep trusting the platform when its address or certificates change.
type Deployment struct {
	dir    string
	ca     *x509.Certificate
	caPEM  []byte
	caKey  crypto.Signer
	caFile string
}

// LoadOrCreate loads the deployment CA from dir, creating it on first use.
func LoadOrCreate(dir string) (*Deployment, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("pki: create %s: %w", dir, err)
	}
	d := &Deployment{dir: dir, caFile: filepath.Join(dir, "ca.cer")}
	keyFile := filepath.Join(dir, "ca.key")

	certPEM, certErr := os.ReadFile(d.caFile)
	keyPEM, keyErr := os.ReadFile(keyFile)
	switch {
	case certErr == nil && keyErr == nil:
		ca, key, err := parsePair(certPEM, keyPEM)
		if err != nil {
			return nil, fmt.Errorf("pki: load deployment CA: %w", err)
		}
		d.ca, d.caKey, d.caPEM = ca, key, certPEM
		return d, nil
	case errors.Is(certErr, os.ErrNotExist) && errors.Is(keyErr, os.ErrNotExist):
		// First start: create below.
	default:
		// One half missing would silently replace the CA every device trusts.
		return nil, fmt.Errorf("pki: deployment CA in %s is incomplete (cert: %v, key: %v)", dir, certErr, keyErr)
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber:          randomSerial(),
		Subject:               pkix.Name{CommonName: "GLKVM Cloud Device CA"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(caValidity),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, key.Public(), key)
	if err != nil {
		return nil, err
	}
	certPEM, keyPEM, err = encodePair(der, key)
	if err != nil {
		return nil, err
	}
	if err := writeFile(keyFile, keyPEM, 0o600); err != nil {
		return nil, err
	}
	if err := writeFile(d.caFile, certPEM, 0o644); err != nil {
		return nil, err
	}
	d.ca, _ = x509.ParseCertificate(der)
	d.caKey, d.caPEM = key, certPEM
	return d, nil
}

// CAFile is the path of the deployment CA certificate.
func (d *Deployment) CAFile() string { return d.caFile }

// CAPEM is the deployment CA certificate in PEM form.
func (d *Deployment) CAPEM() []byte { return d.caPEM }

// EnsureLeafKeeping is EnsureLeaf for a certificate whose clients keep the
// address they were given, as devices do: a new certificate also covers the
// addresses of the current one, so that clients still dialing an old address
// can verify it.
func (d *Deployment) EnsureLeafKeeping(name string, sans []string) (certFile, keyFile string, err error) {
	all := slices.Clone(sans)
	if b, err := os.ReadFile(filepath.Join(d.dir, name+".cer")); err == nil {
		if blk, _ := pem.Decode(b); blk != nil {
			if cur, err := x509.ParseCertificate(blk.Bytes); err == nil && cur.CheckSignatureFrom(d.ca) == nil {
				for _, ip := range cur.IPAddresses {
					all = append(all, ip.String())
				}
				all = append(all, cur.DNSNames...)
			}
		}
	}
	return d.EnsureLeaf(name, all)
}

// EnsureLeaf returns a server certificate <name>.cer/.key in the deployment
// directory for the given SANs (IPs or DNS names). The existing pair is kept
// when it is signed by this CA, covers exactly these SANs and is not about to
// expire; otherwise a new one is issued.
func (d *Deployment) EnsureLeaf(name string, sans []string) (certFile, keyFile string, err error) {
	certFile = filepath.Join(d.dir, name+".cer")
	keyFile = filepath.Join(d.dir, name+".key")
	ips, dns := splitSANs(sans)

	if certPEM, err := os.ReadFile(certFile); err == nil {
		if keyPEM, err := os.ReadFile(keyFile); err == nil {
			if leaf, _, err := parsePair(certPEM, keyPEM); err == nil && d.reusable(leaf, ips, dns) {
				return certFile, keyFile, nil
			}
		}
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}
	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber:          randomSerial(),
		Subject:               pkix.Name{CommonName: "GLKVM Cloud " + name},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(leafValidity),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           ips,
		DNSNames:              dns,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, d.ca, key.Public(), d.caKey)
	if err != nil {
		return "", "", err
	}
	certPEM, keyPEM, err := encodePair(der, key)
	if err != nil {
		return "", "", err
	}
	if err := writeFile(keyFile, keyPEM, 0o600); err != nil {
		return "", "", err
	}
	if err := writeFile(certFile, certPEM, 0o644); err != nil {
		return "", "", err
	}
	return certFile, keyFile, nil
}

func (d *Deployment) reusable(leaf *x509.Certificate, ips []net.IP, dns []string) bool {
	if leaf.CheckSignatureFrom(d.ca) != nil {
		return false
	}
	if time.Until(leaf.NotAfter) < renewBefore {
		return false
	}
	return sameIPs(leaf.IPAddresses, ips) && slices.Equal(sortedLower(leaf.DNSNames), dns)
}

// splitSANs sorts SAN entries into IP addresses and (lowercased, sorted,
// de-duplicated) DNS names.
func splitSANs(sans []string) ([]net.IP, []string) {
	var ips []net.IP
	var dns []string
	for _, s := range sans {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if ip := net.ParseIP(s); ip != nil {
			if !slices.ContainsFunc(ips, ip.Equal) {
				ips = append(ips, ip)
			}
			continue
		}
		dns = append(dns, strings.ToLower(s))
	}
	return ips, sortedLower(dns)
}

func sortedLower(names []string) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		out = append(out, strings.ToLower(n))
	}
	slices.Sort(out)
	return slices.Compact(out)
}

func sameIPs(a, b []net.IP) bool {
	if len(a) != len(b) {
		return false
	}
	for _, ip := range a {
		if !slices.ContainsFunc(b, ip.Equal) {
			return false
		}
	}
	return true
}

func parsePair(certPEM, keyPEM []byte) (*x509.Certificate, crypto.Signer, error) {
	cb, _ := pem.Decode(certPEM)
	if cb == nil {
		return nil, nil, errors.New("certificate is not PEM")
	}
	cert, err := x509.ParseCertificate(cb.Bytes)
	if err != nil {
		return nil, nil, err
	}
	kb, _ := pem.Decode(keyPEM)
	if kb == nil {
		return nil, nil, errors.New("private key is not PEM")
	}
	k, err := x509.ParsePKCS8PrivateKey(kb.Bytes)
	if err != nil {
		return nil, nil, err
	}
	signer, ok := k.(crypto.Signer)
	if !ok {
		return nil, nil, errors.New("private key cannot sign")
	}
	if !publicKeysEqual(cert.PublicKey, signer.Public()) {
		return nil, nil, errors.New("private key does not match certificate")
	}
	return cert, signer, nil
}

func publicKeysEqual(a, b crypto.PublicKey) bool {
	ak, ok := a.(interface{ Equal(crypto.PublicKey) bool })
	return ok && ak.Equal(b)
}

func encodePair(der []byte, key *ecdsa.PrivateKey) (certPEM, keyPEM []byte, err error) {
	kder, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, err
	}
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: kder})
	return certPEM, keyPEM, nil
}

func randomSerial() *big.Int {
	n, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
	return n
}

// writeFile replaces path atomically so a crash never leaves a half-written
// key or certificate behind.
func writeFile(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// IsBundledSample reports whether certPEM is the sample web certificate that
// docker-compose bundles before v3.0 shipped. Its private key was published
// with it, so it must never be served.
func IsBundledSample(certPEM []byte) bool {
	b, _ := pem.Decode(certPEM)
	return b != nil && Fingerprint(b.Bytes) == bundledSampleFingerprint
}

const bundledSampleFingerprint = "23c5774499d739baad7346c301106286c62ebea527247175554059a3cc61ade2"
