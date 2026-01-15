package middleware

import (
    "net/http"
    "rttys/internal/domain/identity"
    "strings"

    "rttys/internal/domain/permission"
    "rttys/internal/domain/user"
    "rttys/internal/http/dto"
    "rttys/internal/store/memory"

    "github.com/gin-gonic/gin"
)

const PrincipalKey = "principal"

type Principal struct {
    UserID         int64         `json:"userId"`
    Username       string        `json:"username"`
    DisplayName    string        `json:"displayName"`
    Role           identity.Role `json:"role"`
    PermissionKeys []string      `json:"permissions"`
}

func MustPrincipal(c *gin.Context) Principal {
    v, ok := c.Get(PrincipalKey)
    if !ok {
        panic("principal missing")
    }
    return v.(Principal)
}

func Auth(sessionStore *memory.SessionStore, userSvc *user.Service, permSvc *permission.Service) gin.HandlerFunc {
    return func(c *gin.Context) {
        traceID := GetTraceID(c)
        token := parseBearer(c.GetHeader("Authorization"))
        if token == "" {
            dto.Write(c, dto.Err(traceID, dto.CodeAuthRequired, "Please login", nil))
            c.Abort()
            return
        }

        sess, ok := sessionStore.Get(token)
        if !ok {
            dto.Write(c, dto.Err(traceID, dto.CodeAuthExpired, "Session expired", nil))
            c.Abort()
            return
        }

        u, err := userSvc.GetByID(c.Request.Context(), sess.UserID)
        if err != nil || u == nil {
            dto.Write(c, dto.Err(traceID, dto.CodeForbidden, "Permission denied", nil))
            c.Abort()
            return
        }

        keys, _ := permSvc.ListByRole(c.Request.Context(), u.Role)
        perms := make([]string, 0, len(keys))
        for _, k := range keys {
            perms = append(perms, string(k))
        }

        c.Set(PrincipalKey, Principal{
            UserID:         u.ID,
            Username:       u.Email,
            DisplayName:    u.DisplayName,
            Role:           u.Role,
            PermissionKeys: perms,
        })

        c.Next()
    }
}

// Require checks capability keys (frontend/back-end single source of truth).
func Require(required permission.Key) gin.HandlerFunc {
    return func(c *gin.Context) {
        traceID := GetTraceID(c)
        p := MustPrincipal(c)
        for _, k := range p.PermissionKeys {
            if k == string(required) {
                c.Next()
                return
            }
        }
        dto.Write(c, dto.Err(traceID, dto.CodeForbidden, "Permission denied", map[string]any{
            "required": string(required),
        }))
        c.Abort()
    }
}

func parseBearer(v string) string {
    v = strings.TrimSpace(v)
    if v == "" {
        return ""
    }
    parts := strings.SplitN(v, " ", 2)
    if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
        return strings.TrimSpace(parts[1])
    }
    return ""
}

// Write wrapper for gin to keep consistent HTTP 200.
func Write(c *gin.Context, payload any) {
    c.JSON(http.StatusOK, payload)
}
