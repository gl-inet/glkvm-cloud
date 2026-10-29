package proxy

import (
	"net/url"
	"strings"
	"testing"
)

func TestBuildAccessPathLocation(t *testing.T) {
	cases := []struct {
		name     string
		scheme   string
		hostPort string
		devid    string
		reqPath  string
		wantPath string
	}{
		{"root path", "https", "app.example.com", "abcd11", "/", "/access/abcd11/"},
		{"empty path", "https", "app.example.com", "abcd11", "", "/access/abcd11/"},
		{"nested path", "https", "app.example.com", "abcd11", "/api/status", "/access/abcd11/api/status"},
		{"path no leading slash", "https", "app.example.com", "abcd11", "api/status", "/access/abcd11/api/status"},
		{"with port", "http", "app.example.com:8080", "fefad53", "/x", "/access/fefad53/x"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildAccessPathLocation(tc.scheme, tc.hostPort, tc.devid, tc.reqPath, "SID123")
			u, err := url.Parse(got)
			if err != nil {
				t.Fatalf("unparseable location %q: %v", got, err)
			}
			if u.Scheme != tc.scheme {
				t.Errorf("scheme=%q want %q (%s)", u.Scheme, tc.scheme, got)
			}
			if u.Host != tc.hostPort {
				t.Errorf("host=%q want %q (%s)", u.Host, tc.hostPort, got)
			}
			if u.Path != tc.wantPath {
				t.Errorf("path=%q want %q (%s)", u.Path, tc.wantPath, got)
			}
			if u.Query().Get("rttysid") != "SID123" {
				t.Errorf("rttysid not carried: %s", got)
			}
			if !strings.HasPrefix(u.Path, "/access/"+tc.devid) {
				t.Errorf("path %q missing /access/%s prefix", u.Path, tc.devid)
			}
		})
	}
}
