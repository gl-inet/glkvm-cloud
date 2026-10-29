package server

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

const testProxySID = "0123456789abcdef0123456789abcdef"

// enterProxy sends the request that opens session sid on the HTTP proxy port
// and returns the proxy's reply.
func enterProxy(t *testing.T, server, client net.Conn, sid, header string) *http.Response {
	t.Helper()
	go doHttpProxy(nil, server)
	req := "GET /?rttysid=" + sid + " HTTP/1.1\r\nHost: 106.55.158.199:10443\r\n" + header + "\r\n"
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
			resp := enterProxy(t, server, client, testProxySID, tc.header)

			if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != "/" {
				t.Fatalf("reply = %d to %q, want 302 to /", resp.StatusCode, resp.Header.Get("Location"))
			}
			got := map[string]*http.Cookie{}
			for _, c := range resp.Cookies() {
				got[c.Name] = c
			}
			if sid := got["rtty-http-sid"]; sid == nil || sid.Value != testProxySID || !sid.HttpOnly {
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

func TestProxyRejectsCraftedSessionID(t *testing.T) {
	for _, sid := range []string{"x%0d%0aX-Probe:%20injected", "a%3Bb", strings.Repeat("a", 65)} {
		server, client := net.Pipe()
		resp := enterProxy(t, server, client, sid, "")
		client.Close()
		if resp.StatusCode == http.StatusFound || resp.Header.Get("X-Probe") != "" || len(resp.Cookies()) != 0 {
			t.Errorf("rttysid %q: reply %d, X-Probe %q, cookies %v; want an error page and nothing set",
				sid, resp.StatusCode, resp.Header.Get("X-Probe"), resp.Cookies())
		}
	}
}

// forward passes a browser request through markProxyRequest and returns it as
// the device reads it.
func forward(t *testing.T, raw string, sharedHost bool) *http.Request {
	t.Helper()
	req, err := http.ReadRequest(bufio.NewReader(strings.NewReader(raw)))
	if err != nil {
		t.Fatal(err)
	}
	markProxyRequest(req, "127.0.0.1:80", sharedHost)
	var out strings.Builder
	if err := req.Write(&out); err != nil {
		t.Fatal(err)
	}
	fwd, err := http.ReadRequest(bufio.NewReader(strings.NewReader(out.String())))
	if err != nil {
		t.Fatal(err)
	}
	return fwd
}

func TestProxyRequestCarriesCloudSession(t *testing.T) {
	fwd := forward(t, "GET /api/info HTTP/1.1\r\nHost: 106.55.158.199:10443\r\nx-cloud-session-flag: none\r\n\r\n", true)
	if fwd.Host != "127.0.0.1:80" {
		t.Errorf("Host = %q, want the device address", fwd.Host)
	}
	if v := fwd.Header.Values(cloudSessionHeader); len(v) != 1 || v[0] != "selfhost" {
		t.Errorf("%s = %q, want only the platform's selfhost", cloudSessionHeader, v)
	}
	if v := fwd.Header.Values("Cookie"); len(v) != 0 {
		t.Errorf("Cookie = %q, want none added", v)
	}
}

func TestProxyKeepsPlatformCookies(t *testing.T) {
	raw := "GET / HTTP/1.1\r\nHost: 106.55.158.199:10443\r\n" +
		"Cookie: sid=platform; auth_token=dev\"1; rtty-http-sid=" + testProxySID + "\r\n" +
		"Cookie: td=trusted; cloud-session-flag=selfhost\r\n\r\n"
	cases := []struct {
		name       string
		sharedHost bool
		want       string
	}{
		{"IP: the platform's host", true, `auth_token=dev"1; cloud-session-flag=selfhost`},
		{"domain: the device's own host", false, `sid=platform; auth_token=dev"1; td=trusted; cloud-session-flag=selfhost`},
	}
	for _, tc := range cases {
		if got := forward(t, raw, tc.sharedHost).Header.Values("Cookie"); len(got) != 1 || got[0] != tc.want {
			t.Errorf("%s: Cookie = %q, want %q", tc.name, got, tc.want)
		}
	}
	if got := forward(t, "GET / HTTP/1.1\r\nHost: h\r\nCookie: sid=platform\r\n\r\n", true).Header.Values("Cookie"); len(got) != 0 {
		t.Errorf("only platform cookies: Cookie = %q, want the header gone", got)
	}
}

// proxyOnce sends one raw request to the HTTP proxy port of srv and returns
// the reply.
func proxyOnce(t *testing.T, srv *RttyServer, raw string) (*http.Response, string) {
	t.Helper()
	// Over TCP like the real port: the proxy keys browser connections by
	// their TCP address.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		if c, err := ln.Accept(); err == nil {
			doHttpProxy(srv, c)
		}
	}()
	client, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.Write([]byte(raw)); err != nil {
		t.Fatal(err)
	}
	br := bufio.NewReader(client)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	return resp, string(body)
}

func TestProxyRefusesAnotherDevicesSession(t *testing.T) {
	discardLogFile()
	deviceEnd, platformEnd := net.Pipe()
	defer deviceEnd.Close()
	srv := &RttyServer{}
	srv.GetGroup("", true).devices.Store("devb", &Device{conn: platformEnd})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	httpProxySessions.Store(testProxySID, &HttpProxySession{ctx: ctx, cancel: cancel, devid: "devb", destaddr: "127.0.0.1:80"})
	defer httpProxySessions.Delete(testProxySID)

	// What reaches the device, if anything.
	got := make(chan int, 1)
	go func() {
		deviceEnd.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, _ := deviceEnd.Read(make([]byte, 64))
		got <- n
	}()

	// A session for devb presented on deva's host.
	_, body := proxyOnce(t, srv, "GET / HTTP/1.1\r\nHost: deva.example.com\r\nCookie: rtty-http-sid="+testProxySID+"\r\n\r\n")
	if body != generateErrorHTML("invalid") {
		t.Errorf("reply is not the invalid page: %.80q", body)
	}
	if n := <-got; n != 0 {
		t.Errorf("the request still reached the device (%d bytes)", n)
	}
}
