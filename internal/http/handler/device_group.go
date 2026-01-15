package handler

import (
    "rttys/internal/domain/devicegroup"
    "rttys/internal/http/dto"
    "rttys/internal/http/middleware"
    "rttys/internal/store/sqlite"

    "github.com/gin-gonic/gin"
)

type DeviceGroupHandler struct {
    groupRepo *sqlite.GroupRepo
}

func NewDeviceGroupHandler(groupRepo *sqlite.GroupRepo) *DeviceGroupHandler {
    return &DeviceGroupHandler{groupRepo: groupRepo}
}

// GET /api/device-groups
func (h *DeviceGroupHandler) ListDeviceGroups(c *gin.Context) {
    traceID := middleware.GetTraceID(c)
    p := middleware.MustPrincipal(c)

    items, err := h.groupRepo.ListDeviceGroupsVisibleToUser(c.Request.Context(), p.UserID, string(p.Role) == "admin")
    if err != nil {
        dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
        return
    }

    out := make([]dto.DeviceGroup, 0, len(items))
    for _, it := range items {
        out = append(out, dto.DeviceGroup{ID: it.ID, Name: it.Name, Description: it.Description})
    }
    dto.Write(c, dto.Ok(traceID, dto.ListDeviceGroupsResp{Items: out}))
}

var _ devicegroup.DeviceGroup
