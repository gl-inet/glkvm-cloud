package handler

import (
    "strconv"
    "strings"

    "rttys/internal/domain/user"
    "rttys/internal/http/dto"
    "rttys/internal/http/middleware"
    "rttys/internal/store/sqlite"

    "github.com/gin-gonic/gin"
)

type UserHandler struct {
    userSvc       *user.Service
    groupRepo     *sqlite.GroupRepo
    relationsRepo *sqlite.RelationsRepo
}

func NewUserHandler(userSvc *user.Service, groupRepo *sqlite.GroupRepo, relationsRepo *sqlite.RelationsRepo) *UserHandler {
    return &UserHandler{
        userSvc:       userSvc,
        groupRepo:     groupRepo,
        relationsRepo: relationsRepo,
    }
}

func (h *UserHandler) ListUsers(c *gin.Context) {
    traceID := middleware.GetTraceID(c)

    items, err := h.userSvc.List(c.Request.Context())
    if err != nil {
        dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
        return
    }

    userIDs := make([]int64, 0, len(items))
    for _, u := range items {
        userIDs = append(userIDs, u.ID)
    }
    var groupsByUserID map[int64][]sqlite.UserGroupBrief
    if h.groupRepo != nil {
        groupsByUserID, err = h.groupRepo.ListUserGroupsByUserIDs(c.Request.Context(), userIDs)
        if err != nil {
            dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
            return
        }
    }

    out := make([]dto.User, 0, len(items))
    for _, u := range items {
        groups := make([]dto.UserGroupRef, 0)
        if list, ok := groupsByUserID[u.ID]; ok {
            for _, g := range list {
                groups = append(groups, dto.UserGroupRef{
                    UserGroupID:   g.ID,
                    UserGroupName: g.Name,
                })
            }
        }
		out = append(out, dto.User{
			ID:           u.ID,
			Role:         string(u.Role),
			Username:     u.Username,
			Description:  u.Description,
			IsSystem:     u.IsSystem,
			UserGroupList: groups,
		})
	}
    dto.Write(c, dto.Ok(traceID, dto.ListUsersResp{Items: out}))
}

func (h *UserHandler) CreateUser(c *gin.Context) {
    traceID := middleware.GetTraceID(c)

    var req dto.CreateUserReq
    if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" {
        dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{
            "field": "username/password",
        }))
        return
    }
    if req.Repassword != "" && req.Repassword != req.Password {
        dto.Write(c, dto.Err(traceID, dto.CodeValidationFailed, "Passwords do not match", map[string]any{
            "field": "repassword",
        }))
        return
    }

    if req.Role == "" {
        req.Role = "user"
    }
    status := "active"

    id, err := h.userSvc.CreateUser(c.Request.Context(), req.Username, req.Description, req.Password, req.Role, status)
    if err != nil {
        // best-effort conflict detection
        if strings.Contains(strings.ToLower(err.Error()), "unique") {
            dto.Write(c, dto.Err(traceID, dto.CodeConflict, "Username already exists", nil))
            return
        }
        dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
        return
    }

    if h.relationsRepo != nil {
        if err := h.relationsRepo.SetUserGroups(c.Request.Context(), id, req.UserGroupIDs); err != nil {
            _ = h.userSvc.DeleteUser(c.Request.Context(), id)
            dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", map[string]any{"detail": err.Error()}))
            return
        }
    }

    dto.Write(c, dto.Ok(traceID, dto.CreateUserResp{}))
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
    traceID := middleware.GetTraceID(c)

    id, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil || id <= 0 {
        dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{
            "field": "id",
        }))
        return
    }

    var req dto.UpdateUserReq
    if err := c.ShouldBindJSON(&req); err != nil {
        dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", nil))
        return
    }
    if req.Password != nil && req.Repassword != nil && *req.Password != *req.Repassword {
        dto.Write(c, dto.Err(traceID, dto.CodeValidationFailed, "Passwords do not match", map[string]any{
            "field": "repassword",
        }))
        return
    }

    if err := h.userSvc.UpdateUser(c.Request.Context(), id, req.Username, req.Description, req.Password, req.Role, nil); err != nil {
        if strings.Contains(strings.ToLower(err.Error()), "not found") {
            dto.Write(c, dto.Err(traceID, dto.CodeNotFound, "Not found", nil))
            return
        }
        if strings.Contains(strings.ToLower(err.Error()), "unique") {
            dto.Write(c, dto.Err(traceID, dto.CodeConflict, "Username already exists", nil))
            return
        }
        dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
        return
    }

    if h.relationsRepo != nil && req.UserGroupIDs != nil {
        if err := h.relationsRepo.SetUserGroups(c.Request.Context(), id, *req.UserGroupIDs); err != nil {
            dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", map[string]any{"detail": err.Error()}))
            return
        }
    }

    dto.Write(c, dto.Ok(traceID, struct{}{}))
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
    traceID := middleware.GetTraceID(c)
    p := middleware.MustPrincipal(c)

    id, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil || id <= 0 {
        dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{
            "field": "id",
        }))
        return
    }

    if id == p.UserID {
        dto.Write(c, dto.Err(traceID, dto.CodeForbidden, "Cannot delete your own account", nil))
        return
    }

    u, err := h.userSvc.FindByID(c.Request.Context(), id)
    if err != nil {
        dto.Write(c, dto.Err(traceID, dto.CodeNotFound, "Not found", nil))
        return
    }
    if u.IsSystem {
        dto.Write(c, dto.Err(traceID, dto.CodeForbidden, "System user cannot be deleted", nil))
        return
    }

    if err := h.userSvc.DeleteUser(c.Request.Context(), id); err != nil {
        dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
        return
    }
    dto.Write(c, dto.Ok(traceID, dto.DeleteUserResp{}))
}
