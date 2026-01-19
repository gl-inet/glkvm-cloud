package handler

import (
    "rttys/internal/pkg/ldap"
    "rttys/xconfig"
    "strings"

    "rttys/internal/domain/user"
    "rttys/internal/http/dto"
    "rttys/internal/http/middleware"
    "rttys/internal/pkg/randtoken"
    "rttys/internal/store/memory"

    "github.com/gin-gonic/gin"
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
        ok, errorType := ldap.AuthenticateUserWithError(cfg, req.Username, req.Password, authMethod)
        if !ok {
            // Keep behavior consistent with /signin
            if errorType == "authorization" {
                dto.Write(c, dto.Err(traceID, dto.CodeForbidden, "User not authorized", nil))
            } else {
                dto.Write(c, dto.Err(traceID, dto.CodeForbidden, "Authentication failed", nil))
            }
            return
        }
        const adminUserID int64 = 1
        userID = adminUserID
    } else {
        u, err := h.userSvc.Authenticate(c.Request.Context(), req.Username, req.Password)
        if err != nil || u == nil {
            // do not leak details
            dto.Write(c, dto.Err(traceID, dto.CodeForbidden, "Permission denied", nil))
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

    c.SetCookie("sid", sid, 0, "/", "", false, true)
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
