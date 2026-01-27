package handler

import (
	"rttys/internal/domain/device"
	"rttys/internal/http/dto"
	"rttys/internal/http/middleware"
	"rttys/internal/store/sqlite"
	"time"

	"github.com/gin-gonic/gin"
)

type DeviceHandler struct {
	devSvc         *device.Service
	groupRepo      *sqlite.GroupRepo
	relationsRepo  *sqlite.RelationsRepo
}

func NewDeviceHandler(
	devSvc *device.Service,
	groupRepo *sqlite.GroupRepo,
	relationsRepo *sqlite.RelationsRepo,
) *DeviceHandler {
	return &DeviceHandler{
		devSvc:         devSvc,
		groupRepo:      groupRepo,
		relationsRepo:  relationsRepo,
	}
}

// GET /api/devices
func (h *DeviceHandler) ListDevices(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	p := middleware.MustPrincipal(c)

	items, err := h.devSvc.ListVisible(c.Request.Context(), p.Role, p.UserID)
	if err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
		return
	}

	groupNameByID := map[int64]string{}
	if h.groupRepo != nil {
		groups, err := h.groupRepo.ListDeviceGroupsVisibleToUser(c.Request.Context(), p.UserID, string(p.Role) == "admin")
		if err != nil {
			dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
			return
		}
		for _, g := range groups {
			groupNameByID[g.ID] = g.Name
		}
	}

	now := time.Now().Unix()
	out := make([]dto.Device, 0, len(items))
	for _, d := range items {
		var groupName string
		if d.DeviceGroupID != nil {
			groupName = groupNameByID[*d.DeviceGroupID]
		}

		var connectedTime int64
		if d.LastSeenAt != nil {
			connectedTime = *d.LastSeenAt
		}

		var upTime int64
		if d.LastSeenAt != nil && d.Status == device.StatusOnline && now >= *d.LastSeenAt {
			upTime = now - *d.LastSeenAt
		}

		out = append(out, dto.Device{
			ID:              d.ID,
			Ddns:            d.Ddns,
			Status:          string(d.Status),
			ConnectedTime:   connectedTime,
			UpTime:          upTime,
			IP:              d.IP,
			Mac:             d.Mac,
			DeviceGroupID:   d.DeviceGroupID,
			DeviceGroupName: groupName,
		})
	}

	dto.Write(c, dto.Ok(traceID, dto.ListDevicesResp{
		Items:    out,
		Page:     1,
		PageSize: len(out),
		Total:    len(out),
	}))
}

// POST /api/devices/move-to-device-group
func (h *DeviceHandler) MoveToDeviceGroup(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	_ = middleware.MustPrincipal(c)

	if h.relationsRepo == nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
		return
	}

	var req dto.MoveDevicesToGroupReq
	if err := c.ShouldBindJSON(&req); err != nil || req.GroupID <= 0 {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{"field": "groupId"}))
		return
	}

	if err := h.relationsRepo.AddDevicesToGroup(c.Request.Context(), req.GroupID, req.DeviceIDs); err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", map[string]any{"detail": err.Error()}))
		return
	}

	dto.Write(c, dto.Ok(traceID, dto.MoveDevicesToGroupResp{}))
}
