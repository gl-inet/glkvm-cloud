package server

import (
    "net/http"

    "rttys/internal/proxy"
)

type HostInfo = proxy.HostInfo

func getHostInfoFromRequest(req *http.Request) HostInfo {
    return proxy.GetHostInfoFromRequest(req)
}

func isIPHost(host string) bool {
    return proxy.IsIPHost(host)
}

func domainAllowed(host, base string) bool {
    return proxy.DomainAllowed(host, base)
}

func buildRedirectHost(hostname, devid string) string {
    return proxy.BuildRedirectHost(hostname, devid)
}

func joinHostPortIfNeeded(host, scheme, port string) string {
    return proxy.JoinHostPortIfNeeded(host, scheme, port)
}

func buildRedirectLocation(scheme, hostPort, path, sid string) string {
    return proxy.BuildRedirectLocation(scheme, hostPort, path, sid)
}

func getRequestHostInfo(req *http.Request) (host string, port string, proto string) {
    return proxy.GetRequestHostInfo(req)
}

func extractDeviceIDFromHost(host string) (string, bool) {
    return proxy.ExtractDeviceIDFromHost(host)
}
