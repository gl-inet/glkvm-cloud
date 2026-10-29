package xconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOutdatedBundle(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	// What the v2.9 and v3.0 templates render: only v3.0 has the mTLS port.
	old := write("v29.conf", "password: p\naddr-dev: :5912\n")
	cur := write("v30.conf", "password: p\naddr-dev: :5912\naddr-dev-mtls: :5915\n")
	tmpl := write("rttys.conf.tmpl", "")

	defer func(v string) { BundleTemplate = v }(BundleTemplate)
	cases := []struct {
		name, conf, template string
		want                 bool
	}{
		{"v2.9 bundle", old, tmpl, true},
		{"v3.0 bundle", cur, tmpl, false},
		{"own config, not from a bundle", old, filepath.Join(dir, "absent"), false},
	}
	for _, tc := range cases {
		BundleTemplate = tc.template
		var cfg Config
		if err := cfg.Load(tc.conf); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if cfg.OutdatedBundle != tc.want {
			t.Errorf("%s: OutdatedBundle = %v, want %v", tc.name, cfg.OutdatedBundle, tc.want)
		}
	}
}
