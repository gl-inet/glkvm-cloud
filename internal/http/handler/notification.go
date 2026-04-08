package handler

import (
	"rttys/internal/domain/notification"
	"rttys/internal/http/dto"
	"rttys/internal/http/middleware"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	svc *notification.Service
}

func NewNotificationHandler(svc *notification.Service) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

// GET /api/notification/smtp
func (h *NotificationHandler) GetSMTPConfig(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	cfg, err := h.svc.GetSMTPConfig(c.Request.Context())
	if err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Failed to load SMTP config", nil))
		return
	}
	dto.Write(c, dto.Ok(traceID, dto.SMTPConfigResp{
		Host:       cfg.Host,
		Port:       cfg.Port,
		Username:   cfg.Username,
		Password:   cfg.Password,
		FromEmail:  cfg.FromEmail,
		Encryption: cfg.Encryption,
		Enabled:    cfg.Enabled,
		UpdatedAt:  cfg.UpdatedAt,
	}))
}

// PUT /api/notification/smtp
func (h *NotificationHandler) SaveSMTPConfig(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	var req dto.SMTPConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, err.Error(), nil))
		return
	}
	cfg := &notification.SMTPConfig{
		Host:       req.Host,
		Port:       req.Port,
		Username:   req.Username,
		Password:   req.Password,
		FromEmail:  req.FromEmail,
		Encryption: req.Encryption,
		Enabled:    req.Enabled,
	}
	if err := h.svc.SaveSMTPConfig(c.Request.Context(), cfg); err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Failed to save SMTP config", nil))
		return
	}
	dto.Write(c, dto.Ok(traceID, dto.SMTPConfigResp{
		Host:       cfg.Host,
		Port:       cfg.Port,
		Username:   cfg.Username,
		Password:   cfg.Password,
		FromEmail:  cfg.FromEmail,
		Encryption: cfg.Encryption,
		Enabled:    cfg.Enabled,
		UpdatedAt:  cfg.UpdatedAt,
	}))
}

// POST /api/notification/smtp/test
func (h *NotificationHandler) TestSMTP(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	var req dto.SMTPTestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, err.Error(), nil))
		return
	}
	if err := h.svc.TestSMTP(c.Request.Context(), req.Email); err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, err.Error(), nil))
		return
	}
	dto.Write(c, dto.Ok(traceID, gin.H{"message": "Test email sent successfully"}))
}

// GET /api/notification/rules
func (h *NotificationHandler) GetNotifyRules(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	rules, err := h.svc.GetNotifyRules(c.Request.Context())
	if err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Failed to load rules", nil))
		return
	}
	dto.Write(c, dto.Ok(traceID, dto.NotifyRulesResp{
		DeviceOnline:  rules.DeviceOnline,
		DeviceOffline: rules.DeviceOffline,
		RemoteAccess:  rules.RemoteAccess,
		UpdatedAt:     rules.UpdatedAt,
	}))
}

// PUT /api/notification/rules
func (h *NotificationHandler) SaveNotifyRules(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	var req dto.NotifyRulesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, err.Error(), nil))
		return
	}
	rules := &notification.NotifyRules{
		DeviceOnline:  req.DeviceOnline,
		DeviceOffline: req.DeviceOffline,
		RemoteAccess:  req.RemoteAccess,
	}
	if err := h.svc.SaveNotifyRules(c.Request.Context(), rules); err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Failed to save rules", nil))
		return
	}
	dto.Write(c, dto.Ok(traceID, dto.NotifyRulesResp{
		DeviceOnline:  rules.DeviceOnline,
		DeviceOffline: rules.DeviceOffline,
		RemoteAccess:  rules.RemoteAccess,
		UpdatedAt:     rules.UpdatedAt,
	}))
}

// GET /api/notification/recipients
func (h *NotificationHandler) ListRecipients(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	list, err := h.svc.ListRecipients(c.Request.Context())
	if err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Failed to list recipients", nil))
		return
	}
	items := make([]dto.RecipientResp, 0, len(list))
	for _, r := range list {
		items = append(items, dto.RecipientResp{
			ID:        r.ID,
			Email:     r.Email,
			CreatedAt: r.CreatedAt,
		})
	}
	dto.Write(c, dto.Ok(traceID, dto.ListRecipientsResp{Items: items}))
}

// POST /api/notification/recipients
func (h *NotificationHandler) AddRecipient(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	var req dto.AddRecipientReq
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, err.Error(), nil))
		return
	}
	r, err := h.svc.AddRecipient(c.Request.Context(), req.Email)
	if err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Failed to add recipient", nil))
		return
	}
	dto.Write(c, dto.Ok(traceID, dto.RecipientResp{
		ID:        r.ID,
		Email:     r.Email,
		CreatedAt: r.CreatedAt,
	}))
}

// DELETE /api/notification/recipients/:id
func (h *NotificationHandler) RemoveRecipient(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	id := parseInt64(c.Param("id"))
	if id <= 0 {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid ID", nil))
		return
	}
	if err := h.svc.RemoveRecipient(c.Request.Context(), id); err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Failed to remove recipient", nil))
		return
	}
	dto.Write(c, dto.Ok(traceID, gin.H{"message": "Recipient removed"}))
}
