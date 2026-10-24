package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"strings"
	"time"

	"rttys/internal/domain/access"
	"rttys/internal/pkg/pki"
	"rttys/internal/store/sqlite"

	"github.com/rs/zerolog/log"
)

// ListenDevicesMTLS serves the mTLS device port. Devices present their GL
// factory certificate; an unlisted device joins with a one-time registration
// code, a whitelisted one simply reconnects.
func (srv *RttyServer) ListenDevicesMTLS() {
	cfg := &srv.cfg

	cert, err := tls.LoadX509KeyPair(cfg.DeviceCertFile, cfg.DeviceKeyFile)
	if err != nil {
		log.Fatal().Err(err).Msg("load device-port certificate")
	}
	roots, err := pki.GLRootPool()
	if err != nil {
		log.Fatal().Err(err).Msg("load GL root CA")
	}

	ln, err := net.Listen("tcp", cfg.AddrDevMTLS)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer ln.Close()

	log.Info().Msgf("Listen devices (mTLS) on: %s", ln.Addr().(*net.TCPAddr))

	srv.serveDevicesMTLS(tls.NewListener(ln, deviceTLSConfig(cert, roots)))
}

// deviceTLSConfig requires a client certificate that chains to roots and is a
// factory device certificate (see pki.DeviceMAC).
func deviceTLSConfig(cert tls.Certificate, roots *x509.CertPool) *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    roots,
		MinVersion:   tls.VersionTLS12,
		VerifyConnection: func(cs tls.ConnectionState) error {
			if len(cs.PeerCertificates) == 0 {
				return errors.New("no client certificate")
			}
			_, err := pki.DeviceMAC(cs.PeerCertificates[0])
			return err
		},
	}
}

func (srv *RttyServer) serveDevicesMTLS(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			log.Error().Msg(err.Error())
			continue
		}
		go handleDeviceConnMTLS(srv, conn.(*tls.Conn))
	}
}

// handleDeviceConnMTLS finishes the handshake within the register timeout, so a
// silent peer cannot hold the connection, then serves it as the device the
// certificate names.
func handleDeviceConnMTLS(srv *RttyServer, conn *tls.Conn) {
	conn.SetDeadline(time.Now().Add(WaitRegistTimeout))
	if err := conn.Handshake(); err != nil {
		log.Warn().Err(err).Str("ip", remoteIP(conn)).Msg("mtls: handshake failed")
		conn.Close()
		return
	}
	conn.SetDeadline(time.Time{})

	mac, err := pki.DeviceMAC(conn.ConnectionState().PeerCertificates[0])
	if err != nil { // already enforced by VerifyConnection
		conn.Close()
		return
	}
	handleDeviceConnection(srv, conn, mac)
}

// registerMTLS admits a device from the mTLS port. Its identity is the MAC in
// the factory certificate: a whitelisted MAC comes in under the device ID it
// is listed with, spending a valid registration code if it brings one; an
// unlisted one must bring a valid registration code, which adds it to the
// whitelist.
func (dev *Device) registerMTLS() byte {
	cont := sqlite.TryContainer()
	if cont == nil || cont.Access == nil {
		log.Warn().Str("mac", dev.certMAC).Msg("mtls: register before the database is ready")
		return devRegErrInternal
	}
	ctx := context.Background()

	entry, err := cont.Access.GetByMAC(ctx, dev.certMAC)
	if err != nil {
		log.Error().Err(err).Str("mac", dev.certMAC).Msg("mtls: whitelist lookup failed")
		return devRegErrInternal
	}
	if entry != nil {
		// A listed device bringing a code was pasted one again (e.g. after a
		// reset): spend it, so it shows this device and admits no other. An
		// invalid code does not keep a listed device out.
		if dev.regCode != "" {
			claimed, err := cont.Access.ClaimListed(ctx, access.HashCode(dev.regCode), dev.certMAC, time.Now().Unix())
			if err != nil {
				log.Error().Err(err).Str("mac", dev.certMAC).Msg("mtls: claim failed")
				return devRegErrInternal
			}
			if claimed {
				log.Info().Str("mac", dev.certMAC).Str("devid", entry.Devid).Msg("mtls: listed device joined again with a registration code")
				if cont.DeviceLogSvc != nil {
					cont.DeviceLogSvc.RecordDeviceJoined(ctx, entry.Devid, dev.certMAC, remoteIP(dev.conn))
				}
			}
		}
		dev.id, dev.desc, dev.joinedAt = entry.Devid, dev.certMAC, entry.CreatedAt
		return 0
	}

	if dev.regCode == "" {
		log.Warn().Str("mac", dev.certMAC).Str("ip", remoteIP(dev.conn)).Msg("mtls: device is not registered and sent no registration code")
		return devRegErrNotRegistered
	}

	// A new device keeps the ID it registers with (its C rtty ID), unless the
	// platform already knows its MAC under another one.
	devid := strings.TrimSpace(dev.id)
	if devid == "" || len(devid) > 32 || strings.ContainsAny(devid, " \t") {
		devid = dev.certMAC
	}
	now := time.Now().Unix()
	devid, err = cont.Access.Claim(ctx, access.HashCode(dev.regCode), dev.certMAC, devid, now)
	switch {
	case errors.Is(err, access.ErrCodeInvalid):
		log.Warn().Str("mac", dev.certMAC).Str("ip", remoteIP(dev.conn)).Msg("mtls: invalid registration code")
		return devRegErrInvalidRegCode
	case errors.Is(err, access.ErrDevidTaken):
		log.Warn().Str("mac", dev.certMAC).Str("devid", dev.id).Msg("mtls: device ID belongs to another device")
		return devRegErrDevidTaken
	case err != nil:
		log.Error().Err(err).Str("mac", dev.certMAC).Msg("mtls: claim failed")
		return devRegErrInternal
	}

	dev.id, dev.desc, dev.joinedAt = devid, dev.certMAC, now
	log.Info().Str("mac", dev.certMAC).Str("devid", devid).Msg("mtls: device joined with a registration code")
	if cont.DeviceLogSvc != nil {
		cont.DeviceLogSvc.RecordDeviceJoined(ctx, devid, dev.certMAC, remoteIP(dev.conn))
	}
	return 0
}

func remoteIP(conn net.Conn) string {
	if conn == nil {
		return ""
	}
	if host, _, err := net.SplitHostPort(conn.RemoteAddr().String()); err == nil {
		return host
	}
	return conn.RemoteAddr().String()
}
