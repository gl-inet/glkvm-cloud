package server

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"rttys/xconfig"

	"github.com/gin-gonic/gin"
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

// sendToProxy opens a connection to the HTTP proxy port of srv and sends raw
// on it. It goes over TCP like the real port: the proxy keys browser
// connections by their TCP address.
func sendToProxy(t *testing.T, srv *RttyServer, raw string) net.Conn {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		if c, err := ln.Accept(); err == nil {
			doHttpProxy(srv, c)
		}
	}()
	client, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	if _, err := client.Write([]byte(raw)); err != nil {
		t.Fatal(err)
	}
	return client
}

// proxyOnce sends one raw request to the HTTP proxy port of srv and returns
// the reply.
func proxyOnce(t *testing.T, srv *RttyServer, raw string) (*http.Response, string) {
	t.Helper()
	resp, err := http.ReadResponse(bufio.NewReader(sendToProxy(t, srv, raw)), nil)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	return resp, string(body)
}

// fakeDevice registers devid on srv with one proxy session, and returns what
// the platform sends the device within a short wait.
func fakeDevice(t *testing.T, srv *RttyServer, devid string) <-chan []byte {
	t.Helper()
	deviceEnd, platformEnd := net.Pipe()
	t.Cleanup(func() { deviceEnd.Close() })
	srv.GetGroup("", true).devices.Store(devid, &Device{conn: platformEnd})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	httpProxySessions.Store(testProxySID, &HttpProxySession{ctx: ctx, cancel: cancel, devid: devid, destaddr: "127.0.0.1:80"})
	t.Cleanup(func() { httpProxySessions.Delete(testProxySID) })

	got := make(chan []byte, 1)
	go func() {
		deviceEnd.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		b := make([]byte, 4096)
		n, _ := deviceEnd.Read(b)
		got <- b[:n]
	}()
	return got
}

func TestProxyRefusesAnotherDevicesSession(t *testing.T) {
	discardLogFile()
	// A session for devb presented as deva's: by host, and by path.
	for name, header := range map[string]string{
		"per-device host": "Host: deva.example.com\r\n",
		"path":            "Host: app.example.com\r\nX-KVM-Access-ID: deva\r\n",
	} {
		t.Run(name, func(t *testing.T) {
			srv := &RttyServer{}
			got := fakeDevice(t, srv, "devb")
			_, body := proxyOnce(t, srv, "GET / HTTP/1.1\r\n"+header+"Cookie: rtty-http-sid="+testProxySID+"\r\n\r\n")
			if body != generateErrorHTML("invalid") {
				t.Errorf("reply is not the invalid page: %.80q", body)
			}
			if b := <-got; len(b) != 0 {
				t.Errorf("the request still reached the device (%d bytes)", len(b))
			}
		})
	}
}

func TestProxyPathAccessCookies(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()
	resp := enterProxy(t, server, client, testProxySID, "X-Forwarded-Proto: https\r\nX-KVM-Access-ID: zh71fb1\r\n")
	if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != "/access/zh71fb1/" {
		t.Fatalf("reply = %d to %q, want 302 to /access/zh71fb1/", resp.StatusCode, resp.Header.Get("Location"))
	}
	for _, c := range resp.Cookies() {
		if c.Path != "/access/zh71fb1" {
			t.Errorf("%s: Path=%q, want the device's own /access/zh71fb1", c.Name, c.Path)
		}
	}
	if len(resp.Cookies()) != 2 {
		t.Errorf("cookies = %v, want rtty-http-sid and %s", resp.Cookies(), cloudSessionCookie)
	}

	// The ID goes into a cookie path: nothing but letters and digits.
	server, client = net.Pipe()
	defer client.Close()
	resp = enterProxy(t, server, client, testProxySID, "X-KVM-Access-ID: zh71fb1; Path=/\r\n")
	if resp.StatusCode == http.StatusFound || len(resp.Cookies()) != 0 {
		t.Errorf("crafted ID: reply %d with cookies %v, want an error page and nothing set", resp.StatusCode, resp.Cookies())
	}
}

func TestProxyPathAccessKeepsPlatformCookies(t *testing.T) {
	discardLogFile()
	srv := &RttyServer{}
	got := fakeDevice(t, srv, "zh71fb1")
	sendToProxy(t, srv, "GET /api/info HTTP/1.1\r\nHost: app.example.com\r\nX-KVM-Access-ID: zh71fb1\r\nX-KVM-Access-Style: path\r\n"+
		"Cookie: sid=platform; auth_token=device-own; td=trusted; rtty-http-sid="+testProxySID+"\r\n\r\n")
	b := string(<-got)
	if !strings.Contains(b, "Cookie: auth_token=device-own\r\n") {
		t.Errorf("the device's own cookie did not come through alone: %q", b)
	}
	if strings.Contains(b, "platform") || strings.Contains(b, "trusted") || strings.Contains(strings.ToLower(b), "x-kvm-access") {
		t.Errorf("the platform's cookies or routing headers reached the device: %q", b)
	}
}

func TestPathAccessLocation(t *testing.T) {
	cases := []struct {
		name   string
		host   string
		header map[string]string
		want   string
	}{
		{"behind nginx on 443", "127.0.0.1:1443", map[string]string{"X-Forwarded-Host": "app.example.com", "X-Forwarded-Proto": "https", "X-Forwarded-Port": "443"},
			"https://app.example.com/access/zh71fb1/?rttysid=" + testProxySID},
		{"forwarded host with its port", "127.0.0.1:1443", map[string]string{"X-Forwarded-Host": "app.example.com:8443", "X-Forwarded-Proto": "https"},
			"https://app.example.com:8443/access/zh71fb1/?rttysid=" + testProxySID},
		{"port only in X-Forwarded-Port", "127.0.0.1:1443", map[string]string{"X-Forwarded-Host": "app.example.com", "X-Forwarded-Proto": "https", "X-Forwarded-Port": "8443"},
			"https://app.example.com:8443/access/zh71fb1/?rttysid=" + testProxySID},
		{"no forwarded headers", "app.example.com", nil,
			"http://app.example.com/access/zh71fb1/?rttysid=" + testProxySID},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "/web/zh71fb1/http/127.0.0.1%3A80/", nil)
		req.Host = tc.host
		for k, v := range tc.header {
			req.Header.Set(k, v)
		}
		if got := pathAccessLocation(req, "zh71fb1", "/", testProxySID); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
	}
}

// The style is the entry nginx's call, but only devices that joined with an
// access code get the path; the others keep their per-device host.
func TestProxyRedirectStyle(t *testing.T) {
	discardLogFile()
	gin.SetMode(gin.TestMode)
	for name, tc := range map[string]struct {
		certMAC  string
		wantPath bool
	}{
		"joined with an access code": {"9483c4b71fb1", true},
		"old firmware, token":        {"", false},
	} {
		t.Run(name, func(t *testing.T) {
			srv := &RttyServer{cfg: xconfig.Config{AddrHttpProxy: ":10443", ReverseProxyEnabled: true}}
			srv.GetGroup("", true).devices.Store("zh71fb1", &Device{ctx: context.Background(), certMAC: tc.certMAC})

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/web/zh71fb1/http/127.0.0.1%3A80/", nil)
			c.Request.Host = "app.example.com"
			c.Request.Header.Set("X-Forwarded-Proto", "https")
			c.Request.Header.Set(accessStyleHeader, "path")
			c.Params = gin.Params{{Key: "devid", Value: "zh71fb1"}, {Key: "proto", Value: "http"}, {Key: "addr", Value: "127.0.0.1:80"}, {Key: "path", Value: "/"}}
			httpProxyRedirect(srv, c, "")

			loc, err := url.Parse(w.Header().Get("Location"))
			if err != nil || w.Code != http.StatusFound {
				t.Fatalf("reply %d to %q", w.Code, w.Header().Get("Location"))
			}
			httpProxySessions.Delete(loc.Query().Get("rttysid"))
			if isPath := loc.Host == "app.example.com" && loc.Path == "/access/zh71fb1/"; isPath != tc.wantPath {
				t.Errorf("redirect to %s, want path access: %v", loc, tc.wantPath)
			}
		})
	}
}
