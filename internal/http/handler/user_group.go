package handler

import (
    "rttys/internal/http/dto"
    "rttys/internal/http/middleware"
    "rttys/internal/store/sqlite"
    "strconv"
    "strings"

    "github.com/gin-gonic/gin"
)

type UserGroupHandler struct {
    groupRepo *sqlite.GroupRepo
}

func NewUserGroupHandler(groupRepo *sqlite.GroupRepo) *UserGroupHandler {
    return &UserGroupHandler{groupRepo: groupRepo}
}

// GET /api/user-groups
func (h *UserGroupHandler) ListUserGroups(c *gin.Context) {
    traceID := middleware.GetTraceID(c)
    p := middleware.MustPrincipal(c)

    items, err := h.groupRepo.ListUserGroupDetails(c.Request.Context(), p.UserID, string(p.Role) == "admin")
    if err != nil {
        dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
        return
    }

    out := make([]dto.UserGroup, 0, len(items))
    for _, it := range items {
        deviceGroups := make([]dto.UserGroupDeviceGroupRef, 0, len(it.DeviceGroups))
        for _, dg := range it.DeviceGroups {
            deviceGroups = append(deviceGroups, dto.UserGroupDeviceGroupRef{
                DeviceGroupID:   dg.ID,
                DeviceGroupName: dg.Name,
            })
        }
        out = append(out, dto.UserGroup{
            ID:              it.ID,
            UserGroup:       it.Name,
            Description:     it.Description,
            UserCount:       it.UserCount,
            DeviceGroupList: deviceGroups,
        })
    }
    dto.Write(c, dto.Ok(traceID, dto.ListUserGroupsResp{Items: out}))
}

// GET /api/user-groups/options
func (h *UserGroupHandler) ListOptions(c *gin.Context) {
    traceID := middleware.GetTraceID(c)
    p := middleware.MustPrincipal(c)

    items, err := h.groupRepo.ListUserGroupsVisibleToUser(c.Request.Context(), p.UserID, string(p.Role) == "admin")
    if err != nil {
        dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
        return
    }

    out := make([]dto.UserGroupOption, 0, len(items))
    for _, it := range items {
        out = append(out, dto.UserGroupOption{UserGroupID: it.ID, Name: it.Name})
    }
    dto.Write(c, dto.Ok(traceID, dto.ListUserGroupOptionsResp{Items: out}))
}

func (h *UserGroupHandler) Create(c *gin.Context) {
    traceID := middleware.GetTraceID(c)

    var req dto.CreateUserGroupReq
    if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
        dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{"field": "name"}))
        return
    }

    id, err := h.groupRepo.CreateUserGroup(c.Request.Context(), req.Name, req.Description)
    if err != nil {
        if strings.Contains(strings.ToLower(err.Error()), "unique") {
            dto.Write(c, dto.Err(traceID, dto.CodeConflict, "Name already exists", nil))
            return
        }
        dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
        return
    }

    dto.Write(c, dto.Ok(traceID, dto.CreateUserGroupResp{ID: id}))
}

func (h *UserGroupHandler) Update(c *gin.Context) {
    traceID := middleware.GetTraceID(c)

    id, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil || id <= 0 {
        dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{"field": "id"}))
        return
    }

    var req dto.UpdateUserGroupReq
    if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
        dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{"field": "name"}))
        return
    }

    if err := h.groupRepo.UpdateUserGroup(c.Request.Context(), id, req.Name, req.Description); err != nil {
        if strings.Contains(strings.ToLower(err.Error()), "unique") {
            dto.Write(c, dto.Err(traceID, dto.CodeConflict, "Name already exists", nil))
            return
        }
        dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
        return
    }

    dto.Write(c, dto.Ok(traceID, struct{}{}))
}

func (h *UserGroupHandler) Delete(c *gin.Context) {
    traceID := middleware.GetTraceID(c)

    id, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil || id <= 0 {
        dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{"field": "id"}))
        return
    }

    if err := h.groupRepo.DeleteUserGroup(c.Request.Context(), id); err != nil {
        dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
        return
    }
    dto.Write(c, dto.Ok(traceID, dto.DeleteUserGroupResp{}))
}
