package server

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
)

// enterProxy sends the request that opens a session on the HTTP proxy port
// and returns the proxy's reply.
func enterProxy(t *testing.T, server, client net.Conn, header string) *http.Response {
	t.Helper()
	go doHttpProxy(nil, server)
	req := "GET /?rttysid=sid-1 HTTP/1.1\r\nHost: 106.55.158.199:10443\r\n" + header + "\r\n"
	if _, err := client.Write([]byte(req)); err != nil {
		t.Fatal(err)
	}
	br := bufio.NewReader(client)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, br) // until the proxy hangs up, so its TLS close need not wait on us
	return resp
}

func TestProxyMarksCloudSession(t *testing.T) {
	ca := newTestCA(t)
	cert := ca.device(t, "proxy", []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth})

	cases := []struct {
		name   string
		conns  func() (server, client net.Conn)
		header string
		secure bool
	}{
		{"TLS on the proxy port", func() (net.Conn, net.Conn) {
			s, c := net.Pipe()
			return tls.Server(s, &tls.Config{Certificates: []tls.Certificate{cert}}),
				tls.Client(c, &tls.Config{InsecureSkipVerify: true})
		}, "", true},
		{"HTTPS ended by a reverse proxy", func() (net.Conn, net.Conn) { return net.Pipe() },
			"X-Forwarded-Proto: https\r\n", true},
		{"plain HTTP", func() (net.Conn, net.Conn) { return net.Pipe() }, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server, client := tc.conns()
			defer client.Close()
			resp := enterProxy(t, server, client, tc.header)

			if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != "/" {
				t.Fatalf("reply = %d to %q, want 302 to /", resp.StatusCode, resp.Header.Get("Location"))
			}
			got := map[string]*http.Cookie{}
			for _, c := range resp.Cookies() {
				got[c.Name] = c
			}
			if sid := got["rtty-http-sid"]; sid == nil || sid.Value != "sid-1" || !sid.HttpOnly {
				t.Errorf("rtty-http-sid = %+v", sid)
			}
			flag := got[cloudSessionCookie]
			if flag == nil || flag.Value != "selfhost" || flag.HttpOnly || flag.Path != "/" {
				t.Fatalf("%s = %+v, want selfhost readable by the page", cloudSessionCookie, flag)
			}
			wantSameSite := http.SameSiteLaxMode
			if tc.secure {
				wantSameSite = http.SameSiteNoneMode
			}
			if flag.Secure != tc.secure || flag.SameSite != wantSameSite {
				t.Errorf("%s: Secure=%v SameSite=%v, want Secure=%v SameSite=%v",
					cloudSessionCookie, flag.Secure, flag.SameSite, tc.secure, wantSameSite)
			}
		})
	}
}

func TestProxyRequestCarriesCloudSession(t *testing.T) {
	br := bufio.NewReader(strings.NewReader(
		"GET /api/info HTTP/1.1\r\nHost: 106.55.158.199:10443\r\nx-cloud-session-flag: none\r\n\r\n"))
	req, err := http.ReadRequest(br)
	if err != nil {
		t.Fatal(err)
	}
	markProxyRequest(req, "127.0.0.1:80")

	var out strings.Builder
	if err := req.Write(&out); err != nil {
		t.Fatal(err)
	}
	fwd, err := http.ReadRequest(bufio.NewReader(strings.NewReader(out.String())))
	if err != nil {
		t.Fatal(err)
	}
	if fwd.Host != "127.0.0.1:80" {
		t.Errorf("Host = %q, want the device address", fwd.Host)
	}
	if v := fwd.Header.Values(cloudSessionHeader); len(v) != 1 || v[0] != "selfhost" {
		t.Errorf("%s = %q, want only the platform's selfhost", cloudSessionHeader, v)
	}
}
