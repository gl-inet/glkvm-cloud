package server

import (
	"crypto/tls"
	"fmt"
	"os"

	"rttys/internal/pkg/pki"
	"rttys/xconfig"

	"github.com/rs/zerolog/log"
)

// setupPKI prepares the certificates the server listens with:
//   - the deployment CA, created once and kept for good, which devices trust;
//   - the device-port certificate it signs for the address devices dial;
//   - the web certificate, unless the operator mounted a usable one.
func setupPKI(cfg *xconfig.Config, dir string) error {
	d, err := pki.LoadOrCreate(dir)
	if err != nil {
		return err
	}
	cfg.DeviceCAFile = d.CAFile()

	host := cfg.DeviceAccessHost
	if host == "" {
		host = cfg.WebrtcIP
	}
	if host == "" {
		log.Warn().Msg("neither DEVICE_ACCESS_HOST nor GLKVM_ACCESS_IP is set: devices cannot verify the device port by address")
	}
	cfg.DeviceCertFile, cfg.DeviceKeyFile, err = d.EnsureLeaf("device", []string{host})
	if err != nil {
		return fmt.Errorf("device-port certificate: %w", err)
	}

	if certFile, keyFile, ok := operatorWebCert(); ok {
		cfg.SslCert, cfg.SslKey = certFile, keyFile
		return nil
	}
	cfg.SslCert, cfg.SslKey, err = d.EnsureLeaf("web", []string{host, "localhost", "127.0.0.1"})
	if err != nil {
		return fmt.Errorf("web certificate: %w", err)
	}
	log.Info().Str("cert", cfg.SslCert).Msg("using the generated web certificate")
	return nil
}

// operatorWebCert returns a web certificate the operator mounted: the v3.0
// directory mount first, then the per-file mounts of earlier compose files.
// The sample certificate those older bundles shipped is skipped because its
// private key was published with it.
func operatorWebCert() (string, string, bool) {
	for _, pair := range [][2]string{
		{xconfig.WebCertFile, xconfig.WebKeyFile},
		{xconfig.SslCert, xconfig.SslKey},
	} {
		certPEM, err := os.ReadFile(pair[0])
		if err != nil {
			continue
		}
		if pki.IsBundledSample(certPEM) {
			log.Warn().Str("cert", pair[0]).Msg("ignoring the sample certificate shipped with older bundles; its private key is public")
			continue
		}
		if _, err := tls.LoadX509KeyPair(pair[0], pair[1]); err != nil {
			log.Warn().Err(err).Str("cert", pair[0]).Msg("ignoring unusable web certificate")
			continue
		}
		return pair[0], pair[1], true
	}
	return "", "", false
}
