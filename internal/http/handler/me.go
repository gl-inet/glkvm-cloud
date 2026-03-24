package handler

import (
	"rttys/internal/http/dto"
	"rttys/internal/http/middleware"

	"github.com/gin-gonic/gin"
)

type MeHandler struct{}

func NewMeHandler() *MeHandler { return &MeHandler{} }

// GET /api/me
func (h *MeHandler) GetMe(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	p := middleware.MustPrincipal(c)

	dto.Write(c, dto.Ok(traceID, dto.MeResp{
		User: dto.MeUser{
			ID:           p.UserID,
			Username:     p.Username,
			DisplayName:  p.DisplayName,
			Role:         string(p.Role),
			AuthProvider: p.AuthProvider,
		},
		Permissions: p.PermissionKeys,
	}))
}
