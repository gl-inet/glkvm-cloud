package handler

import (
	"rttys/internal/domain/identity"
	"rttys/internal/pkg/ldap"
	"rttys/xconfig"
	"strings"

	"rttys/internal/domain/user"
	"rttys/internal/http/dto"
	"rttys/internal/http/middleware"
	"rttys/internal/pkg/randtoken"
	"rttys/internal/store/memory"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type AuthHandler struct {
	userSvc      *user.Service
	sessionStore *memory.SessionStore
}

func NewAuthHandler(userSvc *user.Service, sessionStore *memory.SessionStore) *AuthHandler {
	return &AuthHandler{userSvc: userSvc, sessionStore: sessionStore}
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
		userID = u.ID
	}

	sid, err := randtoken.New()
	if err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
		return
	}

	h.sessionStore.Create(sid, userID)

	dto.Write(c, dto.Ok(traceID, dto.LoginResp{
		Token: sid,
	}))
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
