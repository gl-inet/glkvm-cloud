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
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"path"
	"rttys/internal/domain/device"
	"rttys/internal/domain/devicelog"
	"rttys/internal/domain/permission"
	"rttys/internal/domain/user"
	httpx "rttys/internal/http"
	"rttys/internal/http/middleware"
	"rttys/internal/pkg/password"
	"rttys/internal/proxy"
	"rttys/internal/store/memory"
	"rttys/internal/store/sqlite"
	"rttys/ui"
	"rttys/xconfig"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type AppContainer struct {
	DB             *sqlite.AppDB
	DeviceMetaRepo *sqlite.DeviceMetaRepo
	UserSvc        *user.Service
	DeviceLogSvc   *devicelog.Service
}

var sessionStore *memory.SessionStore

const defaultDBPath = "/home/database/glkvm-cloud.db"

func InitAppContainer(r *gin.Engine) (*AppContainer, error) {
	ctx := context.Background()
	cfg := xconfig.Must()
	// --- DB ---
	appDB, err := sqlite.Open(ctx, sqlite.Options{
		DSN:          defaultDBPath,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
		LogSQL:       true,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("open sqlite failed")
	}
	deviceMetaRepo := sqlite.NewDeviceMetaRepo(appDB.Gorm())

	if err := sqlite.InitSchema(ctx, appDB.SQL(), "/home/database/schema.sql"); err != nil {
		log.Fatal().Err(err).Msg("init schema failed")
	}
	if err := ensureAdminUser(ctx, appDB.Gorm(), cfg.AdminName, cfg.Password); err != nil {
		log.Fatal().Err(err).Msg("ensure admin user failed")
	}

	// --- Repos & Services ---
	userRepo := sqlite.NewUserRepo(appDB.Gorm())
	groupRepo := sqlite.NewGroupRepo(appDB.Gorm())
	deviceRepo := sqlite.NewDeviceRepo(appDB.Gorm())
	relationsRepo := sqlite.NewRelationsRepo(appDB.Gorm())
	trustedDeviceRepo := sqlite.NewTrustedDeviceRepo(appDB.Gorm())
	deviceLogRepo := sqlite.NewDeviceLogRepo(appDB.Gorm())

	userSvc := user.NewService(userRepo)
	devSvc := device.NewService(deviceRepo, groupRepo)
	deviceLogSvc := devicelog.NewService(deviceLogRepo)

	permRepo := memory.NewPermissionRepo() // permissions stay in-memory
	permSvc := permission.NewService(permRepo)

	sessionStore = memory.NewSessionStore(cfg.AuthSessionTTL)

	httpx.RegisterAPIRoutes(r, httpx.Deps{
		UserSvc:           userSvc,
		PermSvc:           permSvc,
		DevSvc:            devSvc,
		GroupRepo:         groupRepo,
		SessionStore:      sessionStore,
		RelationsRepo:     relationsRepo,
		TrustedDeviceRepo: trustedDeviceRepo,
		DeviceLogSvc:      deviceLogSvc,
		Cfg:               cfg,
		CloudVersion:      KVMCloudVersion,
	})

	c := &AppContainer{
		DB:             appDB,
		DeviceMetaRepo: deviceMetaRepo,
		UserSvc:        userSvc,
		DeviceLogSvc:   deviceLogSvc,
	}
	return c, nil
}

func ensureAdminUser(ctx context.Context, db *gorm.DB, adminName, plainPassword string) error {
    if db == nil {
        return fmt.Errorf("db is nil")
    }

    hash, err := password.HashPassword(plainPassword)
    if err != nil {
        return err
    }

    // First, rename the existing system admin user to the configured name (if changed).
    // This handles the case where the admin username was previously "admin" (or another name)
    // and the user now wants a different username via RTTYS_ADMIN_NAME.
    if err := db.WithContext(ctx).Exec(
        `UPDATE users SET username = ? WHERE is_system = 1 AND role = 'admin' AND username != ?`,
        adminName, adminName,
    ).Error; err != nil {
        return fmt.Errorf("rename system admin user: %w", err)
    }

    // Upsert: create the admin user if not exists, or update password/role/status.
    // On conflict, also set description to 'System Administrator' if it is currently empty.
    return db.WithContext(ctx).Exec(
        `INSERT INTO users (username, description, password_hash, role, status, is_system)
         VALUES (?, 'System Administrator', ?, 'admin', 'active', 1)
         ON CONFLICT(username) DO UPDATE SET
           password_hash=excluded.password_hash,
           role='admin',
           status='active',
           is_system=1,
           description=CASE WHEN (description IS NULL OR description = '') THEN 'System Administrator' ELSE description END`,
        adminName, hash,
    ).Error
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

	container, err := InitAppContainer(r)
	if err != nil {
		return err
	}
	defer container.DB.Close()
	sqlite.SetContainer(&sqlite.Container{
		Gorm:         container.DB.Gorm(),
		DeviceMeta:   sqlite.NewDeviceMetaRepo(container.DB.Gorm()),
		DeviceLogSvc: container.DeviceLogSvc,
		UserSvc:      container.UserSvc,
	})

	// ===== 添加OIDC路由 =====
	RegisterOIDCRoutes(r, cfg, container.UserSvc)

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

// principalFromCtx best-effort extracts the logged-in user (id + username)
// from the request, used for tagging device-event logs. Returns (0, "")
// when no session can be resolved. Never blocks the calling path.
func principalFromCtx(c *gin.Context) (int64, string) {
	if c == nil || c.Request == nil || sessionStore == nil {
		return 0, ""
	}
	sid, err := c.Cookie("sid")
	if err != nil {
		return 0, ""
	}
	sid = strings.TrimSpace(sid)
	if sid == "" {
		return 0, ""
	}
	sess, ok := sessionStore.Get(sid)
	if !ok {
		return 0, ""
	}
	cont := sqlite.TryContainer()
	if cont == nil || cont.UserSvc == nil {
		return sess.UserID, ""
	}
	u, err := cont.UserSvc.FindByID(c.Request.Context(), sess.UserID)
	if err != nil || u == nil {
		return sess.UserID, ""
	}
	return sess.UserID, u.Username
}
