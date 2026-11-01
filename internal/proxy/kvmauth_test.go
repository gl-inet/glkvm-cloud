package proxy

import (
    "bufio"
    "bytes"
    "errors"
    "net/http"
    "strings"
    "testing"
)

func TestIsKVMRemoteControlTarget(t *testing.T) {
    cases := []struct {
        name       string
        clientType string
        proto      string
        addr       string
        want       bool
    }{
        {"remote control button", "", "https", "127.0.0.1:443", true},
        {"kvm reporting a client name", "glkvm", "https", "127.0.0.1:443", true},
        {"surrounding spaces", "", "https", " 127.0.0.1:443 ", true},
        {"rtty-go linux host", "rtty-go", "https", "127.0.0.1:443", false},
        {"plain http", "", "http", "127.0.0.1:443", false},
        {"other port", "", "https", "127.0.0.1:8443", false},
        {"no port", "", "https", "127.0.0.1", false},
        {"other loopback", "", "https", "127.0.0.2:443", false},
        {"lan host via device", "", "https", "192.168.8.1:443", false},
        {"hostname", "", "https", "localhost:443", false},
        {"empty", "", "https", "", false},
        {"ipv6 loopback", "", "https", "[::1]:443", false},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            got := IsKVMRemoteControlTarget(tc.clientType, tc.proto, tc.addr)
            if got != tc.want {
                t.Fatalf("IsKVMRemoteControlTarget(%q, %q, %q) = %v, want %v",
                    tc.clientType, tc.proto, tc.addr, got, tc.want)
            }
        })
    }
}

func TestNewKVMAuthDisabledWithoutPassword(t *testing.T) {
    if a := NewKVMAuth("admin", "", nil); a != nil {
        t.Fatalf("expected nil KVMAuth for empty password")
    }
    // A nil KVMAuth must be safe to use.
    var a *KVMAuth
    h := http.Header{}
    if err := a.Apply(h); err != nil {
        t.Fatalf("nil Apply returned error: %v", err)
    }
    if len(h) != 0 {
        t.Fatalf("nil Apply modified headers: %v", h)
    }
}

func TestKVMAuthApplyReplacesClientHeaders(t *testing.T) {
    a := NewKVMAuth("", "s3cret pass", nil) // empty user defaults to admin
    h := http.Header{}
    h.Set(HeaderKVMDUser, "mallory")
    h.Add(HeaderKVMDPasswd, "guess1")
    h.Add(HeaderKVMDPasswd, "guess2")

    if err := a.Apply(h); err != nil {
        t.Fatal(err)
    }
    if got := h.Values(HeaderKVMDUser); len(got) != 1 || got[0] != "admin" {
        t.Fatalf("user header = %v, want [admin]", got)
    }
    if got := h.Values(HeaderKVMDPasswd); len(got) != 1 || got[0] != "s3cret pass" {
        t.Fatalf("passwd header = %v, want [s3cret pass]", got)
    }
}

func TestKVMAuthApplyAppendsTOTP(t *testing.T) {
    a := NewKVMAuth("admin", "pw", func() (string, error) { return "123456", nil })
    h := http.Header{}
    if err := a.Apply(h); err != nil {
        t.Fatal(err)
    }
    if got := h.Get(HeaderKVMDPasswd); got != "pw123456" {
        t.Fatalf("passwd header = %q, want pw123456", got)
    }
}

func TestKVMAuthApplyTOTPErrorStripsHeaders(t *testing.T) {
    a := NewKVMAuth("admin", "pw", func() (string, error) { return "", errors.New("bad secret") })
    h := http.Header{}
    h.Set(HeaderKVMDUser, "admin")
    h.Set(HeaderKVMDPasswd, "client-supplied")
    if err := a.Apply(h); err == nil {
        t.Fatal("expected error")
    }
    if h.Get(HeaderKVMDUser) != "" || h.Get(HeaderKVMDPasswd) != "" {
        t.Fatalf("headers not stripped on error: %v", h)
    }
}

// The proxy parses each browser request with http.ReadRequest, mutates it,
// and re-serializes it with req.Write toward the device. Check the injected
// headers survive that round trip for a normal request and for a websocket
// upgrade handshake, and that the body is untouched.
func TestKVMAuthSurvivesRequestRoundTrip(t *testing.T) {
    raws := []string{
        "POST /api/hid/events/send_key?key=Enter HTTP/1.1\r\n" +
            "Host: dev1.example.com\r\n" +
            "Cookie: rtty-http-sid=abc; auth_token=stale\r\n" +
            "Content-Length: 5\r\n\r\nhello",
        "GET /api/ws?stream=0 HTTP/1.1\r\n" +
            "Host: dev1.example.com\r\n" +
            "Upgrade: websocket\r\n" +
            "Connection: Upgrade\r\n" +
            "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n" +
            "Sec-WebSocket-Version: 13\r\n\r\n",
    }

    a := NewKVMAuth("admin", "pw", nil)
    for _, raw := range raws {
        req, err := http.ReadRequest(bufio.NewReader(strings.NewReader(raw)))
        if err != nil {
            t.Fatal(err)
        }
        req.Host = "127.0.0.1:443"
        if err := a.Apply(req.Header); err != nil {
            t.Fatal(err)
        }

        var buf bytes.Buffer
        if err := req.Write(&buf); err != nil {
            t.Fatal(err)
        }

        got, err := http.ReadRequest(bufio.NewReader(&buf))
        if err != nil {
            t.Fatal(err)
        }
        if got.Header.Get(HeaderKVMDUser) != "admin" || got.Header.Get(HeaderKVMDPasswd) != "pw" {
            t.Fatalf("headers lost in round trip for %s %s: %v", got.Method, got.URL, got.Header)
        }
        if got.Host != "127.0.0.1:443" {
            t.Fatalf("host = %q", got.Host)
        }
        if got.Method == http.MethodGet && got.Header.Get("Upgrade") != "websocket" {
            t.Fatalf("websocket upgrade header lost: %v", got.Header)
        }
        if got.Method == http.MethodPost {
            var body bytes.Buffer
            if _, err := body.ReadFrom(got.Body); err != nil || body.String() != "hello" {
                t.Fatalf("body = %q, err = %v", body.String(), err)
            }
        }
    }
}
