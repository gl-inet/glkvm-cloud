package server

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"io"
	"math/big"
	"net"
	"path/filepath"
	"testing"
	"time"

	"rttys/internal/domain/access"
	"rttys/internal/domain/devicelog"
	"rttys/internal/pkg/pki"
	"rttys/internal/store/sqlite"
	"rttys/xconfig"
)

// testCA is a stand-in for the GL factory CA.
type testCA struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
}

func newTestCA(t *testing.T) *testCA {
	t.Helper()
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test factory CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, key.Public(), key)
	if err != nil {
		t.Fatal(err)
	}
	cert, _ := x509.ParseCertificate(der)
	return &testCA{cert, key}
}

func (ca *testCA) pool() *x509.CertPool {
	p := x509.NewCertPool()
	p.AddCert(ca.cert)
	return p
}

// device issues a client certificate shaped like a factory certificate.
func (ca *testCA) device(t *testing.T, cn string, eku []x509.ExtKeyUsage) tls.Certificate {
	t.Helper()
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  eku,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca.cert, key.Public(), ca.key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

type mtlsFixture struct {
	srv    *RttyServer
	addr   string
	rootCA *x509.CertPool // what devices trust: the deployment CA
	db     *sqlite.AppDB
}

func newMTLSFixture(t *testing.T, factory *testCA) *mtlsFixture {
	t.Helper()
	dir := t.TempDir()
	discardLogFile()
	ctx := context.Background()

	db, err := sqlite.Open(ctx, sqlite.Options{DSN: filepath.Join(dir, "mtls.db")})
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlite.InitSchema(ctx, db.SQL(), "../../docker-compose/database/schema.sql"); err != nil {
		t.Fatal(err)
	}
	g := db.Gorm()
	prev := sqlite.TryContainer()
	sqlite.SetContainer(&sqlite.Container{
		Gorm:         g,
		DeviceMeta:   sqlite.NewDeviceMetaRepo(g),
		DeviceLogSvc: devicelog.NewService(sqlite.NewDeviceLogRepo(g)),
		Access:       sqlite.NewAccessRepo(g),
	})

	dep, err := pki.LoadOrCreate(filepath.Join(dir, "pki"))
	if err != nil {
		t.Fatal(err)
	}
	certFile, keyFile, err := dep.EnsureLeaf("device", []string{"127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	serverCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(dep.CAPEM())

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := New(xconfig.Config{})
	go srv.serveDevicesMTLS(tls.NewListener(ln, deviceTLSConfig(serverCert, factory.pool())))

	t.Cleanup(func() {
		ln.Close()
		sqlite.SetContainer(prev)
		db.Close()
	})
	return &mtlsFixture{srv: srv, addr: ln.Addr().String(), rootCA: roots, db: db}
}

func (f *mtlsFixture) addCode(t *testing.T, code string) {
	t.Helper()
	now := time.Now().Unix()
	if err := f.db.Gorm().Exec(
		`INSERT INTO registration_codes (code_hash, created_at, expires_at) VALUES (?, ?, ?)`,
		access.HashCode(code), now, now+3600,
	).Error; err != nil {
		t.Fatal(err)
	}
}

func tlv(typ byte, v string) []byte {
	b := []byte{typ, 0, 0}
	binary.BigEndian.PutUint16(b[1:], uint16(len(v)))
	return append(b, v...)
}

// register connects with cert, sends an rtty register frame and returns the
// server's register code. The connection is returned open so the device stays
// online until the caller closes it.
func (f *mtlsFixture) register(cert tls.Certificate, devid, mac, code string) (byte, net.Conn, error) {
	conn, err := tls.Dial("tcp", f.addr, &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      f.rootCA,
		ServerName:   "127.0.0.1",
	})
	if err != nil {
		return 0, nil, err
	}
	payload := []byte{5} // proto 5, TLV attributes
	payload = append(payload, tlv(msgRegAttrDevid, devid)...)
	payload = append(payload, tlv(msgRegAttrDescription, mac)...)
	if code != "" {
		payload = append(payload, tlv(msgRegAttrRegCode, code)...)
	}
	head := []byte{msgTypeRegister, 0, 0}
	binary.BigEndian.PutUint16(head[1:], uint16(len(payload)))
	if _, err := conn.Write(append(head, payload...)); err != nil {
		conn.Close()
		return 0, nil, err
	}

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if _, err := io.ReadFull(conn, head); err != nil {
		conn.Close()
		return 0, nil, err
	}
	resp := make([]byte, binary.BigEndian.Uint16(head[1:]))
	if _, err := io.ReadFull(conn, resp); err != nil {
		conn.Close()
		return 0, nil, err
	}
	conn.SetReadDeadline(time.Time{})
	return resp[0], conn, nil
}

// closeAndWait closes a registered device's connection and waits until the
// server has finished taking it offline. The last step of that writes the
// database, so it must not outlive the test's database and container.
func (f *mtlsFixture) closeAndWait(t *testing.T, conn net.Conn, devid string) {
	t.Helper()
	conn.Close()
	deadline := time.Now().Add(3 * time.Second)
	for {
		var status string
		f.db.Gorm().Raw(`SELECT status FROM devices WHERE ddns = ?`, devid).Scan(&status)
		if status == "offline" && f.srv.GetDevice("", devid) == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("device %s still online (status %q)", devid, status)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestMTLSRegister(t *testing.T) {
	factory := newTestCA(t)
	f := newMTLSFixture(t, factory)
	clientAuth := []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	dev := factory.device(t, "9483c4b71fb1", clientAuth)

	// Unlisted, no code.
	if code, conn, err := f.register(dev, "zh71fb1", "9483c4b71fb1", ""); err != nil || code != devRegErrNotRegistered {
		t.Fatalf("no code: (%d, %v), want %d", code, err, devRegErrNotRegistered)
	} else {
		conn.Close()
	}
	// Unlisted, unknown code.
	if code, conn, err := f.register(dev, "zh71fb1", "9483c4b71fb1", "nope"); err != nil || code != devRegErrInvalidRegCode {
		t.Fatalf("bad code: (%d, %v), want %d", code, err, devRegErrInvalidRegCode)
	} else {
		conn.Close()
	}

	// Valid code: joins and comes online.
	f.addCode(t, "code-1")
	code, conn, err := f.register(dev, "zh71fb1", "9483c4b71fb1", "code-1")
	if err != nil || code != 0 {
		t.Fatalf("valid code: (%d, %v), want 0", code, err)
	}
	if f.srv.GetDevice("", "zh71fb1") == nil {
		t.Fatal("joined device is not online")
	}
	var joined int64
	f.db.Gorm().Raw(`SELECT COUNT(1) FROM device_event_logs WHERE event_type = 'device_joined' AND device_id = 'zh71fb1'`).Scan(&joined)
	if joined != 1 {
		t.Errorf("device_joined events = %d, want 1", joined)
	}
	f.closeAndWait(t, conn, "zh71fb1")

	// Whitelisted now: reconnects without a code, and the frame cannot rename it.
	code, conn, err = f.register(dev, "renamed", "000000000000", "")
	if err != nil || code != 0 {
		t.Fatalf("reconnect: (%d, %v), want 0", code, err)
	}
	if f.srv.GetDevice("", "zh71fb1") == nil || f.srv.GetDevice("", "renamed") != nil {
		t.Error("reconnect did not keep the whitelisted device ID")
	}
	f.closeAndWait(t, conn, "zh71fb1")

	// Another device cannot take that ID, and its code survives the attempt.
	f.addCode(t, "code-2")
	other := factory.device(t, "9483c4e1ddff", clientAuth)
	if code, conn, err := f.register(other, "zh71fb1", "9483c4e1ddff", "code-2"); err != nil || code != devRegErrDevidTaken {
		t.Fatalf("taken ID: (%d, %v), want %d", code, err, devRegErrDevidTaken)
	} else {
		conn.Close()
	}
	if code, conn, err := f.register(other, "dl1ddff", "9483c4e1ddff", "code-2"); err != nil || code != 0 {
		t.Fatalf("free ID with the surviving code: (%d, %v), want 0", code, err)
	} else {
		f.closeAndWait(t, conn, "dl1ddff")
	}
}

func TestMTLSRejectsNonDeviceCertificates(t *testing.T) {
	factory := newTestCA(t)
	f := newMTLSFixture(t, factory)
	f.addCode(t, "code")

	cases := map[string]tls.Certificate{
		"no extended key usage": factory.device(t, "9483c4b71fb1", nil),
		"server auth only":      factory.device(t, "9483c4b71fb1", []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}),
		"CN is not a MAC":       factory.device(t, "kvm-server", []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}),
		"unknown issuer":        newTestCA(t).device(t, "9483c4b71fb1", []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}),
	}
	for name, cert := range cases {
		if code, conn, err := f.register(cert, "zh71fb1", "9483c4b71fb1", "code"); err == nil {
			conn.Close()
			t.Errorf("%s: registered with code %d, want the handshake refused", name, code)
		}
	}
	if f.srv.GetDevice("", "zh71fb1") != nil {
		t.Error("a non-device certificate got a device online")
	}
}

func TestMTLSDeletedDeviceIsRejected(t *testing.T) {
	factory := newTestCA(t)
	f := newMTLSFixture(t, factory)
	dev := factory.device(t, "9483c4b71fb1", []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth})

	f.addCode(t, "code")
	code, conn, err := f.register(dev, "zh71fb1", "9483c4b71fb1", "code")
	if err != nil || code != 0 {
		t.Fatalf("join: (%d, %v)", code, err)
	}

	// The device row is written just after the register reply.
	var id int64
	for deadline := time.Now().Add(3 * time.Second); id == 0; time.Sleep(10 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("device row never appeared")
		}
		f.db.Gorm().Raw(`SELECT id FROM devices WHERE ddns = 'zh71fb1'`).Scan(&id)
	}

	// What DELETE /api/devices/:id does: database first, then drop the link.
	if _, _, err := sqlite.NewAccessRepo(f.db.Gorm()).RemoveDevice(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	f.srv.DisconnectDevice("zh71fb1")

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := conn.Read(make([]byte, 1)); err == nil {
		t.Error("the deleted device's connection is still open")
	}
	conn.Close()
	if f.srv.GetDevice("", "zh71fb1") != nil {
		t.Error("the deleted device is still online")
	}

	if code, conn, err := f.register(dev, "zh71fb1", "9483c4b71fb1", ""); err != nil || code != devRegErrNotRegistered {
		t.Fatalf("reconnect after delete: (%d, %v), want %d", code, err, devRegErrNotRegistered)
	} else {
		conn.Close()
	}
}
