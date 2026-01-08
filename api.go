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
    "crypto/tls"
    "embed"
    "io/fs"
    "net"
    "net/http"
    "path"
    "sort"
    "strings"
    "time"

    "rttys/utils"

    "github.com/fanjindong/go-cache"
    "github.com/gin-contrib/cors"
    "github.com/gin-gonic/gin"
    "github.com/rs/zerolog/log"
)

var httpSessions = cache.NewMemCache(cache.WithClearInterval(time.Minute))

const httpSessionExpire = 30 * time.Minute

//go:embed all:ui/dist
var staticFs embed.FS

func (srv *RttyServer) ListenAPI() error {
    cfg := &srv.cfg

    gin.SetMode(gin.ReleaseMode)

    r := gin.New()
    r.Use(func(c *gin.Context) {
        hi := getHostInfoFromRequest(c.Request)

        host := hi.Host
        allowedHost := cfg.WebUIHost
        // If WebUIHost is configured, enforce host validation
        if allowedHost != "" && !isIPHost(host) {
            if !domainAllowed(host, allowedHost) {
                html := generateErrorHTML("invalid")
                c.Data(http.StatusBadRequest, "text/html; charset=utf-8", []byte(html))
                c.Abort()
                return
            }
        }
        c.Next()
    })

    if cfg.AllowOrigins {
        log.Debug().Msg("Allow all origins")
        r.Use(cors.Default())
    }

    authorized := r.Group("/", func(c *gin.Context) {
        if !cfg.LocalAuth && isLocalRequest(c) {
            return
        }

        if !httpAuth(cfg, c) {
            c.AbortWithStatus(http.StatusUnauthorized)
            return
        }
    })

    authorized.GET("/connect/:devid", func(c *gin.Context) {
        if !callUserHookUrl(cfg, c) {
            c.Status(http.StatusForbidden)
            return
        }

        if c.GetHeader("Upgrade") != "websocket" {
            group := c.Query("group")
            devid := c.Param("devid")
            if dev := srv.GetDevice(group, devid); dev == nil {
                c.Redirect(http.StatusFound, "/error/offline")
                return
            }

            url := "/rtty/" + devid

            if group != "" {
                url += "?group=" + group
            }

            c.Redirect(http.StatusFound, url)
        } else {
            handleUserConnection(srv, c)
        }
    })

    authorized.GET("/counts", func(c *gin.Context) {
        count := 0

        srv.groups.Range(func(key, value any) bool {
            count += int(value.(*DeviceGroup).count.Load())
            return true
        })

        c.JSON(http.StatusOK, gin.H{"count": count})
    })

    authorized.GET("/groups", func(c *gin.Context) {
        groups := []string{""}

        srv.groups.Range(func(key, value any) bool {
            if key != "" {
                groups = append(groups, key.(string))
            }
            return true
        })

        c.JSON(http.StatusOK, groups)
    })

    authorized.GET("/devs", func(c *gin.Context) {
        devs := make([]*DeviceInfo, 0)
        keyword := c.Query("keyword")

        // 1. Query all device metadata from DB (offline + online)
        metas, err := GetAllDeviceMeta(keyword)
        if err != nil || len(metas) == 0 {
            c.JSON(http.StatusOK, devs)
            return
        }

        // 2. Build online device map from memory
        onlineMap := make(map[string]*Device)

        g := srv.GetGroup("", false)
        if g != nil {
            g.devices.Range(func(key, value any) bool {
                dev := value.(*Device)
                onlineMap[dev.id] = dev
                return true
            })
        }

        now := time.Now().Unix()

        // 3. Iterate metas (DB is the source of truth)
        for _, meta := range metas {
            info := &DeviceInfo{
                ID:        meta.DeviceID,
                Mac:       meta.Mac,
                Connected: 0,
                Uptime:    0,
                Desc:      meta.Description,
                Proto:     0,
                IPaddr:    meta.IP, // fallback: last known IP
            }

            // 4. If device is online, override with in-memory data
            if dev, ok := onlineMap[meta.DeviceID]; ok {
                info.Connected = uint32(now - dev.timestamp)
                info.Uptime = dev.uptime
                info.Proto = dev.proto

                if addr, ok := dev.conn.RemoteAddr().(*net.TCPAddr); ok {
                    info.IPaddr = addr.IP.String()
                } else if host, _, err := net.SplitHostPort(dev.conn.RemoteAddr().String()); err == nil {
                    info.IPaddr = host
                }
            }

            devs = append(devs, info)
        }

        // Sort devices:
        // 1. Online devices first (Connected > 0)
        // 2. Within the same online/offline group, sort by device ID alphabetically
        sort.Slice(devs, func(i, j int) bool {
            di := devs[i]
            dj := devs[j]

            // Determine online status
            diOnline := di.Connected > 0
            djOnline := dj.Connected > 0

            if diOnline != djOnline {
                return diOnline
            }

            // If both devices are in the same state (online or offline),
            // sort by device ID in ascending alphabetical order
            return di.ID < dj.ID
        })

        c.JSON(http.StatusOK, devs)
    })

    // UpdateDeviceMetaRequest defines the JSON payload to update device metadata.
    // Only DeviceID is mandatory; other fields are optional and will be updated
    // only when provided.
    type UpdateDeviceMetaRequest struct {
        DeviceID    string `json:"deviceId" binding:"required"` // DeviceID is the unique device identifier (immutable).
        Description string `json:"description,omitempty"`       // Description can be updated if provided.
    }

    // Update device metadata (new interface)
    authorized.POST("/devs/update", func(c *gin.Context) {
        var req UpdateDeviceMetaRequest

        // 1. Parse JSON body
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{
                "code": 400,
                "msg":  "invalid request body",
                "err":  err.Error(),
            })
            return
        }

        // 2. Load existing metadata by device_id
        meta, err := GetDeviceMetaByDeviceID(req.DeviceID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{
                "code": 500,
                "msg":  "failed to query device meta",
                "err":  err.Error(),
            })
            return
        }

        if meta == nil {
            c.JSON(http.StatusNotFound, gin.H{
                "code": 404,
                "msg":  "device meta not found",
            })
            return
        }

        // 3. Merge data: deviceID/mac/ip, now only description
        newDesc := meta.Description
        if req.Description != "" {
            newDesc = req.Description
        }

        // 4. Reuse SaveOrUpdateDeviceMeta for UPSERT
        if err := SaveOrUpdateDeviceMeta(
            meta.DeviceID, // keep original device_id
            meta.Mac,      // keep original MAC, not editable
            newDesc,       // new description from request
            meta.IP,
        ); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{
                "code": 500,
                "msg":  "failed to update device meta",
                "err":  err.Error(),
            })
            return
        }

        c.JSON(http.StatusOK, gin.H{
            "code": 0,
            "msg":  "ok",
        })
    })

    // DeleteDeviceMetaRequest is used to logically delete a device meta record.
    // Only DeviceID is required.
    type DeleteDeviceMetaRequest struct {
        DeviceID string `json:"deviceId" binding:"required"` // DeviceID is the unique device identifier (immutable).
    }
    // Delete device metadata (physical delete)
    authorized.POST("/devs/delete", func(c *gin.Context) {
        var req DeleteDeviceMetaRequest

        // 1. Parse JSON body
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{
                "code": 400,
                "msg":  "invalid request body",
                "err":  err.Error(),
            })
            return
        }

        // 2. Check existence first (optional but recommended)
        meta, err := GetDeviceMetaByDeviceID(req.DeviceID)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{
                "code": 500,
                "msg":  "failed to query device meta",
                "err":  err.Error(),
            })
            return
        }

        if meta == nil {
            c.JSON(http.StatusNotFound, gin.H{
                "code": 404,
                "msg":  "device meta not found",
            })
            return
        }

        // 3. Physical delete
        if err := DeleteDeviceMetaByDeviceID(req.DeviceID); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{
                "code": 500,
                "msg":  "failed to delete device meta",
                "err":  err.Error(),
            })
            return
        }

        // 4. Success response
        c.JSON(http.StatusOK, gin.H{
            "code": 0,
            "msg":  "ok",
        })
    })

    authorized.GET("/dev/:devid", func(c *gin.Context) {
        if dev := srv.GetDevice(c.Query("group"), c.Param("devid")); dev != nil {
            info := &DeviceInfo{
                ID:        dev.id,
                Desc:      dev.desc,
                Connected: uint32(time.Now().Unix() - dev.timestamp),
                Uptime:    dev.uptime,
                Proto:     dev.proto,
                IPaddr:    dev.conn.RemoteAddr().(*net.TCPAddr).IP.String(),
            }
            c.JSON(http.StatusOK, info)
        } else {
            c.Status(http.StatusNotFound)
        }
    })

    authorized.POST("/cmd/:devid", func(c *gin.Context) {
        if !callUserHookUrl(cfg, c) {
            c.Status(http.StatusForbidden)
            return
        }

        cmdInfo := &CommandReqInfo{}

        err := c.BindJSON(&cmdInfo)
        if err != nil || cmdInfo.Cmd == "" || cmdInfo.Username == "" {
            cmdErrResp(c, rttyCmdErrInvalid)
            return
        }

        dev := srv.GetDevice(c.Query("group"), c.Param("devid"))
        if dev == nil {
            cmdErrResp(c, rttyCmdErrOffline)
            return
        }

        dev.handleCmdReq(c, cmdInfo)
    })

    authorized.Any("/web/:devid/:proto/:addr/*path", func(c *gin.Context) {
        httpProxyRedirect(srv, c, "")
    })

    authorized.Any("/web2/:group/:devid/:proto/:addr/*path", func(c *gin.Context) {
        group := c.Param("group")
        httpProxyRedirect(srv, c, group)
    })

    authorized.GET("/signout", func(c *gin.Context) {
        sid, err := c.Cookie("sid")
        if err != nil || !httpSessions.Exists(sid) {
            return
        }

        httpSessions.Del(sid)

        c.Status(http.StatusOK)
    })

    r.POST("/signin", func(c *gin.Context) {
        type credentials struct {
            Username   string `json:"username"`
            Password   string `json:"password"`
            AuthMethod string `json:"authMethod"`
        }

        creds := credentials{}

        err := c.BindJSON(&creds)
        if err != nil {
            c.Status(http.StatusBadRequest)
            return
        }

        // 自动确定认证方法或使用指定的方法 (Auto-determine auth method or use specified method)
        authMethod := creds.AuthMethod
        if authMethod == "" {
            // 基于是否提供用户名进行自动检测 (Auto-detect based on whether username is provided)
            if creds.Username != "" && cfg.LdapEnabled {
                authMethod = "ldap"
            } else {
                authMethod = "legacy"
            }
        }

        success, errorType := AuthenticateUserWithError(cfg, creds.Username, creds.Password, authMethod)
        if success {
            sid := utils.GenUniqueID()
            httpSessions.Set(sid, true, cache.WithEx(httpSessionExpire))
            c.SetCookie("sid", sid, 0, "", "", false, true)
            c.Status(http.StatusOK)
            return
        }

        // 根据错误类型返回适当的错误信息 (Return appropriate error message based on error type)
        if errorType == "authorization" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authorized"})
        } else {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication failed"})
        }
    })

    r.GET("/auth-config", func(c *gin.Context) {
        authConfig := gin.H{
            "ldapEnabled":     cfg.LdapEnabled,
            "legacyPassword":  cfg.Password != "",
            "oidcEnabled":     cfg.OIDCEnabled,
            "kvmCloudVersion": KVMCloudVersion,
        }
        c.JSON(http.StatusOK, authConfig)
    })

    r.GET("/alive", func(c *gin.Context) {
        if !httpAuth(cfg, c) {
            c.AbortWithStatus(http.StatusUnauthorized)
        } else {
            c.Status(http.StatusOK)
        }
    })

    // ===== 添加OIDC路由 =====
    RegisterOIDCRoutes(r, cfg)
    fs, err := fs.Sub(staticFs, "ui/dist")
    if err != nil {
        return err
    }

    root := http.FS(fs)
    fh := http.FileServer(root)

    r.NoRoute(func(c *gin.Context) {
        upath := path.Clean(c.Request.URL.Path)

        if strings.HasSuffix(upath, ".js") || strings.HasSuffix(upath, ".css") {
            if strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip") {
                f, err := root.Open(upath + ".gz")
                if err == nil {
                    f.Close()

                    c.Request.URL.Path += ".gz"

                    if strings.HasSuffix(upath, ".js") {
                        c.Writer.Header().Set("Content-Type", "application/javascript")
                    } else if strings.HasSuffix(upath, ".css") {
                        c.Writer.Header().Set("Content-Type", "text/css")
                    }

                    c.Writer.Header().Set("Content-Encoding", "gzip")
                }
            }
        } else if upath != "/" {
            f, err := root.Open(upath)
            if err != nil {
                c.Request.URL.Path = "/"
                r.HandleContext(c)
                return
            }
            defer f.Close()
        }

        fh.ServeHTTP(c.Writer, c.Request)
    })

    r.GET("/get/scriptInfo", func(c *gin.Context) {
        // Get domain info
        host := c.Request.Host
        hostname, _, err := net.SplitHostPort(host)
        if err != nil {
            hostname = host // Use host directly if no port
        }

        chosen := hostname
        // --------  Reverse proxy mode: force IP ----------
        if cfg.ReverseProxyEnabled {
            // Reverse proxy mode: always use configured WebRTC IP
            if strings.TrimSpace(cfg.WebrtcIP) != "" {
                chosen = strings.TrimSpace(cfg.WebrtcIP)
            }
        } else {
            // -------- 3) Original behavior (unchanged) ----------
            // 1) If hostname is domain, keep it
            // 2) If hostname is IP and cfg.WebrtcIP is set, use cfg.WebrtcIP
            if isIP(hostname) && cfg.WebrtcIP != "" {
                chosen = cfg.WebrtcIP
            }
        }

        c.JSON(http.StatusOK, gin.H{
            "hostname":       chosen, // reuse the same chosen value
            "port":           cfg.AddrDev,
            "token":          cfg.Token,
            "webrtcIP":       chosen, // same as hostname
            "webrtcPort":     cfg.WebrtcPort,
            "webrtcUsername": cfg.WebrtcUsername,
            "webrtcPassword": cfg.WebrtcPassword,
        })
    })

    ln, err := net.Listen("tcp", cfg.AddrUser)
    if err != nil {
        return err
    }
    defer ln.Close()

    // If we're behind a reverse proxy (TLS terminated by nginx), never enable TLS here.
    enableTLS := !cfg.ReverseProxyEnabled && cfg.SslCert != "" && cfg.SslKey != ""

    if enableTLS {
        crt, err := tls.LoadX509KeyPair(cfg.SslCert, cfg.SslKey)
        if err != nil {
            log.Fatal().Msg(err.Error())
        }

        tlsConfig := &tls.Config{Certificates: []tls.Certificate{crt}}

        ln = tls.NewListener(ln, tlsConfig)
    }

    log.Info().Msgf("Listen users on: %s", ln.Addr().(*net.TCPAddr))

    return r.RunListener(ln)
}

func isIP(addr string) bool {
    return net.ParseIP(addr) != nil
}

func callUserHookUrl(cfg *Config, c *gin.Context) bool {
    if cfg.UserHookUrl == "" {
        return true
    }

    upath := c.Request.URL.RawPath

    // Create HTTP request with original headers
    req, err := http.NewRequest("GET", cfg.UserHookUrl, nil)
    if err != nil {
        log.Error().Err(err).Msgf("create hook request for \"%s\" fail", upath)
        return false
    }

    // Copy all headers from original request
    for key, values := range c.Request.Header {
        lowerKey := strings.ToLower(key)
        if lowerKey == "upgrade" || lowerKey == "connection" || lowerKey == "accept-encoding" {
            continue
        }

        for _, value := range values {
            req.Header.Add(key, value)
        }
    }

    // Add custom headers for hook identification
    req.Header.Set("X-Rttys-Hook", "true")
    req.Header.Set("X-Original-Method", c.Request.Method)
    req.Header.Set("X-Original-URL", c.Request.URL.String())

    cli := &http.Client{
        Timeout: 3 * time.Second,
    }

    resp, err := cli.Do(req)
    if err != nil {
        log.Error().Err(err).Msgf("call user hook url for \"%s\" fail", upath)
        return false
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        log.Error().Msgf("call user hook url for \"%s\", StatusCode: %d", upath, resp.StatusCode)
        return false
    }

    return true
}

func httpLogin(cfg *Config, password string) bool {
    return cfg.Password == password
}

func isLocalRequest(c *gin.Context) bool {
    addr, _ := net.ResolveTCPAddr("tcp", c.Request.RemoteAddr)
    return addr.IP.IsLoopback()
}

func httpAuth(cfg *Config, c *gin.Context) bool {
    if !cfg.LocalAuth && isLocalRequest(c) {
        return true
    }

    if cfg.Password == "" {
        return true
    }

    sid, err := c.Cookie("sid")
    if err != nil || !httpSessions.Exists(sid) {
        return false
    }

    httpSessions.Expire(sid, httpSessionExpire)

    return true
}
