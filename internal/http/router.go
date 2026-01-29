package http

import (
	"net"
	"strings"

	"rttys/internal/domain/device"
	"rttys/internal/domain/permission"
	"rttys/internal/domain/user"
	"rttys/internal/http/dto"
	"rttys/internal/http/handler"
	"rttys/internal/http/middleware"
	"rttys/internal/store/memory"
	"rttys/internal/store/sqlite"
	"rttys/xconfig"

	"github.com/gin-gonic/gin"
)

type Deps struct {
	UserSvc       *user.Service
	PermSvc       *permission.Service
	DevSvc        *device.Service
	GroupRepo     *sqlite.GroupRepo
	SessionStore  *memory.SessionStore
	RelationsRepo *sqlite.RelationsRepo
	Cfg           *xconfig.Config
	CloudVersion  string
}

func RegisterAPIRoutes(r *gin.Engine, d Deps) {
	cfg := d.Cfg
	if cfg == nil {
		cfg = xconfig.Must()
	}

	authH := handler.NewAuthHandler(d.UserSvc, d.SessionStore)
	meH := handler.NewMeHandler()
	devH := handler.NewDeviceHandler(d.DevSvc, d.GroupRepo, d.RelationsRepo)
	dgH := handler.NewDeviceGroupHandler(d.GroupRepo, d.RelationsRepo)
	ugH := handler.NewUserGroupHandler(d.GroupRepo)
	relH := handler.NewRelationsHandler(d.RelationsRepo)

	userH := handler.NewUserHandler(d.UserSvc, d.GroupRepo, d.RelationsRepo)

	// public
	r.GET("/auth-config", func(c *gin.Context) {
		traceID := middleware.GetTraceID(c)
		data := authConfigResp{
			LdapEnabled:     cfg.LdapEnabled,
			LegacyPassword:  cfg.Password != "",
			OidcEnabled:     cfg.OIDCEnabled,
			KVMCloudVersion: d.CloudVersion,
		}

		dto.Write(c, dto.Ok(traceID, data))
	})

	// public
	r.POST("/api/login", authH.Login)

	// authed group
	api := r.Group("/api")
	api.Use(middleware.Auth(d.SessionStore, d.UserSvc, d.PermSvc))

	// device script info (requires login)
	api.GET("/script-info", func(c *gin.Context) {
		traceID := middleware.GetTraceID(c)

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

		data := scriptInfoResp{
			Hostname:       chosen, // reuse the same chosen value
			Port:           cfg.AddrDev,
			Token:          cfg.Token,
			WebrtcIP:       chosen, // same as hostname
			WebrtcPort:     cfg.WebrtcPort,
			WebrtcUsername: cfg.WebrtcUsername,
			WebrtcPassword: cfg.WebrtcPassword,
		}

		dto.Write(c, dto.Ok(traceID, data))
	})

	// auth
	api.POST("/logout", middleware.Require(permission.AuthWrite), authH.Logout)

	// me
	api.GET("/me", middleware.Require(permission.MeRead), meH.GetMe)

	// device scope list
	api.GET("/devices", middleware.Require(permission.DeviceRead), devH.ListDevices)
	api.POST("/devices/move-to-device-group", middleware.Require(permission.DeviceGroupWrite), devH.MoveToDeviceGroup)

	// --- users ---
	api.GET("/users", middleware.Require(permission.UserRead), userH.ListUsers)
	api.POST("/users", middleware.Require(permission.UserWrite), userH.CreateUser)
	api.PUT("/users/:id", middleware.Require(permission.UserWrite), userH.UpdateUser)
	api.DELETE("/users/:id", middleware.Require(permission.UserWrite), userH.DeleteUser)

	// user groups
	api.GET("/user-groups", middleware.Require(permission.UserGroupRead), ugH.ListUserGroups)
	api.GET("/user-groups/options", middleware.Require(permission.UserGroupRead), ugH.ListOptions)
	api.POST("/user-groups", middleware.Require(permission.UserGroupWrite), ugH.Create)
	api.PUT("/user-groups/:id", middleware.Require(permission.UserGroupWrite), ugH.Update)
	api.DELETE("/user-groups/:id", middleware.Require(permission.UserGroupWrite), ugH.Delete)

	// device groups list
	api.GET("/device-groups", middleware.Require(permission.DeviceGroupRead), dgH.ListDeviceGroups)
	api.GET("/device-groups/options", middleware.Require(permission.DeviceGroupRead), dgH.ListOptions)
	api.POST("/device-groups", middleware.Require(permission.DeviceGroupWrite), dgH.Create)
	api.PUT("/device-groups/:id", middleware.Require(permission.DeviceGroupWrite), dgH.Update)
	api.DELETE("/device-groups/:id", middleware.Require(permission.DeviceGroupWrite), dgH.Delete)
	api.POST("/device-groups/:id/devices", middleware.Require(permission.DeviceGroupWrite), dgH.AddDevices)
	api.DELETE("/device-groups/:id/devices", middleware.Require(permission.DeviceGroupWrite), dgH.RemoveDevices)

	// Relations (cover / set)
	api.PUT("/users/:id/user-groups", middleware.Require(permission.UserWrite), relH.SetUserGroups)
	api.PUT("/user-groups/:id/device-groups", middleware.Require(permission.UserGroupWrite), relH.SetUserGroupDeviceGroups)
	api.PUT("/device-groups/:id/devices", middleware.Require(permission.DeviceGroupWrite), relH.SetDeviceGroupDevices)
}

type authConfigResp struct {
	LdapEnabled     bool   `json:"ldapEnabled"`
	LegacyPassword  bool   `json:"legacyPassword"`
	OidcEnabled     bool   `json:"oidcEnabled"`
	KVMCloudVersion string `json:"kvmCloudVersion"`
}

type scriptInfoResp struct {
	Hostname       string `json:"hostname"`
	Port           string `json:"port"`
	Token          string `json:"token"`
	WebrtcIP       string `json:"webrtcIP"`
	WebrtcPort     string `json:"webrtcPort"`
	WebrtcUsername string `json:"webrtcUsername"`
	WebrtcPassword string `json:"webrtcPassword"`
}

func isIP(addr string) bool {
	return net.ParseIP(addr) != nil
}
