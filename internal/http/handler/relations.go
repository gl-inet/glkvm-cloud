package handler

import (
    "strconv"

    "rttys/internal/http/dto"
    "rttys/internal/http/middleware"
    "rttys/internal/store/sqlite"

    "github.com/gin-gonic/gin"
)

type RelationsHandler struct{ repo *sqlite.RelationsRepo }

func NewRelationsHandler(repo *sqlite.RelationsRepo) *RelationsHandler {
    return &RelationsHandler{repo: repo}
}

func (h *RelationsHandler) SetUserGroups(c *gin.Context) {
    traceID := middleware.GetTraceID(c)
    _ = middleware.MustPrincipal(c)

    userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil || userID <= 0 {
        dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{"field": "id"}))
        return
    }

    var req dto.SetUserGroupsReq
    if err := c.ShouldBindJSON(&req); err != nil {
        dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", nil))
        return
    }

    if err := h.repo.SetUserGroups(c.Request.Context(), userID, req.GroupIDs); err != nil {
        dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", map[string]any{"detail": err.Error()}))
        return
    }

    dto.Write(c, dto.Ok(traceID, dto.SetUserGroupsResp{UserID: userID, GroupIDs: req.GroupIDs}))
}

func (h *RelationsHandler) SetUserGroupDeviceGroups(c *gin.Context) {
    traceID := middleware.GetTraceID(c)
    _ = middleware.MustPrincipal(c)

    ugID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil || ugID <= 0 {
        dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{"field": "id"}))
        return
    }

    var req dto.SetUserGroupDeviceGroupsReq
    if err := c.ShouldBindJSON(&req); err != nil {
        dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", nil))
        return
    }

    if err := h.repo.SetUserGroupDeviceGroups(c.Request.Context(), ugID, req.DeviceGroupIDs); err != nil {
        dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", map[string]any{"detail": err.Error()}))
        return
    }

    dto.Write(c, dto.Ok(traceID, dto.SetUserGroupDeviceGroupsResp{UserGroupID: ugID, DeviceGroupIDs: req.DeviceGroupIDs}))
}

func (h *RelationsHandler) SetDeviceGroupDevices(c *gin.Context) {
    traceID := middleware.GetTraceID(c)
    _ = middleware.MustPrincipal(c)

    dgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil || dgID <= 0 {
        dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{"field": "id"}))
        return
    }

    var req dto.SetDeviceGroupDevicesReq
    if err := c.ShouldBindJSON(&req); err != nil {
        dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", nil))
        return
    }

    if err := h.repo.SetDeviceGroupDevices(c.Request.Context(), dgID, req.DeviceIDs); err != nil {
        dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", map[string]any{"detail": err.Error()}))
        return
    }

    dto.Write(c, dto.Ok(traceID, dto.SetDeviceGroupDevicesResp{DeviceGroupID: dgID, DeviceIDs: req.DeviceIDs}))
}
