package handler

import (
    "net/http"
    "strings"

    "rttys/internal/domain/permission"
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

    u, err := h.userSvc.Authenticate(c.Request.Context(), req.Username, req.Password)
    if err != nil || u == nil {
        // do not leak details
        dto.Write(c, dto.Err(traceID, dto.CodeForbidden, "Permission denied", nil))
        return
    }

    token, err := randtoken.New()
    if err != nil {
        dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
        return
    }
    h.sessionStore.Create(token, u.ID)

    dto.Write(c, dto.Ok(traceID, dto.LoginResp{Token: token}))
}

// POST /api/logout
func (h *AuthHandler) Logout(c *gin.Context) {
    traceID := middleware.GetTraceID(c)
    p := middleware.MustPrincipal(c)

    // Remove current bearer token (best-effort).
    authz := strings.TrimSpace(c.GetHeader("Authorization"))
    if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
        token := strings.TrimSpace(authz[7:])
        if token != "" {
            h.sessionStore.Delete(token)
        }
    }
    _ = p
    dto.Write(c, dto.Ok(traceID, dto.LogoutResp{}))
}

var _ = permission.AuthWrite // keep imports stable when extending
var _ = http.StatusOK
