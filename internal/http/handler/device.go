package handler

import (
	"rttys/internal/domain/device"
	"rttys/internal/domain/permission"
	"rttys/internal/http/dto"
	"rttys/internal/http/middleware"

	"github.com/gin-gonic/gin"
)

type DeviceHandler struct {
	devSvc *device.Service
}

func NewDeviceHandler(devSvc *device.Service) *DeviceHandler { return &DeviceHandler{devSvc: devSvc} }

// GET /api/devices
func (h *DeviceHandler) ListDevices(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	p := middleware.MustPrincipal(c)

	items, err := h.devSvc.ListVisible(c.Request.Context(), p.Role, p.UserID)
	if err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
		return
	}

	out := make([]dto.Device, 0, len(items))
	for _, d := range items {
		out = append(out, dto.Device{
			ID:            d.ID,
			DeviceUID:     d.DeviceUID,
			Name:          d.Name,
			Description:   d.Description,
			DeviceGroupID: d.DeviceGroupID,
			Status:        string(d.Status),
			LastSeenAt:    d.LastSeenAt,
		})
	}

	dto.Write(c, dto.Ok(traceID, dto.ListDevicesResp{Items: out}))
}

var _ = permission.DeviceRead
