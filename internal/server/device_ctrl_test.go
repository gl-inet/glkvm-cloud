package server

import (
	"encoding/binary"
	"encoding/json"
	"net"
	"testing"
	"time"

	"rttys/xconfig"
)

func TestMTLSCloudInfo(t *testing.T) {
	factory := newTestCA(t)
	f := newMTLSFixtureWith(t, factory, xconfig.Config{WebrtcIP: "192.0.2.10", AddrUser: ":8443"})

	f.addCode(t, "code")
	code, conn, err := f.register(factory.device(t, "9483c4b71fb1", clientAuthEKU), "zh71fb1", "9483c4b71fb1", "code")
	if err != nil || code != 0 {
		t.Fatalf("join: (%d, %v)", code, err)
	}

	typ, data, err := readFrame(conn)
	if err != nil || typ != msgTypeCtrl {
		t.Fatalf("frame after the register reply: type %d, %v; want the cloud info", typ, err)
	}
	var e struct {
		T, Cmd string
		Data   cloudInfo
	}
	if err := json.Unmarshal(data, &e); err != nil {
		t.Fatal(err)
	}
	var joinedAt int64
	f.db.Gorm().Raw(`SELECT created_at FROM device_whitelist WHERE mac = '9483c4b71fb1'`).Scan(&joinedAt)
	want := cloudInfo{Devid: "zh71fb1", WebURL: "https://192.0.2.10:8443", JoinedAt: joinedAt}
	if e.T != "evt" || e.Cmd != "cloud.info" || e.Data != want || joinedAt == 0 {
		t.Errorf("got %s %s %+v, want evt cloud.info %+v", e.T, e.Cmd, e.Data, want)
	}
	f.closeAndWait(t, conn, "zh71fb1")
}

func TestDeviceWebURL(t *testing.T) {
	for _, c := range []struct {
		cfg  xconfig.Config
		want string
	}{
		{xconfig.Config{AddrUser: ":443"}, ""},
		{xconfig.Config{WebrtcIP: "192.0.2.10", AddrUser: ":443"}, "https://192.0.2.10"},
		{xconfig.Config{WebrtcIP: "192.0.2.10", AddrUser: ":8443"}, "https://192.0.2.10:8443"},
		{xconfig.Config{WebrtcIP: "2001:db8::1", AddrUser: ":443"}, "https://[2001:db8::1]"},
		{xconfig.Config{WebrtcIP: "2001:db8::1", AddrUser: ":8443"}, "https://[2001:db8::1]:8443"},
		{xconfig.Config{WebrtcIP: "192.0.2.10", DeviceAccessHost: "kvm.example.com", AddrUser: ":443"}, "https://kvm.example.com"},
		{xconfig.Config{DeviceAccessHost: "kvm.example.com", WebUIHost: "web.example.com:9443", AddrUser: ":443"}, "https://web.example.com:9443"},
		{xconfig.Config{WebrtcIP: "192.0.2.10", SelfhostWebUIURL: " https://kvm.example.com/ "}, "https://kvm.example.com/"},
	} {
		if got := deviceWebURL(&c.cfg); got != c.want {
			t.Errorf("%+v: %q, want %q", c.cfg, got, c.want)
		}
	}
}

type ctrlResp struct {
	T, ID string
	OK    bool
	Err   *ctrlErr
}

// ctrlCall sends a control request from a device and returns the answer,
// skipping other frames such as cloud.info.
func ctrlCall(t *testing.T, conn net.Conn, id, cmd string) ctrlResp {
	t.Helper()
	b, _ := json.Marshal(map[string]any{"v": 1, "t": "req", "id": id, "cmd": cmd})
	head := []byte{msgTypeCtrl, 0, 0}
	binary.BigEndian.PutUint16(head[1:], uint16(len(b)))
	if _, err := conn.Write(append(head, b...)); err != nil {
		t.Fatal(err)
	}
	for {
		typ, data, err := readFrame(conn)
		if err != nil {
			t.Fatalf("waiting for the answer to %s: %v", cmd, err)
		}
		var r ctrlResp
		if typ == msgTypeCtrl && json.Unmarshal(data, &r) == nil && r.T == "resp" && r.ID == id {
			return r
		}
	}
}

func TestMTLSDeviceLeaves(t *testing.T) {
	factory := newTestCA(t)
	f := newMTLSFixture(t, factory)
	dev := factory.device(t, "9483c4b71fb1", clientAuthEKU)

	f.addCode(t, "code")
	code, conn, err := f.register(dev, "zh71fb1", "9483c4b71fb1", "code")
	if err != nil || code != 0 {
		t.Fatalf("join: (%d, %v)", code, err)
	}
	f.waitStatus(t, "zh71fb1", "online")

	if r := ctrlCall(t, conn, "d1", "cloud.leave"); !r.OK {
		t.Fatalf("cloud.leave answered %+v", r)
	}
	if !closedByServer(conn) {
		t.Error("the connection of the device that left is still open")
	}
	conn.Close()
	// Taking it offline is the last thing the server does with it.
	for deadline := time.Now().Add(3 * time.Second); f.events("zh71fb1", "device_offline") == 0; time.Sleep(10 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("the device that left never went offline")
		}
	}

	var listed, rows int64
	f.db.Gorm().Raw(`SELECT COUNT(1) FROM device_whitelist WHERE mac = '9483c4b71fb1'`).Scan(&listed)
	f.db.Gorm().Raw(`SELECT COUNT(1) FROM devices WHERE ddns = 'zh71fb1'`).Scan(&rows)
	if listed != 0 || rows != 0 {
		t.Errorf("after leaving: %d whitelist entries, %d device rows; want none", listed, rows)
	}
	if n := f.events("zh71fb1", "device_left"); n != 1 {
		t.Errorf("%d device_left events, want 1", n)
	}

	// Coming back takes a new code.
	code, conn, err = f.register(dev, "zh71fb1", "9483c4b71fb1", "")
	if err != nil || code != devRegErrNotRegistered {
		t.Fatalf("reconnect after leaving: (%d, %v), want %d", code, err, devRegErrNotRegistered)
	}
	conn.Close()
}

// A token connection proves nothing, so it cannot remove a device.
func TestTokenDeviceCannotLeave(t *testing.T) {
	factory := newTestCA(t)
	f := newMTLSFixture(t, factory)
	tokenAddr := f.serveToken(t)

	code, conn := registerToken(t, tokenAddr, "zh71fb1", "94:83:c4:b7:1f:b1")
	if code != 0 {
		t.Fatalf("token register: %d", code)
	}
	f.waitStatus(t, "zh71fb1", "online")

	if r := ctrlCall(t, conn, "d1", "cloud.leave"); r.OK || r.Err == nil || r.Err.Code != "not_supported" {
		t.Errorf("token device leaving: %+v, want not_supported", r)
	}
	if n := f.events("zh71fb1", "device_left"); n != 0 {
		t.Errorf("%d device_left events, want none", n)
	}
	f.closeAndWait(t, conn, "zh71fb1")
}

func TestUnknownCtrlRequest(t *testing.T) {
	factory := newTestCA(t)
	f := newMTLSFixture(t, factory)
	f.addCode(t, "code")
	code, conn, err := f.register(factory.device(t, "9483c4b71fb1", clientAuthEKU), "zh71fb1", "9483c4b71fb1", "code")
	if err != nil || code != 0 {
		t.Fatalf("join: (%d, %v)", code, err)
	}

	if r := ctrlCall(t, conn, "d7", "no.such.thing"); r.OK || r.Err == nil || r.Err.Code != "unknown_cmd" {
		t.Errorf("unknown request answered %+v, want unknown_cmd", r)
	}
	if f.srv.GetDevice("", "zh71fb1") == nil {
		t.Error("an unknown request dropped the device")
	}
	f.closeAndWait(t, conn, "zh71fb1")
}
