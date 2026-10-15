package server

import (
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
