package server

import (
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	xlog "rttys/log"
	"rttys/xconfig"
)

var logFileOnce sync.Once

// discardLogFile points the log package's file hook at the null device. Code
// under test logs, and without a path the hook fails fatally. It is set once
// for the whole package: server goroutines can still log after the test that
// started them has returned.
func discardLogFile() {
	logFileOnce.Do(func() { xlog.SetPath(os.DevNull) })
}

func leafSANs(t *testing.T, path string) ([]net.IP, []string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	blk, _ := pem.Decode(b)
	c, err := x509.ParseCertificate(blk.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	return c.IPAddresses, c.DNSNames
}

func TestSetupPKI(t *testing.T) {
	dir := t.TempDir()
	discardLogFile()

	cfg := xconfig.Config{WebrtcIP: "192.0.2.10"}
	if err := setupPKI(&cfg, dir); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{cfg.DeviceCAFile, cfg.DeviceCertFile, cfg.DeviceKeyFile} {
		if !strings.HasPrefix(f, dir) {
			t.Errorf("%s is outside the PKI dir %s", f, dir)
		}
	}
	if ips, _ := leafSANs(t, cfg.DeviceCertFile); len(ips) != 1 || !ips[0].Equal(net.ParseIP("192.0.2.10")) {
		t.Errorf("device cert IPs = %v, want the access IP", ips)
	}
	// No operator certificate on this machine, so the generated one is used.
	if filepath.Dir(cfg.SslCert) != dir {
		t.Errorf("web cert = %s, want the generated one in %s", cfg.SslCert, dir)
	}

	// An explicit device access host wins over the WebRTC IP; the CA stays.
	ca1, _ := os.ReadFile(cfg.DeviceCAFile)
	cfg2 := xconfig.Config{WebrtcIP: "192.0.2.10", DeviceAccessHost: "kvm.example.com"}
	if err := setupPKI(&cfg2, dir); err != nil {
		t.Fatal(err)
	}
	if _, dns := leafSANs(t, cfg2.DeviceCertFile); len(dns) != 1 || dns[0] != "kvm.example.com" {
		t.Errorf("device cert DNS = %v, want kvm.example.com", dns)
	}
	if ca2, _ := os.ReadFile(cfg2.DeviceCAFile); string(ca1) != string(ca2) {
		t.Error("changing the access host replaced the deployment CA")
	}

	// The token port serves the device-port certificate, which the deployment
	// CA signs, not the web certificate an operator may replace.
	if c, k := tokenPortCert(&cfg2); c != cfg2.DeviceCertFile || k != cfg2.DeviceKeyFile {
		t.Errorf("token port cert = %s/%s, want the device-port one", c, k)
	}
	if !verifiesWith(t, cfg2.DeviceCertFile, cfg2.DeviceCAFile) {
		t.Error("the token port certificate does not chain to the deployment CA")
	}
}

func TestTokenPortCertWithoutDeploymentCA(t *testing.T) {
	cfg := xconfig.Config{SslCert: "web.cer", SslKey: "web.key"}
	if c, k := tokenPortCert(&cfg); c != "web.cer" || k != "web.key" {
		t.Errorf("token port cert = %s/%s, want the web one when there is no device-port cert", c, k)
	}
}

func verifiesWith(t *testing.T, leafFile, caFile string) bool {
	t.Helper()
	leafPEM, err := os.ReadFile(leafFile)
	if err != nil {
		t.Fatal(err)
	}
	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caPEM)
	blk, _ := pem.Decode(leafPEM)
	leaf, err := x509.ParseCertificate(blk.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	_, err = leaf.Verify(x509.VerifyOptions{Roots: pool, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}})
	return err == nil
}
