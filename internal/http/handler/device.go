package handler

import (
	"errors"
	"io"
	"strconv"
	"strings"

	"rttys/internal/domain/access"
	"rttys/internal/domain/device"
	"rttys/internal/domain/devicelog"
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
	accessRepo    *sqlite.AccessRepo
	devLogSvc     *devicelog.Service
	// disconnect drops a device's live connection and its sessions.
	disconnect func(devid string)
}

func NewDeviceHandler(
	devSvc *device.Service,
	groupRepo *sqlite.GroupRepo,
	relationsRepo *sqlite.RelationsRepo,
	accessRepo *sqlite.AccessRepo,
	devLogSvc *devicelog.Service,
	disconnect func(devid string),
) *DeviceHandler {
	return &DeviceHandler{
		devSvc:        devSvc,
		groupRepo:     groupRepo,
		relationsRepo: relationsRepo,
		accessRepo:    accessRepo,
		devLogSvc:     devLogSvc,
		disconnect:    disconnect,
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

	emptyResp := func() {
		dto.Write(c, dto.Ok(traceID, dto.ListDevicesResp{
			Items: []dto.Device{}, Page: 1, PageSize: 0, Total: 0,
		}))
	}

	// Resolve visibility into a group-id restriction. nil = no restriction
	// (admin sees all devices, including ungrouped).
	var restrictGroups []int64
	if isAdmin {
		if filterGroupID != nil {
			restrictGroups = []int64{*filterGroupID}
		}
	} else {
		if h.groupRepo == nil {
			dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
			return
		}
		dgIDs, err := h.groupRepo.ListDeviceGroupIDsByUser(c.Request.Context(), p.UserID)
		if err != nil || len(dgIDs) == 0 {
			emptyResp()
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
				emptyResp()
				return
			}
			dgIDs = []int64{*filterGroupID}
		}
		restrictGroups = dgIDs
	}

	// Pagination params: pageSize omitted => return all (backward compatible).
	page := 1
	if v, err := strconv.Atoi(strings.TrimSpace(c.Query("page"))); err == nil && v > 0 {
		page = v
	}
	pageSize := 0
	if v, err := strconv.Atoi(strings.TrimSpace(c.Query("pageSize"))); err == nil && v > 0 {
		pageSize = v
	}

	// Status filter: only accept known values, ignore anything else.
	status := strings.ToLower(strings.TrimSpace(c.Query("status")))
	switch status {
	case "online", "offline", "disabled":
	default:
		status = ""
	}

	// All filtering/sorting/pagination is pushed to SQL. The search ':' is
	// stripped to match the colon-less MAC stored in the DB, mirroring the
	// previous client-side search behavior.
	items, total, err := h.devSvc.ListPaged(c.Request.Context(), device.ListQuery{
		RestrictGroups: restrictGroups,
		Search:         strings.ToLower(strings.ReplaceAll(strings.TrimSpace(c.Query("q")), ":", "")),
		Unassigned:     strings.EqualFold(strings.TrimSpace(c.Query("unassigned")), "true"),
		Status:         status,
		SortBy:         strings.TrimSpace(c.Query("sortBy")),
		Order:          strings.TrimSpace(c.Query("order")),
		Page:           page,
		PageSize:       pageSize,
	})
	if err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
		return
	}

	out := make([]dto.Device, 0, len(items))
	for _, d := range items {
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
			Model:           d.Model,
			DeviceGroupID:   d.DeviceGroupID,
			DeviceGroupName: d.GroupName,
		})
	}

	respPageSize := int(total)
	if pageSize > 0 {
		respPageSize = pageSize
	}
	dto.Write(c, dto.Ok(traceID, dto.ListDevicesResp{
		Items:    out,
		Page:     page,
		PageSize: respPageSize,
		Total:    int(total),
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

	devid, mac, err := h.accessRepo.RemoveDevice(c.Request.Context(), id)
	if errors.Is(err, access.ErrDeviceNotFound) {
		dto.Write(c, dto.Err(traceID, dto.CodeNotFound, "Device not found", nil))
		return
	}
	if err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
		return
	}

	// Drop the device only after its whitelist entry is gone, so it cannot
	// reconnect over mTLS in between.
	if h.disconnect != nil {
		h.disconnect(devid)
	}
	p := middleware.MustPrincipal(c)
	h.devLogSvc.RecordDeviceRemoved(c.Request.Context(), devid, mac, p.UserID, p.Username, c.ClientIP())

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
