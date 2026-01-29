package handler

import (
	"rttys/internal/domain/devicegroup"
	"rttys/internal/http/dto"
	"rttys/internal/http/middleware"
	"rttys/internal/store/sqlite"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type DeviceGroupHandler struct {
	groupRepo     *sqlite.GroupRepo
	relationsRepo *sqlite.RelationsRepo
}

func NewDeviceGroupHandler(groupRepo *sqlite.GroupRepo, relationsRepo *sqlite.RelationsRepo) *DeviceGroupHandler {
	return &DeviceGroupHandler{groupRepo: groupRepo, relationsRepo: relationsRepo}
}

// GET /api/device-groups
func (h *DeviceGroupHandler) ListDeviceGroups(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	p := middleware.MustPrincipal(c)

	items, err := h.groupRepo.ListDeviceGroupDetails(c.Request.Context(), p.UserID, string(p.Role) == "admin")
	if err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
		return
	}

	out := make([]dto.DeviceGroup, 0, len(items))
	for _, it := range items {
		userGroups := make([]dto.DeviceGroupUserGroupRef, 0, len(it.UserGroups))
		for _, ug := range it.UserGroups {
			userGroups = append(userGroups, dto.DeviceGroupUserGroupRef{
				UserGroupID:   ug.ID,
				UserGroupName: ug.Name,
			})
		}
		out = append(out, dto.DeviceGroup{
			ID:            it.ID,
			Name:          it.Name,
			DeviceCount:   it.DeviceCount,
			Description:   it.Description,
			UserGroupList: userGroups,
		})
	}
	dto.Write(c, dto.Ok(traceID, dto.ListDeviceGroupsResp{Items: out}))
}

var _ devicegroup.DeviceGroup

// GET /api/device-groups/options
func (h *DeviceGroupHandler) ListOptions(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	p := middleware.MustPrincipal(c)

	items, err := h.groupRepo.ListDeviceGroupsVisibleToUser(c.Request.Context(), p.UserID, string(p.Role) == "admin")
	if err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
		return
	}

	out := make([]dto.DeviceGroupOption, 0, len(items))
	for _, it := range items {
		out = append(out, dto.DeviceGroupOption{GroupID: it.ID, Name: it.Name})
	}
	dto.Write(c, dto.Ok(traceID, dto.ListDeviceGroupOptionsResp{Items: out}))
}

func (h *DeviceGroupHandler) Create(c *gin.Context) {
	traceID := middleware.GetTraceID(c)

	var req dto.CreateDeviceGroupReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{"field": "name"}))
		return
	}

	id, err := h.groupRepo.CreateDeviceGroup(c.Request.Context(), req.Name, req.Description)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			dto.Write(c, dto.Err(traceID, dto.CodeConflict, "Name already exists", nil))
			return
		}
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
		return
	}

	if h.relationsRepo != nil {
		if err := h.relationsRepo.SetDeviceGroupUserGroups(c.Request.Context(), id, req.UserGroupIDs); err != nil {
			_ = h.groupRepo.DeleteDeviceGroup(c.Request.Context(), id)
			dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", map[string]any{"detail": err.Error()}))
			return
		}
		if err := h.relationsRepo.AddDevicesToGroup(c.Request.Context(), id, req.DeviceIDs); err != nil {
			_ = h.groupRepo.DeleteDeviceGroup(c.Request.Context(), id)
			dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", map[string]any{"detail": err.Error()}))
			return
		}
	}

	dto.Write(c, dto.Ok(traceID, dto.CreateDeviceGroupResp{ID: id}))
}

func (h *DeviceGroupHandler) Update(c *gin.Context) {
	traceID := middleware.GetTraceID(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{"field": "id"}))
		return
	}

	var req dto.UpdateDeviceGroupReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{"field": "name"}))
		return
	}

	if err := h.groupRepo.UpdateDeviceGroup(c.Request.Context(), id, req.Name, req.Description); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			dto.Write(c, dto.Err(traceID, dto.CodeConflict, "Name already exists", nil))
			return
		}
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
		return
	}

	if h.relationsRepo != nil {
		if err := h.relationsRepo.SetDeviceGroupUserGroups(c.Request.Context(), id, req.UserGroupIDs); err != nil {
			dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", map[string]any{"detail": err.Error()}))
			return
		}
	}

	dto.Write(c, dto.Ok(traceID, struct{}{}))
}

func (h *DeviceGroupHandler) Delete(c *gin.Context) {
	traceID := middleware.GetTraceID(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{"field": "id"}))
		return
	}

	if err := h.groupRepo.DeleteDeviceGroup(c.Request.Context(), id); err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
		return
	}
	dto.Write(c, dto.Ok(traceID, dto.DeleteDeviceGroupResp{}))
}

// POST /api/device-groups/:id/devices
func (h *DeviceGroupHandler) AddDevices(c *gin.Context) {
	traceID := middleware.GetTraceID(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{"field": "id"}))
		return
	}

	var req dto.ModifyDeviceGroupDevicesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", nil))
		return
	}

	if h.relationsRepo == nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
		return
	}

	if err := h.relationsRepo.AddDevicesToGroup(c.Request.Context(), id, req.DeviceIDs); err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", map[string]any{"detail": err.Error()}))
		return
	}

	dto.Write(c, dto.Ok(traceID, struct{}{}))
}

// DELETE /api/device-groups/:id/devices
func (h *DeviceGroupHandler) RemoveDevices(c *gin.Context) {
	traceID := middleware.GetTraceID(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{"field": "id"}))
		return
	}

	var req dto.ModifyDeviceGroupDevicesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", nil))
		return
	}

	if h.relationsRepo == nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
		return
	}

	if err := h.relationsRepo.RemoveDevicesFromGroup(c.Request.Context(), id, req.DeviceIDs); err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", map[string]any{"detail": err.Error()}))
		return
	}

	dto.Write(c, dto.Ok(traceID, struct{}{}))
}
