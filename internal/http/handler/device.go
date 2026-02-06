package handler

import (
	"errors"
	"io"
	"sort"
	"strconv"
	"strings"

	"rttys/internal/domain/device"
	"rttys/internal/domain/identity"
	"rttys/internal/http/dto"
	"rttys/internal/http/middleware"
	"rttys/internal/store/sqlite"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DeviceHandler struct {
	devSvc        *device.Service
	groupRepo     *sqlite.GroupRepo
	relationsRepo *sqlite.RelationsRepo
}

func NewDeviceHandler(
	devSvc *device.Service,
	groupRepo *sqlite.GroupRepo,
	relationsRepo *sqlite.RelationsRepo,
) *DeviceHandler {
	return &DeviceHandler{
		devSvc:        devSvc,
		groupRepo:     groupRepo,
		relationsRepo: relationsRepo,
	}
}

// GET /api/devices
func (h *DeviceHandler) ListDevices(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	p := middleware.MustPrincipal(c)

	var filterGroupID *int64
	if raw := strings.TrimSpace(c.Query("groupId")); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{
				"field": "groupId",
			}))
			return
		}
		filterGroupID = &id
	}

	isAdmin := p.Role == identity.RoleAdmin
	var items []device.Device
	if isAdmin {
		var err error
		if filterGroupID != nil {
			items, err = h.devSvc.ListByDeviceGroupIDs(c.Request.Context(), []int64{*filterGroupID})
		} else {
			items, err = h.devSvc.ListVisible(c.Request.Context(), p.Role, p.UserID)
		}
		if err != nil {
			dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
			return
		}
	} else {
		if h.groupRepo == nil {
			dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
			return
		}
		dgIDs, err := h.groupRepo.ListDeviceGroupIDsByUser(c.Request.Context(), p.UserID)
		if err != nil || len(dgIDs) == 0 {
			dto.Write(c, dto.Ok(traceID, dto.ListDevicesResp{
				Items:    []dto.Device{},
				Page:     1,
				PageSize: 0,
				Total:    0,
			}))
			return
		}
		if filterGroupID != nil {
			allowed := false
			for _, gid := range dgIDs {
				if gid == *filterGroupID {
					allowed = true
					break
				}
			}
			if !allowed {
				dto.Write(c, dto.Ok(traceID, dto.ListDevicesResp{
					Items:    []dto.Device{},
					Page:     1,
					PageSize: 0,
					Total:    0,
				}))
				return
			}
			dgIDs = []int64{*filterGroupID}
		}
		items, err = h.devSvc.ListByDeviceGroupIDs(c.Request.Context(), dgIDs)
		if err != nil {
			dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
			return
		}
	}

	groupNameByID := map[int64]string{}
	if h.groupRepo != nil {
		groups, err := h.groupRepo.ListDeviceGroupsVisibleToUser(c.Request.Context(), p.UserID, isAdmin)
		if err != nil {
			dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
			return
		}
		for _, g := range groups {
			groupNameByID[g.ID] = g.Name
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		oi := items[i].Status == device.StatusOnline
		oj := items[j].Status == device.StatusOnline
		if oi != oj {
			return oi
		}
		return false
	})

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

		out = append(out, dto.Device{
			ID:              d.ID,
			Ddns:            d.Ddns,
			Status:          string(d.Status),
			ConnectedTime:   connectedTime,
			IP:              d.IP,
			Mac:             d.Mac,
			Description:     d.Description,
			Client:          d.Client,
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

type UpdateDeviceRequest struct {
	Description *string `json:"description"`
}

// PUT /api/devices/:id
func (h *DeviceHandler) UpdateDevice(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	idStr := strings.TrimSpace(c.Param("id"))
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{
			"field": "id",
		}))
		return
	}

	var req UpdateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{
			"field": "description",
			"error": err.Error(),
		}))
		return
	}

	db := sqlite.MustContainer().Gorm.WithContext(c.Request.Context())

	var row struct {
		Description string `gorm:"column:description"`
	}
	tx := db.Table("devices").Select("description").Where("id = ?", id).First(&row)
	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		dto.Write(c, dto.Err(traceID, dto.CodeNotFound, "Device not found", nil))
		return
	}
	if tx.Error != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", map[string]any{
			"detail": tx.Error.Error(),
		}))
		return
	}

	newDesc := row.Description
	if req.Description != nil {
		newDesc = *req.Description
	}

	if req.Description != nil {
		res := db.Exec(
			`UPDATE devices SET description=? WHERE id=?`,
			newDesc, id,
		)
		if res.Error != nil {
			dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", map[string]any{
				"detail": res.Error.Error(),
			}))
			return
		}
		if res.RowsAffected == 0 {
			dto.Write(c, dto.Err(traceID, dto.CodeNotFound, "Device not found", nil))
			return
		}
	}

	dto.Write(c, dto.Ok(traceID, struct{}{}))
}

// DELETE /api/devices/:id
func (h *DeviceHandler) DeleteDevice(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	idStr := strings.TrimSpace(c.Param("id"))
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{
			"field": "id",
		}))
		return
	}

	db := sqlite.MustContainer().Gorm.WithContext(c.Request.Context())
	res := db.Exec(`DELETE FROM devices WHERE id=?`, id)
	if res.Error != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", map[string]any{
			"detail": res.Error.Error(),
		}))
		return
	}
	if res.RowsAffected == 0 {
		dto.Write(c, dto.Err(traceID, dto.CodeNotFound, "Device not found", nil))
		return
	}

	dto.Write(c, dto.Ok(traceID, struct{}{}))
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
