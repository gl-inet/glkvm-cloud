package server

import (
	"strings"
	"testing"

	"rttys/xconfig"
)

func legacyCfg(t *testing.T) xconfig.Config {
	t.Helper()
	discardLogFile()
	cfg := xconfig.Config{AddrDev: ":5912", Token: "device-token", LegacyTokenAccess: true}
	if err := setupPKI(&cfg, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestLegacyScriptAddress(t *testing.T) {
	for _, tc := range []struct {
		name       string
		accessHost string
		webrtcIP   string
		reqHost    string
		want       string
	}{
		{"access host first", "kvm.example.com", "192.0.2.10", "10.0.0.5:8443", "SERVER='kvm.example.com:5912'"},
		{"then the access IP", "", "192.0.2.10", "10.0.0.5:8443", "SERVER='192.0.2.10:5912'"},
		{"then the web host", "", "", "10.0.0.5:8443", "SERVER='10.0.0.5:5912'"},
	} {
		cfg := legacyCfg(t)
		cfg.DeviceAccessHost, cfg.WebrtcIP = tc.accessHost, tc.webrtcIP
		s, err := legacyScript(&cfg, tc.reqHost)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if !strings.Contains(s.Text, tc.want) {
			t.Errorf("%s: script lacks %s", tc.name, tc.want)
		}
		if !strings.Contains(s.Text, "TOKEN='device-token'") {
			t.Errorf("%s: script lacks the device token", tc.name)
		}
	}
}

func TestLegacyScriptTURN(t *testing.T) {
	cfg := legacyCfg(t)
	cfg.WebrtcIP, cfg.WebrtcPort, cfg.WebrtcUsername, cfg.WebrtcPassword = "192.0.2.10", "3478", "turnuser", "turnpass"
	s, err := legacyScript(&cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"username": "turnuser"`, `"password": "turnpass"`, `"turn:192.0.2.10:3478?transport=udp"`} {
		if !strings.Contains(s.Text, want) {
			t.Errorf("script lacks %s", want)
		}
	}

	cfg.WebrtcPassword = ""
	if s, err = legacyScript(&cfg, ""); err != nil || strings.Contains(s.Text, "turnserver.json") {
		t.Errorf("without a TURN server the script still writes one (err %v)", err)
	}
}

func TestLegacyScriptNeedsLegacyAccess(t *testing.T) {
	cfg := legacyCfg(t)
	cfg.WebrtcIP = "192.0.2.10"
	cfg.LegacyTokenAccess = false
	if _, err := legacyScript(&cfg, ""); err == nil {
		t.Error("script rendered with legacy token access off")
	}
	cfg.LegacyTokenAccess = true
	cfg.WebrtcIP = ""
	if _, err := legacyScript(&cfg, ""); err == nil {
		t.Error("script rendered without any address for devices")
	}
}
