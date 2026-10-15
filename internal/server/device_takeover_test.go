package server

import (
	"crypto/x509"
	"errors"
	"net"
	"testing"
	"time"
)

var clientAuthEKU = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}

// serveToken serves the fixture's server on a token port as well, where the C
// rtty connects. No token is configured, so any token is accepted.
func (f *mtlsFixture) serveToken(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handleDeviceConnection(f.srv, conn, "")
		}
	}()
	return ln.Addr().String()
}

func registerToken(t *testing.T, addr, devid, mac string) (byte, net.Conn) {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	code, conn, err := sendRegister(conn, devid, mac, "")
	if err != nil {
		t.Fatal(err)
	}
	return code, conn
}

func (f *mtlsFixture) waitStatus(t *testing.T, devid, status string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		var got string
		f.db.Gorm().Raw(`SELECT status FROM devices WHERE ddns = ?`, devid).Scan(&got)
		if got == status {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("device %s is %q, want %q", devid, got, status)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (f *mtlsFixture) events(devid, typ string) int64 {
	var n int64
	f.db.Gorm().Raw(`SELECT COUNT(1) FROM device_event_logs WHERE device_id = ? AND event_type = ?`, devid, typ).Scan(&n)
	return n
}

// closedByServer reports whether the server closes conn within 3 seconds,
// discarding whatever it sends before that.
func closedByServer(conn net.Conn) bool {
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 512)
	for {
		if _, err := conn.Read(buf); err != nil {
			var ne net.Error
			return !errors.As(err, &ne) || !ne.Timeout()
		}
	}
}

// Upgrading in place: the device's C rtty is online on the token port when
// rtty-go joins on the mTLS port under the same ID.
func TestMTLSTakesOverTokenConnection(t *testing.T) {
	factory := newTestCA(t)
	f := newMTLSFixture(t, factory)
	tokenAddr := f.serveToken(t)

	code, legacy := registerToken(t, tokenAddr, "zh71fb1", "94:83:c4:b7:1f:b1")
	if code != 0 {
		t.Fatalf("token register: %d", code)
	}
	f.waitStatus(t, "zh71fb1", "online")

	f.addCode(t, "code")
	code, conn, err := f.register(factory.device(t, "9483c4b71fb1", clientAuthEKU), "zh71fb1", "9483c4b71fb1", "code")
	if err != nil || code != 0 {
		t.Fatalf("join while the token connection is online: (%d, %v), want 0", code, err)
	}
	if !closedByServer(legacy) {
		t.Error("the token connection was not dropped")
	}
	legacy.Close()
	if d := f.srv.GetDevice("", "zh71fb1"); d == nil || d.certMAC == "" {
		t.Error("the mTLS connection does not hold the device ID")
	}
	f.waitStatus(t, "zh71fb1", "online")
	if n := f.events("zh71fb1", "device_offline"); n != 0 {
		t.Errorf("%d device_offline events, want none: the device never went offline", n)
	}

	// The C rtty coming back does not get the ID back.
	code, again := registerToken(t, tokenAddr, "zh71fb1", "94:83:c4:b7:1f:b1")
	again.Close()
	if code != devRegErrIdConflicting {
		t.Errorf("token reconnect: %d, want %d", code, devRegErrIdConflicting)
	}
	f.closeAndWait(t, conn, "zh71fb1")
}

// After a network change the device reconnects before the platform has noticed
// that its old link is dead.
func TestMTLSTakesOverItsStaleConnection(t *testing.T) {
	factory := newTestCA(t)
	f := newMTLSFixture(t, factory)
	dev := factory.device(t, "9483c4b71fb1", clientAuthEKU)

	f.addCode(t, "code")
	code, stale, err := f.register(dev, "zh71fb1", "9483c4b71fb1", "code")
	if err != nil || code != 0 {
		t.Fatalf("join: (%d, %v)", code, err)
	}
	f.waitStatus(t, "zh71fb1", "online")

	code, conn, err := f.register(dev, "zh71fb1", "9483c4b71fb1", "")
	if err != nil || code != 0 {
		t.Fatalf("reconnect while the old link is open: (%d, %v), want 0", code, err)
	}
	if !closedByServer(stale) {
		t.Error("the stale connection was not dropped")
	}
	stale.Close()
	f.waitStatus(t, "zh71fb1", "online")
	if n := f.events("zh71fb1", "device_offline"); n != 0 {
		t.Errorf("%d device_offline events, want none", n)
	}
	f.closeAndWait(t, conn, "zh71fb1")
}

// Anyone with the shared token can claim any ID, so a token connection never
// displaces a device.
func TestTokenConnectionCannotTakeOver(t *testing.T) {
	factory := newTestCA(t)
	f := newMTLSFixture(t, factory)
	tokenAddr := f.serveToken(t)

	f.addCode(t, "code")
	code, conn, err := f.register(factory.device(t, "9483c4b71fb1", clientAuthEKU), "zh71fb1", "9483c4b71fb1", "code")
	if err != nil || code != 0 {
		t.Fatalf("join: (%d, %v)", code, err)
	}

	code, other := registerToken(t, tokenAddr, "zh71fb1", "9483c4b71fb1")
	other.Close()
	if code != devRegErrIdConflicting {
		t.Errorf("token register over an mTLS device: %d, want %d", code, devRegErrIdConflicting)
	}
	if d := f.srv.GetDevice("", "zh71fb1"); d == nil || d.certMAC == "" {
		t.Error("the mTLS device lost its ID")
	}
	f.closeAndWait(t, conn, "zh71fb1")
}
