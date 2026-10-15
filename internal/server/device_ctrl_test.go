package server

import (
	"encoding/json"
	"testing"

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
