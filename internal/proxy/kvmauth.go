package proxy

import (
    "net"
    "net/http"
    "strings"
)

// KVM device auth pass-through.
//
// GL KVM firmware (gl-inet/glkvm, a kvmd fork) authorizes each request by
// checking, in order: X-KVMD-User/X-KVMD-Passwd headers, the Token header,
// the auth_token query parameter, the auth_token cookie, then HTTP basic
// auth (kvmd/apps/kvmd/api/auth.py, check_request_auth). Header auth is
// tried first, so headers injected here take effect even when the browser
// holds a stale or missing auth_token cookie.
//
// When device two-factor auth is on, kvmd expects the current 6-digit TOTP
// code appended to the password (kvmd/apps/kvmd/auth.py, authorize).

const (
    HeaderKVMDUser   = "X-KVMD-User"
    HeaderKVMDPasswd = "X-KVMD-Passwd"

    // rttyGoClient is the client type reported by plain Linux hosts running
    // rtty-go. Those are not KVMs and never get KVM credentials.
    rttyGoClient = "rtty-go"
)

// KVMAuth injects kvmd credentials into requests tunneled to a KVM's own
// web UI. A nil *KVMAuth injects nothing.
type KVMAuth struct {
    user     string
    password string
    // totpCode, when non-nil, returns the current TOTP code to append to
    // the password on every request.
    totpCode func() (string, error)
}

// NewKVMAuth returns a KVMAuth, or nil when password is empty (feature off).
// totpCode may be nil when device two-factor auth is not used.
func NewKVMAuth(user, password string, totpCode func() (string, error)) *KVMAuth {
    if password == "" {
        return nil
    }
    if strings.TrimSpace(user) == "" {
        user = "admin"
    }
    return &KVMAuth{user: strings.TrimSpace(user), password: password, totpCode: totpCode}
}

// Apply sets the kvmd auth headers on h, replacing any client-supplied
// values. If a TOTP code cannot be produced, both headers are removed and
// the error is returned, so the device falls back to its own login page
// instead of rejecting a half-built credential.
func (a *KVMAuth) Apply(h http.Header) error {
    if a == nil || h == nil {
        return nil
    }

    passwd := a.password
    if a.totpCode != nil {
        code, err := a.totpCode()
        if err != nil {
            StripKVMAuth(h)
            return err
        }
        passwd += code
    }

    h.Set(HeaderKVMDUser, a.user)
    h.Set(HeaderKVMDPasswd, passwd)
    return nil
}

// StripKVMAuth removes kvmd auth headers from h.
func StripKVMAuth(h http.Header) {
    if h == nil {
        return
    }
    h.Del(HeaderKVMDUser)
    h.Del(HeaderKVMDPasswd)
}

// IsKVMRemoteControlTarget reports whether a /web proxy session targets the
// KVM's own web UI: a non rtty-go device, over https, to exactly
// 127.0.0.1:443 on the device. This is what the Remote Control button opens.
//
// The addr comes from the request URL, and "Access Device Web" lets users
// pick arbitrary hosts and ports on the device's network, so credentials
// must never be attached to any other target.
func IsKVMRemoteControlTarget(clientType, proto, addr string) bool {
    if clientType == rttyGoClient {
        return false
    }
    if proto != "https" {
        return false
    }

    host, port, err := net.SplitHostPort(strings.TrimSpace(addr))
    if err != nil || port != "443" {
        return false
    }
    ip := net.ParseIP(host)
    if ip == nil {
        return false
    }
    ip4 := ip.To4()
    return ip4 != nil && ip4.Equal(net.IPv4(127, 0, 0, 1))
}
