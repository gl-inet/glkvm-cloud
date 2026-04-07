package handler

import (
	"rttys/internal/domain/identity"
	"rttys/internal/domain/trusteddevice"
	"rttys/internal/pkg/ldap"
	"rttys/internal/pkg/totp"
	"rttys/internal/pkg/useragent"
	"rttys/xconfig"
	"strings"
	"time"

	"rttys/internal/domain/user"
	"rttys/internal/http/dto"
	"rttys/internal/http/middleware"
	"rttys/internal/pkg/randtoken"
	"rttys/internal/store/memory"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

const (
	trustedDeviceCookieName = "td"
	trustedDeviceTTL        = 30 * 24 * time.Hour
)

type AuthHandler struct {
	userSvc           *user.Service
	sessionStore      *memory.SessionStore
	trustedDeviceRepo trusteddevice.Repository
}

func NewAuthHandler(userSvc *user.Service, sessionStore *memory.SessionStore, tdRepo trusteddevice.Repository) *AuthHandler {
	return &AuthHandler{
		userSvc:           userSvc,
		sessionStore:      sessionStore,
		trustedDeviceRepo: tdRepo,
	}
}

// POST /api/login
func (h *AuthHandler) Login(c *gin.Context) {
	traceID := middleware.GetTraceID(c)

	var req dto.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{
			"field": "username/password",
		}))
		return
	}

	cfg := xconfig.Must()
	var userID int64
	// ---- LDAP ----
	authMethod := req.AuthMethod
	if authMethod == "ldap" {
		ok, errorType, userDN, isAdmin := ldap.AuthenticateUserWithError(cfg, req.Username, req.Password, authMethod)
		if !ok {
			if errorType == "authorization" {
				dto.Write(c, dto.Err(traceID, dto.CodeForbidden, "User not authorized", nil))
			} else {
				dto.Write(c, dto.Err(traceID, dto.CodeForbidden, "Authentication failed", nil))
			}
			return
		}
		role := identity.RoleUser
		if isAdmin {
			role = identity.RoleAdmin
		}
		ldapUser, err := h.userSvc.FindOrCreateExternalUser(c.Request.Context(), "ldap", userDN, req.Username, "", req.Username, role)
		if err != nil {
			dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Failed to create LDAP user", nil))
			return
		}
		log.Info().
			Str("username", req.Username).
			Str("userDN", userDN).
			Str("role", string(role)).
			Int64("userID", ldapUser.ID).
			Msg("LDAP user login completed")
		userID = ldapUser.ID
	} else {
		u, err := h.userSvc.Authenticate(c.Request.Context(), req.Username, req.Password)
		if err != nil || u == nil {
			dto.Write(c, dto.Err(traceID, dto.CodeForbidden, "Authentication failed", nil))
			return
		}

		// Local user with 2FA enabled: enforce TOTP unless a valid trusted-device cookie is present.
		if u.TotpEnabled && u.TotpSecret != "" {
			if !h.trustedDeviceCookieValid(c, u.ID) {
				if strings.TrimSpace(req.TotpCode) == "" {
					dto.Write(c, dto.Ok(traceID, dto.LoginResp{TwoFactorRequired: true}))
					return
				}
				if !totp.Verify(u.TotpSecret, strings.TrimSpace(req.TotpCode)) {
					dto.Write(c, dto.Err(traceID, dto.CodeForbidden, "Invalid verification code", nil))
					return
				}
				// Optionally remember this device.
				if req.RememberDevice {
					h.issueTrustedDevice(c, u.ID)
				}
			}
		}

		userID = u.ID
	}

	sid, err := randtoken.New()
	if err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
		return
	}

	h.sessionStore.Create(sid, userID)

	// Best-effort: refresh last_login_at. Failure here should not block login.
	_ = h.userSvc.TouchLastLogin(c.Request.Context(), userID, time.Now().Unix())

	dto.Write(c, dto.Ok(traceID, dto.LoginResp{
		Token: sid,
	}))
}

// trustedDeviceCookieValid returns true if the request carries a non-expired
// trusted-device cookie that maps to the given user. It also refreshes
// last_used_at as a side effect.
func (h *AuthHandler) trustedDeviceCookieValid(c *gin.Context, userID int64) bool {
	if h.trustedDeviceRepo == nil {
		return false
	}
	token, err := c.Cookie(trustedDeviceCookieName)
	if err != nil || strings.TrimSpace(token) == "" {
		return false
	}
	dev, err := h.trustedDeviceRepo.FindByToken(c.Request.Context(), strings.TrimSpace(token))
	if err != nil || dev == nil {
		return false
	}
	if dev.UserID != userID {
		return false
	}
	now := time.Now().Unix()
	if dev.ExpiresAt < now {
		_ = h.trustedDeviceRepo.Delete(c.Request.Context(), dev.ID, dev.UserID)
		return false
	}
	_ = h.trustedDeviceRepo.TouchLastUsed(c.Request.Context(), dev.ID, now)
	return true
}

// issueTrustedDevice creates a new trusted-device record and writes the token cookie.
func (h *AuthHandler) issueTrustedDevice(c *gin.Context, userID int64) {
	if h.trustedDeviceRepo == nil {
		return
	}
	token, err := randtoken.New()
	if err != nil {
		log.Warn().Err(err).Msg("trusted device: generate token failed")
		return
	}
	now := time.Now()
	dev := &trusteddevice.Device{
		UserID:     userID,
		Token:      token,
		DeviceName: trimToLen(useragent.Friendly(c.Request.UserAgent()), 200),
		IP:         clientIP(c),
		CreatedAt:  now.Unix(),
		LastUsedAt: now.Unix(),
		ExpiresAt:  now.Add(trustedDeviceTTL).Unix(),
	}
	if _, err := h.trustedDeviceRepo.Create(c.Request.Context(), dev); err != nil {
		log.Warn().Err(err).Msg("trusted device: create failed")
		return
	}
	c.SetCookie(trustedDeviceCookieName, token, int(trustedDeviceTTL.Seconds()), "/", "", false, true)
}

func clientIP(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	if ip := c.ClientIP(); ip != "" {
		return ip
	}
	return c.Request.RemoteAddr
}

func trimToLen(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// POST /api/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	traceID := middleware.GetTraceID(c)

	// 1) Try bearer
	authz := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
		token := strings.TrimSpace(authz[7:])
		if token != "" {
			h.sessionStore.Delete(token)
		}
	}

	// 2) Try cookie sid
	if sid, err := c.Cookie("sid"); err == nil && strings.TrimSpace(sid) != "" {
		h.sessionStore.Delete(strings.TrimSpace(sid))
		// 清 cookie
		c.SetCookie("sid", "", -1, "/", "", false, true)
	}

	dto.Write(c, dto.Ok(traceID, dto.LogoutResp{}))
}
