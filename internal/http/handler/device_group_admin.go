package handler

import (
	"context"
	"strconv"
	"strings"

	"rttys/internal/http/dto"
	"rttys/internal/http/middleware"

	"github.com/gin-gonic/gin"
)

type DeviceGroupAdminHandler struct {
	create func(ctx context.Context, name, description string) (int64, error)
	update func(ctx context.Context, id int64, name, description string) error
	delete func(ctx context.Context, id int64) error
}

func NewDeviceGroupAdminHandler(
	create func(ctx context.Context, name, description string) (int64, error),
	update func(ctx context.Context, id int64, name, description string) error,
	del func(ctx context.Context, id int64) error,
) *DeviceGroupAdminHandler {
	return &DeviceGroupAdminHandler{create: create, update: update, delete: del}
}

func (h *DeviceGroupAdminHandler) Create(c *gin.Context) {
	traceID := middleware.GetTraceID(c)

	var req dto.CreateDeviceGroupReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{"field": "name"}))
		return
	}

	id, err := h.create(c.Request.Context(), req.Name, req.Description)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			dto.Write(c, dto.Err(traceID, dto.CodeConflict, "Name already exists", nil))
			return
		}
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
		return
	}

	dto.Write(c, dto.Ok(traceID, dto.CreateDeviceGroupResp{ID: id}))
}

func (h *DeviceGroupAdminHandler) Update(c *gin.Context) {
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

	if err := h.update(c.Request.Context(), id, req.Name, req.Description); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			dto.Write(c, dto.Err(traceID, dto.CodeConflict, "Name already exists", nil))
			return
		}
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
		return
	}

	dto.Write(c, dto.Ok(traceID, struct{}{}))
}

func (h *DeviceGroupAdminHandler) Delete(c *gin.Context) {
	traceID := middleware.GetTraceID(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{"field": "id"}))
		return
	}

	if err := h.delete(c.Request.Context(), id); err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
		return
	}
	dto.Write(c, dto.Ok(traceID, dto.DeleteDeviceGroupResp{}))
}
