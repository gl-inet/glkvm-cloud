package handler

import (
    "rttys/internal/http/dto"
    "rttys/internal/http/middleware"
    "rttys/internal/store/sqlite"

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

    items, err := h.groupRepo.ListUserGroupsVisibleToUser(c.Request.Context(), p.UserID, string(p.Role) == "admin")
    if err != nil {
        dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
        return
    }

    out := make([]dto.UserGroup, 0, len(items))
    for _, it := range items {
        out = append(out, dto.UserGroup{ID: it.ID, Name: it.Name, Description: it.Description})
    }
    dto.Write(c, dto.Ok(traceID, dto.ListUserGroupsResp{Items: out}))
}
