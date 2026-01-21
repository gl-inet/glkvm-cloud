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

package server

import (
	"context"
	"crypto/tls"
	"io/fs"
	"net"
	"net/http"
	"path"
	"rttys/internal/domain/device"
	"rttys/internal/domain/permission"
	"rttys/internal/domain/user"
	httpx "rttys/internal/http"
	"rttys/internal/http/middleware"
	"rttys/internal/legacy"
	"rttys/internal/pkg/ldap"
	"rttys/internal/pkg/randtoken"
	"rttys/internal/proxy"
	"rttys/internal/store/memory"
	"rttys/internal/store/sqlite"
	"rttys/ui"
	"rttys/xconfig"
	"sort"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type AppContainer struct {
	DB             *sqlite.AppDB
	DeviceMetaRepo *sqlite.DeviceMetaRepo
}

var sessionStore *memory.SessionStore

func InitAppContainer(r *gin.Engine) (*AppContainer, error) {
	ctx := context.Background()
	cfg := xconfig.Must()
	// --- DB ---
	appDB, err := sqlite.Open(ctx, sqlite.Options{
		DSN:          "/home/database/glkvm-cloud.db",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("open sqlite failed")
	}
	_ = sqlite.NewDeviceMetaRepo(appDB.Gorm())

	if err := sqlite.InitSchema(ctx, appDB.SQL(), "/home/database/schema.sql"); err != nil {
		log.Fatal().Err(err).Msg("init schema failed")
	}

	// --- Repos & Services ---
	userRepo := sqlite.NewUserRepo(appDB.Gorm())
	groupRepo := sqlite.NewGroupRepo(appDB.Gorm())
	deviceRepo := sqlite.NewDeviceRepo(appDB.Gorm())
	relationsRepo := sqlite.NewRelationsRepo(appDB.Gorm())

	userSvc := user.NewService(userRepo)
	devSvc := device.NewService(deviceRepo, groupRepo)

	permRepo := memory.NewPermissionRepo() // permissions stay in-memory
	permSvc := permission.NewService(permRepo)

	sessionStore = memory.NewSessionStore(cfg.AuthSessionTTL)

	httpx.RegisterAPIRoutes(r, httpx.Deps{
		UserSvc:       userSvc,
		PermSvc:       permSvc,
		DevSvc:        devSvc,
		GroupRepo:     groupRepo,
		SessionStore:  sessionStore,
		RelationsRepo: relationsRepo,
	})

	c := &AppContainer{
		DB:             appDB,
		DeviceMetaRepo: sqlite.NewDeviceMetaRepo(appDB.Gorm()),
	}
	return c, nil
}

func (srv *RttyServer) ListenAPI() error {
	cfg := &srv.cfg

	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Trace())

	r.Use(func(c *gin.Context) {
		hi := proxy.GetHostInfoFromRequest(c.Request)

		host := hi.Host
		allowedHost := cfg.WebUIHost
		// If WebUIHost is configured, enforce host validation
		if allowedHost != "" && !proxy.IsIPHost(host) {
			if !proxy.DomainAllowed(host, allowedHost) {
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
		metas, err := legacy.GetAllDeviceMeta(keyword)
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
		meta, err := legacy.GetDeviceMetaByDeviceID(req.DeviceID)
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
		if err := legacy.SaveOrUpdateDeviceMeta(
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
		meta, err := legacy.GetDeviceMetaByDeviceID(req.DeviceID)
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
		if err := legacy.DeleteDeviceMetaByDeviceID(req.DeviceID); err != nil {
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
		if err != nil || strings.TrimSpace(sid) == "" {
			c.Status(http.StatusOK)
			return
		}
		sid = strings.TrimSpace(sid)

		// Only use the new session store
		sessionStore.Delete(sid)

		// Clear cookie
		c.SetCookie("sid", "", -1, "/", "", false, true)

		c.Status(http.StatusOK)
	})

	r.POST("/signin", func(c *gin.Context) {
		type credentials struct {
			Username   string `json:"username"`
			Password   string `json:"password"`
			AuthMethod string `json:"authMethod"`
		}

		creds := credentials{}
		if err := c.ShouldBindJSON(&creds); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		cfg := xconfig.Must()

		// Auto-detect auth method (keep original behavior)
		authMethod := creds.AuthMethod
		if authMethod == "" {
			if creds.Username != "" && cfg.LdapEnabled {
				authMethod = "ldap"
			} else {
				authMethod = "legacy"
			}
		}

		// Temporary behavior: all successful logins are treated as admin
		const adminUserID int64 = 1
		var ok bool
		var errorType string

		if authMethod == "ldap" {
			ok, errorType = ldap.AuthenticateUserWithError(cfg, creds.Username, creds.Password, authMethod)
		} else {
			// legacy path: keep old password auth behavior
			ok, errorType = ldap.AuthenticateUserWithError(cfg, creds.Username, creds.Password, "legacy")
			// 可以直接用：
			// ok = (cfg.Password != "" && cfg.Password == creds.Password)
			// errorType = "authentication"
		}

		if !ok {
			if errorType == "authorization" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authorized"})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication failed"})
			}
			return
		}

		// Create sid using the same token generator as new API
		sid, err := randtoken.New()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
			return
		}

		// Store session in the new sessionStore (same as /api/login)
		// NOTE: ensure sessionStore is accessible here (capture it from outer scope or srv/container)
		sessionStore.Create(sid, adminUserID)

		// Set cookie
		c.SetCookie("sid", sid, 0, "/", "", false, true)

		c.Status(http.StatusOK)
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
	container, err := InitAppContainer(r)
	if err != nil {
		return err
	}
	defer container.DB.Close()
	sqlite.SetContainer(&sqlite.Container{
		Gorm:       container.DB.Gorm(),
		DeviceMeta: sqlite.NewDeviceMetaRepo(container.DB.Gorm()),
	})

    fs, err := fs.Sub(ui.StaticFS, "dist")
	if err != nil {
		return err
	}

	root := http.FS(fs)
	fh := http.FileServer(root)
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "not found"})
			return
		}

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

func callUserHookUrl(cfg *xconfig.Config, c *gin.Context) bool {
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

func isLocalRequest(c *gin.Context) bool {
	addr, _ := net.ResolveTCPAddr("tcp", c.Request.RemoteAddr)
	return addr.IP.IsLoopback()
}

func httpAuth(cfg *xconfig.Config, c *gin.Context) bool {
	if !cfg.LocalAuth && isLocalRequest(c) {
		return true
	}

	// Keep legacy behavior: if password is not set, no auth required
	if cfg.Password == "" {
		return true
	}

	sid, err := c.Cookie("sid")
	if err != nil || strings.TrimSpace(sid) == "" {
		return false
	}
	sid = strings.TrimSpace(sid)

	// New session-based auth
	_, ok := sessionStore.Get(sid)
	return ok
}
