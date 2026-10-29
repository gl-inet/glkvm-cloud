package server

import (
	"strings"
	"testing"

	"rttys/xconfig"
)

func TestValidateConfig(t *testing.T) {
	cases := []struct {
		name string
		cfg  xconfig.Config
		ok   bool
	}{
		{"password and token", xconfig.Config{Password: "p", Token: "t", LegacyTokenAccess: true}, true},
		{"no password", xconfig.Config{Token: "t", LegacyTokenAccess: true}, false},
		{"legacy access without a token", xconfig.Config{Password: "p", LegacyTokenAccess: true}, false},
		{"no token, legacy access off", xconfig.Config{Password: "p"}, true},
	}
	for _, tc := range cases {
		if err := validateConfig(&tc.cfg); (err == nil) != tc.ok {
			t.Errorf("%s: err = %v, want ok=%v", tc.name, err, tc.ok)
		}
	}
}

func TestPublicSecrets(t *testing.T) {
	examples := xconfig.Config{Password: "StrongP@ssw0rd", Token: "DeviceTokenYouCanChangeMe", WebrtcPassword: "AnotherS3cret", LegacyTokenAccess: true}
	names := func(cfg xconfig.Config) string {
		var out []string
		for _, s := range publicSecrets(&cfg) {
			out = append(out, s.setting)
		}
		return strings.Join(out, ",")
	}
	if got := names(examples); got != "RTTYS_PASS,RTTYS_TOKEN,TURN_PASS" {
		t.Errorf("all example values: %q", got)
	}
	noLegacy := examples
	noLegacy.LegacyTokenAccess = false
	if got := names(noLegacy); got != "RTTYS_PASS,TURN_PASS" {
		t.Errorf("legacy token access off, the token is unused: %q", got)
	}
	if got := names(xconfig.Config{Password: "p", Token: "t", WebrtcPassword: "w", LegacyTokenAccess: true}); got != "" {
		t.Errorf("own values: %q", got)
	}
}
