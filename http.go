/*
 * MIT License
 *
 * Copyright (c) 2019 Jianhui Zhao <zhaojh329@gmail.com>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package main

import (
    "bufio"
    "context"
    "crypto/tls"
    "encoding/binary"
    "errors"
    "fmt"
    "net"
    "net/http"
    "net/url"
    "strconv"
    "strings"
    "sync"
    "sync/atomic"
    "time"

    "rttys/utils"

    "github.com/gin-gonic/gin"
    "github.com/rs/zerolog/log"
    "github.com/valyala/bytebufferpool"
)

type HttpProxySession struct {
    expire   atomic.Int64
    ctx      context.Context
    cancel   context.CancelFunc
    devid    string
    group    string
    destaddr string
    https    bool
}

var httpProxySessions = sync.Map{}

const httpProxySessionsExpire = 15 * time.Minute

func (ses *HttpProxySession) Expire() {
    ses.expire.Store(time.Now().Add(httpProxySessionsExpire).Unix())
}

func (ses *HttpProxySession) String() string {
    return fmt.Sprintf("{devid: %s, group: %s, destaddr: %s, https: %v}",
        ses.devid, ses.group, ses.destaddr, ses.https)
}

func (srv *RttyServer) ListenHttpProxy() {
    cfg := &srv.cfg

    if cfg.AddrHttpProxy != "" {
        addr, err := net.ResolveTCPAddr("tcp", cfg.AddrHttpProxy)
        if err != nil {
            log.Warn().Msg("invalid http proxy addr: " + err.Error())
        } else {
            srv.httpProxyPort = addr.Port
        }
    }

    ln, err := net.Listen("tcp", cfg.AddrHttpProxy)
    if err != nil {
        log.Fatal().Msg(err.Error())
    }
    defer ln.Close()

    // In reverse proxy mode (TLS terminated by nginx), never enable TLS here.
    enableTLS := !cfg.ReverseProxyEnabled && cfg.SslCert != "" && cfg.SslKey != ""
    if enableTLS {
        crt, err := tls.LoadX509KeyPair(cfg.SslCert, cfg.SslKey)
        if err != nil {
            log.Fatal().Msg(err.Error())
        }

        tlsConfig := &tls.Config{Certificates: []tls.Certificate{crt}}

        ln = tls.NewListener(ln, tlsConfig)
    }

    srv.httpProxyPort = ln.Addr().(*net.TCPAddr).Port

    log.Info().Msgf("Listen http proxy on: %s", ln.Addr().(*net.TCPAddr))

    go httpProxySessionsClean()

    for {
        c, err := ln.Accept()
        if err != nil {
            log.Error().Msg(err.Error())
            continue
        }

        go doHttpProxy(srv, c)
    }
}

func httpProxySessionsClean() {
    for {
        time.Sleep(time.Second * 30)

        httpProxySessions.Range(func(key, value any) bool {
            ses := value.(*HttpProxySession)
            if time.Now().Unix() > ses.expire.Load() {
                log.Debug().Msgf("Http proxy session '%s' expired", key)
                ses.cancel()
                httpProxySessions.Delete(key)
            }
            return true
        })
    }
}

func doHttpProxy(srv *RttyServer, c net.Conn) {
    defer logPanic()
    defer c.Close()

    br := bufio.NewReader(c)

    req, err := http.ReadRequest(br)
    if err != nil {
        return
    }

    domain, port, proto := getRequestHostInfo(req)
    log.Debug().Msgf("http proxy incoming host=%s port=%s proto=%s uri=%s",
        domain, port, proto, req.URL.String())
    devID, ok := extractDeviceIDFromHost(domain)
    if ok {
        log.Debug().Msgf("parsed deviceId from host: %s", devID)
    } else {
        log.Debug().Msgf("host is IP or invalid, skip deviceId parsing")
    }

    // 获取 URL 查询参数
    queryParams := req.URL.Query()
    name := queryParams.Get("sid")
    if name != "" {
        location := "/"
        location += fmt.Sprintf("?_=%d", time.Now().Unix())

        Write302WithCookie(c, location, "rtty-http-sid", name)
        return
    }

    cookie, err := req.Cookie("rtty-http-sid")
    if err != nil {
        log.Debug().Msgf(`not found cookie "rtty-http-sid"`)
        sendHTTPErrorResponse(c, "invalid")
        return
    }
    sid := cookie.Value

    sesVal, ok := httpProxySessions.Load(sid)
    if !ok {
        log.Debug().Msgf(`not found httpProxySession "%s"`, sid)
        sendHTTPErrorResponse(c, "unauthorized")
        return
    }

    ses := sesVal.(*HttpProxySession)

    dev := srv.GetDevice(ses.group, ses.devid)
    if dev == nil {
        log.Debug().Msgf(`device "%s" group "%s" offline`, ses.devid, ses.group)
        sendHTTPErrorResponse(c, "offline")
        return
    }

    // 3) match hostDevID vs session devid, and optionally lookup by hostDevID
    if devID != "" {
        match := devID == ses.devid
        log.Debug().Msgf(
            "http proxy devid check: hostDevID=%s sessionDevid=%s match=%v hostDevFound=%v sid=%s group=%s",
            devID, ses.devid, match, domain, sid, ses.group,
        )

        // If you want, you can also log when mismatch happens
        if !match {
            log.Info().Msgf(
                "http proxy devid mismatch: hostDevID=%s sessionDevid=%s sid=%s group=%s host=%s uri=%s",
                devID, ses.devid, sid, ses.group, domain, req.URL.String(),
            )
            sendHTTPErrorResponse(c, "invalid")
        }
    } else {
        log.Debug().Msgf(
            "http proxy devid check skipped: no hostDevID (host=%s) sid=%s group=%s sessionDevid=%s",
            domain, sid, ses.group, ses.devid,
        )
    }

    hostHeaderRewrite := ses.destaddr

    destAddr := genDestAddr(hostHeaderRewrite)
    srcAddr := tcpAddr2Bytes(c.RemoteAddr().(*net.TCPAddr))

    ctx, cancel := context.WithCancel(ses.ctx)
    defer cancel()

    go func() {
        <-ctx.Done()
        c.Close()
        log.Debug().Msgf("http proxy conn closed: %s", ses)
        dev.https.Delete(string(srcAddr))
        sendHttpReq(dev, ses.https, srcAddr[:], destAddr, nil)
    }()

    log.Debug().Msgf("new http proxy conn: %s", ses)

    dev.https.Store(string(srcAddr), c)

    hpw := &HttpProxyWriter{destAddr, srcAddr, hostHeaderRewrite, dev, ses.https}

    req.Host = hostHeaderRewrite
    hpw.WriteRequest(req)

    if req.Header.Get("Upgrade") == "websocket" {
        b := make([]byte, 4096)

        for {
            n, err := c.Read(b)
            if err != nil {
                return
            }
            sendHttpReq(dev, ses.https, srcAddr, destAddr, b[:n])
            ses.Expire()
        }
    } else {
        for {
            req, err := http.ReadRequest(br)
            if err != nil {
                return
            }
            hpw.WriteRequest(req)
            ses.Expire()
        }
    }
}

func httpProxyRedirect(srv *RttyServer, c *gin.Context, group string) {
    cfg := &srv.cfg
    devid := c.Param("devid")
    proto := c.Param("proto")
    addr := c.Param("addr")
    rawPath := c.Param("path")
    log.Info().Msgf("httpProxyRedirect devid: %s, proto: %s, addr: %s, path: %s", devid, proto, addr, rawPath)

    if !callUserHookUrl(cfg, c) {
        c.Status(http.StatusForbidden)
        return
    }

    log.Debug().Msgf("httpProxyRedirect devid: %s, proto: %s, addr: %s, path: %s", devid, proto, addr, rawPath)

    _, _, err := httpProxyVaildAddr(addr)
    if err != nil {
        log.Debug().Msgf("invalid addr: %s", addr)
        c.Status(http.StatusBadRequest)
        return
    }

    path, err := url.Parse(rawPath)
    if err != nil {
        log.Debug().Msgf("invalid path: %s", rawPath)
        c.Status(http.StatusBadRequest)
        return
    }

    dev := srv.GetDevice(group, devid)
    if dev == nil {
        c.Redirect(http.StatusFound, "/error/offline")
        return
    }

    location := c.Request.Header.Get("HttpProxyRedir")
    log.Info().Msgf("HttpProxyRedir location: %s, devid: %s", location, devid)
    if location == "" {
        location = cfg.HttpProxyRedirURL
        if location != "" {
            log.Debug().Msgf("use HttpProxyRedirURL from config: %s, devid: %s", location, devid)
        }
    } else {
        log.Debug().Msgf("use HttpProxyRedir from HTTP header: %s, devid: %s", location, devid)
    }

    if location == "" {
        host, _, err := net.SplitHostPort(c.Request.Host)
        if err != nil {
            host = c.Request.Host
        }

        location = "http://" + host

        if srv.httpProxyPort != 80 {
            location += fmt.Sprintf(":%d", srv.httpProxyPort)
        }
    }

    location += path.Path

    location += fmt.Sprintf("?_=%d", time.Now().Unix())

    if path.RawQuery != "" {
        location += "&" + path.RawQuery
    }

    sid, err := c.Cookie("rtty-http-sid")
    log.Info().Msgf("rtty-http-sid: %s", sid)
    if err == nil {
        if v, loaded := httpProxySessions.LoadAndDelete(sid); loaded {
            s := v.(*HttpProxySession)
            s.cancel()
            log.Debug().Msgf(`del old httpProxySession "%s" for device "%s"`, sid, devid)
        }
    }

    sid = utils.GenUniqueID()
    log.Info().Msgf("rtty-http-sid: %s", sid)
    ctx, cancel := context.WithCancel(dev.ctx)

    ses := &HttpProxySession{
        ctx:      ctx,
        cancel:   cancel,
        devid:    devid,
        group:    group,
        destaddr: addr,
        https:    proto == "https",
    }
    ses.Expire()
    httpProxySessions.Store(sid, ses)

    log.Debug().Msgf(`new httpProxySession "%s" for device "%s"`, sid, devid)

    domain := c.Request.Header.Get("HttpProxyRedirDomain")
    if domain == "" {
        domain = cfg.HttpProxyRedirDomain
        if domain != "" {
            log.Debug().Msgf("set cookie domain from config: %s, devid: %s", domain, devid)
        }
    } else {
        log.Debug().Msgf("set cookie domain from HTTP header: %s, devid: %s", domain, devid)
    }

    // Get domain info
    host := c.Request.Host
    hostname, _, err := net.SplitHostPort(host)
    if err != nil {
        hostname = host
    }
    log.Info().Msgf("hostname: %s", hostname)

    ip := net.ParseIP(hostname)
    isIP := ip != nil
    if isIP {
        location = fmt.Sprintf("https://%s%s?sid=%s", hostname, cfg.AddrHttpProxy, sid)
        log.Info().Msgf("Using IP redirect: %s", location)
    } else {
        redirHost := buildRedirectHost(hostname, devid)
        // Keep original behavior when NOT in reverse proxy mode
        if !cfg.ReverseProxyEnabled {
            location = fmt.Sprintf("https://%s%s?sid=%s", redirHost, cfg.AddrHttpProxy, sid)
            log.Info().Msgf("Using domain redirect: %s", location)
        } else {
            // ---- verify forwarded headers from reverse proxy ----
            rawHost := c.GetHeader("Host")
            xfHost := c.GetHeader("X-Forwarded-Host")
            xfProto := c.GetHeader("X-Forwarded-Proto")
            xfPort := c.GetHeader("X-Forwarded-Port")
            xRealIP := c.GetHeader("X-Real-IP")
            xFF := c.GetHeader("X-Forwarded-For")

            log.Info().Msgf(
                "reverse-proxy info: method=%s uri=%s host=%q tls=%v remoteIP=%q",
                c.Request.Method,
                c.Request.URL.String(),
                rawHost,
                c.Request.TLS != nil,
                c.ClientIP(),
            )
            log.Info().Msgf(
                "reverse-proxy headers: Host=%q X-Forwarded-Host=%q X-Forwarded-Proto=%q X-Forwarded-Port=%q X-Real-IP=%q X-Forwarded-For=%q",
                rawHost, xfHost, xfProto, xfPort, xRealIP, xFF,
            )

            // -------------------------------------------------
            // Proxy mode:
            // 1) If DEVICE_ENDPOINT_HOST is configured, use it directly
            // 2) Otherwise, fallback to forwarded-header logic
            // -------------------------------------------------

            // 0) scheme: follow reverse proxy
            scheme := ""
            if v := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); v != "" {
                scheme = strings.ToLower(strings.Split(v, ",")[0])
            } else if c.Request.TLS != nil {
                scheme = "https"
            } else {
                scheme = "http"
            }

            // [A] Prefer explicit DEVICE_ENDPOINT_HOST if set
            if v := strings.TrimSpace(cfg.DeviceEndpointHost); v != "" {
                endpoint := v // already normalized when reading env: host[:port] only

                baseHost := endpoint
                port := ""
                if h, p, err := net.SplitHostPort(endpoint); err == nil {
                    baseHost = h
                    port = p
                }

                // Build device host: <deviceId>.<baseHost>
                // NOTE: DEVICE_ENDPOINT_HOST is a base domain (host[:port]) for device access,
                baseHost = strings.TrimSuffix(strings.TrimSpace(baseHost), ".")
                deviceHost := devid
                if baseHost != "" {
                    deviceHost = devid + "." + baseHost
                }

                hostPort := joinHostPortIfNeeded(deviceHost, scheme, port)

                redirectPath := c.Request.URL.Path
                location = buildRedirectLocation(scheme, hostPort, redirectPath, sid)
                log.Info().Msgf("Using domain redirect (proxy mode, DEVICE_ENDPOINT_HOST): %s", location)
            } else {
                // 1) external port: prefer the one user actually accessed
                port := ""
                if fp := strings.TrimSpace(c.GetHeader("X-Forwarded-Port")); fp != "" {
                    port = strings.TrimSpace(strings.Split(fp, ",")[0])
                } else if fh := strings.TrimSpace(c.GetHeader("X-Forwarded-Host")); fh != "" {
                    fh = strings.TrimSpace(strings.Split(fh, ",")[0])
                    if _, p, err := net.SplitHostPort(fh); err == nil && p != "" {
                        port = p
                    }
                }
                log.Info().Msgf("port: %s", port)

                // 3) Build host: in proxy mode redirect domain to be redirHost
                hostPort := joinHostPortIfNeeded(redirHost, scheme, port)

                redirectPath := c.Request.URL.Path
                location = buildRedirectLocation(scheme, hostPort, redirectPath, sid)
                log.Info().Msgf("Using domain redirect (proxy mode): %s", location)
            }
        }
    }

    log.Info().Msgf("Final redirect location: %s", location)
    c.Redirect(http.StatusFound, location)
}

func sendHttpReq(dev *Device, https bool, srcAddr []byte, destAddr []byte, data []byte) {
    bb := bytebufferpool.Get()
    defer bytebufferpool.Put(bb)

    if dev.proto > 3 {
        if https {
            bb.WriteByte(1)
        } else {
            bb.WriteByte(0)
        }
    }

    bb.Write(srcAddr)
    bb.Write(destAddr)
    bb.Write(data)

    dev.WriteMsg(msgTypeHttp, "", bb.Bytes())
}

func genDestAddr(addr string) []byte {
    destIP, destPort, err := httpProxyVaildAddr(addr)
    if err != nil {
        return nil
    }

    b := make([]byte, 6)
    copy(b, destIP)

    binary.BigEndian.PutUint16(b[4:], destPort)

    return b
}

func tcpAddr2Bytes(addr *net.TCPAddr) []byte {
    b := make([]byte, 18)

    binary.BigEndian.PutUint16(b[:2], uint16(addr.Port))

    copy(b[2:], addr.IP)

    return b
}

func httpProxyVaildAddr(addr string) (net.IP, uint16, error) {
    ips, ports, err := net.SplitHostPort(addr)
    if err != nil {
        ips = addr
        ports = "80"
    }

    ip := net.ParseIP(ips)
    if ip == nil {
        return nil, 0, errors.New("invalid IPv4 Addr")
    }

    ip = ip.To4()
    if ip == nil {
        return nil, 0, errors.New("invalid IPv4 Addr")
    }

    port, _ := strconv.Atoi(ports)

    return ip, uint16(port), nil
}

type HttpProxyWriter struct {
    destAddr          []byte
    srcAddr           []byte
    hostHeaderRewrite string
    dev               *Device
    https             bool
}

func (rw *HttpProxyWriter) Write(p []byte) (n int, err error) {
    sendHttpReq(rw.dev, rw.https, rw.srcAddr, rw.destAddr, p)
    return len(p), nil
}

func (rw *HttpProxyWriter) WriteRequest(req *http.Request) {
    req.Host = rw.hostHeaderRewrite
    req.Write(rw)
}

func generateErrorHTML(errorType string) string {
    return fmt.Sprintf(
        `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>RTTY</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            background-color: #555;
            line-height: 1.6;
        }

        .error-container {
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            min-height: 60vh;
            text-align: center;
        }

        .error-icon {
            margin-bottom: 2rem;
            animation: fadeIn 0.8s ease-in-out;
        }

        .error-icon svg {
            width: 90px;
            height: 90px;
            fill: #f56565;
        }

        .error-content {
            max-width: 700px;
            animation: slideUp 0.8s ease-out 0.2s both;
        }

        .error-title {
            font-size: 1.8rem;
            font-weight: 600;
            color: #7a8fb0;
            margin-bottom: 1rem;
            line-height: 1.2;
        }

        .error-message {
            font-size: 1rem;
            color: #b6c1d3;
            margin-bottom: 2rem;
            line-height: 1.6;
            text-align: left;
        }

        @keyframes fadeIn {
            from {
                opacity: 0;
                transform: scale(0.8);
            }
            to {
                opacity: 1;
                transform: scale(1);
            }
        }

        @keyframes slideUp {
            from {
                opacity: 0;
                transform: translateY(20px);
            }
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }
    </style>
</head>
<body>
    <div class="error-container">
        <div class="error-icon">
            <svg viewBox="0 0 24 24">
                <path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"/>
            </svg>
        </div>
        <div class="error-content">
            <h2 class="error-title" id="errorTitle"></h2>
            <p class="error-message" id="errorMessage"></p>
        </div>
    </div>

	<script>
        const translations = {
            en: {
                'Device Unavailable': 'Device Unavailable',
				'Invalid Request': 'Invalid Request',
				'Unauthorized Access': 'Unauthorized Access',
                'Device offline message': 'The device is currently offline. Please check the device status and try again.',
				'Invalid request message': 'The request is invalid or malformed',
				'Unauthorized request message': 'You are not authorized to access this resource. Please check your session and try again.'
            },
            'zh-CN': {
                'Device Unavailable': '设备不可用',
				'Invalid Request': '无效请求',
				'Unauthorized Access': '未授权访问',
                'Device offline message': '设备当前离线，请检查设备状态后重试。',
				'Invalid request message': '请求无效或格式错误',
				'Unauthorized request message': '您无权访问此资源。请检查您的会话并重试。'
            }
        };

        function t(key, lang) {
            return translations[lang][key] || translations.en[key] || key;
        }

        function updateContent() {
            const errorType = '%s';
            const lang = navigator.language === 'zh-CN' ? 'zh-CN' : 'en';

            let title = '', message = '';

            switch (errorType) {
			case 'offline':
				title = t('Device Unavailable', lang);
				message = t('Device offline message', lang);
				break;
			case 'invalid':
				title = t('Invalid Request', lang);
				message = t('Invalid request message', lang);
				break;
			case 'unauthorized':
				title = t('Unauthorized Access', lang);
				message = t('Unauthorized request message', lang);
				break;
            }

            document.getElementById('errorTitle').textContent = title;
            document.getElementById('errorMessage').textContent = message;

            // Update page title
            if (title) {
                document.title = title + ' - RTTY';
            } else {
                document.title = 'Error - RTTY';
            }
        }

        // Initialize page on load
        document.addEventListener('DOMContentLoaded', updateContent);
    </script>
</body>
</html>`, errorType)
}

func sendHTTPErrorResponse(conn net.Conn, errorType string) {
    htmlContent := generateErrorHTML(errorType)

    response := "HTTP/1.1 200 OK\r\n"
    response += "Content-Type: text/html; charset=utf-8\r\n"
    response += fmt.Sprintf("Content-Length: %d\r\n", len(htmlContent))
    response += "Connection: close\r\n"
    response += "\r\n"
    response += htmlContent

    conn.Write([]byte(response))
}

func Write302WithCookie(conn net.Conn, location, cookieName, cookieValue string) {
    cookie := fmt.Sprintf("%s=%s; Path=/; HttpOnly", cookieName, cookieValue)
    response := fmt.Sprintf(
        "HTTP/1.1 302 Found\r\n"+
            "Location: %s\r\n"+
            "Set-Cookie: %s\r\n"+
            "Content-Length: 0\r\n"+
            "Connection: close\r\n"+
            "\r\n",
        location, cookie,
    )
    _, _ = conn.Write([]byte(response))
}
