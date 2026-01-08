package main

import (
    "net"
    "net/http"
    "net/url"
    "strings"
)

type HostInfo struct {
    Host    string // pure host without port
    Port    string // external port if known
    Scheme  string // http/https
    RawHost string // req.Host (may include port)
    XFHost  string // X-Forwarded-Host (raw)
    XFProto string // X-Forwarded-Proto (raw)
    XFPort  string // X-Forwarded-Port (raw)
}

func getHostInfoFromRequest(req *http.Request) HostInfo {
    hi := HostInfo{
        RawHost: req.Host,
        XFHost:  req.Header.Get("X-Forwarded-Host"),
        XFProto: req.Header.Get("X-Forwarded-Proto"),
        XFPort:  req.Header.Get("X-Forwarded-Port"),
    }

    // host: prefer X-Forwarded-Host
    host := strings.TrimSpace(hi.XFHost)
    if host != "" {
        host = strings.TrimSpace(strings.Split(host, ",")[0])
    } else {
        host = strings.TrimSpace(req.Host)
    }

    // split port if host contains it
    if h, p, err := net.SplitHostPort(host); err == nil {
        hi.Host = h
        hi.Port = p
    } else {
        hi.Host = strings.TrimSuffix(host, ".")
    }

    // scheme
    proto := strings.TrimSpace(hi.XFProto)
    if proto != "" {
        proto = strings.ToLower(strings.TrimSpace(strings.Split(proto, ",")[0]))
        hi.Scheme = proto
    } else if req.TLS != nil {
        hi.Scheme = "https"
    } else {
        hi.Scheme = "http"
    }

    // forwarded port overrides
    fp := strings.TrimSpace(hi.XFPort)
    if fp != "" {
        hi.Port = strings.TrimSpace(strings.Split(fp, ",")[0])
    }

    return hi
}

// isIPHost checks whether host is an IP address.
func isIPHost(host string) bool {
    ip := net.ParseIP(strings.TrimSpace(host))
    return ip != nil
}

// domainAllowed checks whether host is allowed.
// Allow:
// - exact match: base
// - subdomain: *.base
func domainAllowed(host, base string) bool {
    host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
    base = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(base), "."))

    if host == "" || base == "" {
        return false
    }
    if host == base {
        return true
    }
    return strings.HasSuffix(host, "."+base)
}

// buildRedirectHost removes the first label of the hostname and prepends devid.
// Rules:
// - "www.example.com"         -> "devid.example.com"
// - "www.l1.example.com"      -> "devid.l1.example.com"
// - "www.l1.l2.example.com"   -> "devid.l1.l2.example.com"
// - Two-level domain "example.com" -> "devid.example.com"
// - Single label / abnormal cases   -> "devid." + hostname (fallback)
//
// The input hostname must be a pure hostname without port.
func buildRedirectHost(hostname, devid string) string {
    // Allow FQDN with trailing dot like "example.com."
    hostname = strings.TrimSuffix(hostname, ".")

    // Split into labels
    labels := strings.Split(hostname, ".")
    // Remove empty labels (in case of consecutive dots)
    compact := make([]string, 0, len(labels))
    for _, l := range labels {
        if l != "" {
            compact = append(compact, l)
        }
    }
    labels = compact

    switch len(labels) {
    case 0:
        return devid // extreme case: just return devid
    case 1:
        // Single label (e.g., "localhost") — keep original as suffix
        return devid + "." + labels[0]
    default:
        // >=2: drop the leftmost label
        suffix := strings.Join(labels[1:], ".")
        return devid + "." + suffix
    }
}

func joinHostPortIfNeeded(host, scheme, port string) string {
    if port == "" {
        return host
    }
    // avoid adding default ports
    if (scheme == "https" && port == "443") || (scheme == "http" && port == "80") {
        return host
    }
    return net.JoinHostPort(host, port)
}

func buildRedirectLocation(scheme, hostPort, path, sid string) string {
    if path == "" {
        path = "/"
    }
    u := &url.URL{
        Scheme: scheme,
        Host:   hostPort,
        Path:   path,
    }
    q := u.Query()
    q.Set("sid", sid)
    u.RawQuery = q.Encode()
    return u.String()
}

// getRequestHostInfo extracts domain(host), port and scheme(proto) from request headers.
// Priority:
// 1) X-Forwarded-Host / X-Forwarded-Proto / X-Forwarded-Port (reverse proxy)
// 2) Host header / TLS info
func getRequestHostInfo(req *http.Request) (host string, port string, proto string) {
    // 1) Reverse-proxy headers
    xfh := strings.TrimSpace(req.Header.Get("X-Forwarded-Host"))
    xfp := strings.TrimSpace(req.Header.Get("X-Forwarded-Proto"))
    xfport := strings.TrimSpace(req.Header.Get("X-Forwarded-Port"))

    // X-Forwarded-Host may contain a comma-separated list. Take the first one.
    if xfh != "" {
        if i := strings.IndexByte(xfh, ','); i >= 0 {
            xfh = strings.TrimSpace(xfh[:i])
        }
        host = xfh
    }

    // 2) Fallback to Host header
    if host == "" {
        host = strings.TrimSpace(req.Host)
    }

    // Split host:port if present
    if h, p, err := net.SplitHostPort(host); err == nil {
        host = h
        port = p
    } else {
        // no explicit port in Host header
        port = ""
    }

    // scheme/proto
    if xfp != "" {
        if i := strings.IndexByte(xfp, ','); i >= 0 {
            xfp = strings.TrimSpace(xfp[:i])
        }
        proto = xfp
    } else if req.TLS != nil {
        proto = "https"
    } else {
        proto = "http"
    }

    // forwarded port overrides parsed port if present
    if xfport != "" {
        if i := strings.IndexByte(xfport, ','); i >= 0 {
            xfport = strings.TrimSpace(xfport[:i])
        }
        port = xfport
    }

    return host, port, proto
}

// extractDeviceIDFromHost extracts deviceId from hostname.
// Rules:
// - IP address        -> ("", false)
// - lv99862.example.com        -> ("lv99862", true)
// - lv99862.l1.example.com     -> ("lv99862", true)
// - localhost / single label   -> ("localhost", true)
func extractDeviceIDFromHost(host string) (string, bool) {
    host = strings.TrimSpace(host)
    if host == "" {
        return "", false
    }

    // remove trailing dot
    host = strings.TrimSuffix(host, ".")

    // If host is IP, skip
    if ip := net.ParseIP(host); ip != nil {
        return "", false
    }

    labels := strings.Split(host, ".")
    for _, l := range labels {
        if l != "" {
            return l, true
        }
    }
    return "", false
}
