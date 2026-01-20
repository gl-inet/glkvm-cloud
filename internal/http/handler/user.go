package handler

import (
    "strconv"
    "strings"

    "rttys/internal/domain/user"
    "rttys/internal/http/dto"
    "rttys/internal/http/middleware"

    "github.com/gin-gonic/gin"
)

type UserHandler struct {
    userSvc *user.Service
}

func NewUserHandler(userSvc *user.Service) *UserHandler {
    return &UserHandler{userSvc: userSvc}
}

func (h *UserHandler) ListUsers(c *gin.Context) {
    traceID := middleware.GetTraceID(c)

    items, err := h.userSvc.List(c.Request.Context())
    if err != nil {
        dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
        return
    }

    out := make([]dto.User, 0, len(items))
    for _, u := range items {
        out = append(out, dto.User{
            ID:          u.ID,
            Email:       u.Email,
            DisplayName: u.DisplayName,
            Role:        string(u.Role),
            Status:      string(u.Status),
        })
    }
    dto.Write(c, dto.Ok(traceID, dto.ListUsersResp{Items: out}))
}

func (h *UserHandler) CreateUser(c *gin.Context) {
    traceID := middleware.GetTraceID(c)

    var req dto.CreateUserReq
    if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" || req.Password == "" {
        dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{
            "field": "email/password",
        }))
        return
    }

    if req.Role == "" {
        req.Role = "user"
    }
    if req.Status == "" {
        req.Status = "active"
    }

    id, err := h.userSvc.CreateUser(c.Request.Context(), req.Email, req.DisplayName, req.Password, req.Role, req.Status)
    if err != nil {
        // best-effort conflict detection
        if strings.Contains(strings.ToLower(err.Error()), "unique") {
            dto.Write(c, dto.Err(traceID, dto.CodeConflict, "Email already exists", nil))
            return
        }
        dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
        return
    }

    dto.Write(c, dto.Ok(traceID, dto.CreateUserResp{ID: id}))
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

    if err := h.userSvc.UpdateUser(c.Request.Context(), id, req.Email, req.DisplayName, req.Password, req.Role, req.Status); err != nil {
        if strings.Contains(strings.ToLower(err.Error()), "not found") {
            dto.Write(c, dto.Err(traceID, dto.CodeNotFound, "Not found", nil))
            return
        }
        if strings.Contains(strings.ToLower(err.Error()), "unique") {
            dto.Write(c, dto.Err(traceID, dto.CodeConflict, "Email already exists", nil))
            return
        }
        dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
        return
    }

    dto.Write(c, dto.Ok(traceID, struct{}{}))
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
    traceID := middleware.GetTraceID(c)

    id, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil || id <= 0 {
        dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{
            "field": "id",
        }))
        return
    }

    if err := h.userSvc.DeleteUser(c.Request.Context(), id); err != nil {
        dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
        return
    }
    dto.Write(c, dto.Ok(traceID, dto.DeleteUserResp{}))
}
