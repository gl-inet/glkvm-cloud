package handler

import (
	"strconv"
	"strings"
	"time"

	"rttys/internal/domain/trusteddevice"
	"rttys/internal/domain/user"
	"rttys/internal/http/dto"
	"rttys/internal/http/middleware"
	"rttys/internal/pkg/totp"
	"rttys/internal/pkg/useragent"

	"github.com/gin-gonic/gin"
)

// PersonalHandler exposes /api/me/profile and /api/me/2fa/* endpoints
// for the logged-in user to view and edit their own account.
type PersonalHandler struct {
	userSvc          *user.Service
	trustedDeviceSvc trusteddevice.Repository
	issuer           string
}

func NewPersonalHandler(userSvc *user.Service, tdRepo trusteddevice.Repository, issuer string) *PersonalHandler {
	if issuer == "" {
		issuer = "GLKVM Cloud"
	}
	return &PersonalHandler{userSvc: userSvc, trustedDeviceSvc: tdRepo, issuer: issuer}
}

// GET /api/me/profile
func (h *PersonalHandler) GetProfile(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	p := middleware.MustPrincipal(c)

	u, err := h.userSvc.FindByID(c.Request.Context(), p.UserID)
	if err != nil || u == nil {
		dto.Write(c, dto.Err(traceID, dto.CodeNotFound, "User not found", nil))
		return
	}

	dto.Write(c, dto.Ok(traceID, dto.PersonalProfileResp{
		ID:               u.ID,
		Username:         u.Username,
		DisplayName:      u.Description,
		Email:            u.Email,
		Role:             string(u.Role),
		AuthProvider:     normalizedAuthProvider(u.AuthProvider),
		RegistrationTime: u.CreatedAt,
		LastLoginTime:    u.LastLoginAt,
		TotpEnabled:      u.TotpEnabled,
	}))
}

// PUT /api/me/profile
func (h *PersonalHandler) UpdateProfile(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	p := middleware.MustPrincipal(c)

	var req dto.UpdatePersonalProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", nil))
		return
	}

	if req.DisplayName != nil {
		desc := strings.TrimSpace(*req.DisplayName)
		if len(desc) > 200 {
			dto.Write(c, dto.Err(traceID, dto.CodeValidationFailed, "Display name too long", nil))
			return
		}
		if err := h.userSvc.UpdateDescription(c.Request.Context(), p.UserID, desc); err != nil {
			dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
			return
		}
	}

	dto.Write(c, dto.Ok(traceID, struct{}{}))
}

// POST /api/me/2fa/setup
//
// Generates a fresh TOTP secret and otpauth URL. The secret is NOT persisted
// until the client confirms by calling /api/me/2fa/enable with a valid code.
func (h *PersonalHandler) Setup2fa(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	p := middleware.MustPrincipal(c)

	if !isLocalAuthProvider(p.AuthProvider) {
		dto.Write(c, dto.Err(traceID, dto.CodeForbidden, "2FA is managed by your identity provider", nil))
		return
	}

	secret, url, err := totp.GenerateSecret(h.issuer, p.Username)
	if err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Failed to generate secret", nil))
		return
	}

	dto.Write(c, dto.Ok(traceID, dto.Setup2faResp{Secret: secret, OtpauthURL: url}))
}

// POST /api/me/2fa/enable
func (h *PersonalHandler) Enable2fa(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	p := middleware.MustPrincipal(c)

	if !isLocalAuthProvider(p.AuthProvider) {
		dto.Write(c, dto.Err(traceID, dto.CodeForbidden, "2FA is managed by your identity provider", nil))
		return
	}

	var req dto.Enable2faReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Secret == "" || req.Code == "" {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", nil))
		return
	}

	if !totp.Verify(req.Secret, req.Code) {
		dto.Write(c, dto.Err(traceID, dto.CodeValidationFailed, "Invalid verification code", nil))
		return
	}

	if err := h.userSvc.SetTotp(c.Request.Context(), p.UserID, req.Secret, true); err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
		return
	}

	dto.Write(c, dto.Ok(traceID, struct{}{}))
}

// POST /api/me/2fa/disable
//
// Requires a current valid TOTP code. After disabling, all trusted-device
// records for this user are revoked.
func (h *PersonalHandler) Disable2fa(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	p := middleware.MustPrincipal(c)

	if !isLocalAuthProvider(p.AuthProvider) {
		dto.Write(c, dto.Err(traceID, dto.CodeForbidden, "2FA is managed by your identity provider", nil))
		return
	}

	var req dto.Disable2faReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Code == "" {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", nil))
		return
	}

	u, err := h.userSvc.FindByID(c.Request.Context(), p.UserID)
	if err != nil || u == nil {
		dto.Write(c, dto.Err(traceID, dto.CodeNotFound, "User not found", nil))
		return
	}
	if !u.TotpEnabled || u.TotpSecret == "" {
		dto.Write(c, dto.Err(traceID, dto.CodeValidationFailed, "2FA is not enabled", nil))
		return
	}
	if !totp.Verify(u.TotpSecret, req.Code) {
		dto.Write(c, dto.Err(traceID, dto.CodeValidationFailed, "Invalid verification code", nil))
		return
	}

	if err := h.userSvc.SetTotp(c.Request.Context(), p.UserID, "", false); err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
		return
	}
	if h.trustedDeviceSvc != nil {
		_ = h.trustedDeviceSvc.DeleteByUserID(c.Request.Context(), p.UserID)
	}

	dto.Write(c, dto.Ok(traceID, struct{}{}))
}

// GET /api/me/2fa/trusted-devices
func (h *PersonalHandler) ListTrustedDevices(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	p := middleware.MustPrincipal(c)

	if h.trustedDeviceSvc == nil {
		dto.Write(c, dto.Ok(traceID, dto.ListTrustedDevicesResp{Items: []dto.TrustedDevice{}}))
		return
	}

	// Lazy-clean expired records so the list never shows stale entries.
	_ = h.trustedDeviceSvc.DeleteExpired(c.Request.Context(), time.Now().Unix())

	rows, err := h.trustedDeviceSvc.ListByUserID(c.Request.Context(), p.UserID)
	if err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
		return
	}

	out := make([]dto.TrustedDevice, 0, len(rows))
	for _, r := range rows {
		// Re-parse the stored device name on every read so legacy records
		// (which contain the raw User-Agent header) get rendered with the same
		// short label as freshly-issued ones — no DB migration needed.
		out = append(out, dto.TrustedDevice{
			ID:         r.ID,
			DeviceName: useragent.Friendly(r.DeviceName),
			IP:         r.IP,
			CreatedAt:  r.CreatedAt,
			LastUsedAt: r.LastUsedAt,
			ExpiresAt:  r.ExpiresAt,
		})
	}
	dto.Write(c, dto.Ok(traceID, dto.ListTrustedDevicesResp{Items: out}))
}

// DELETE /api/me/2fa/trusted-devices/:id
func (h *PersonalHandler) RevokeTrustedDevice(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	p := middleware.MustPrincipal(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", nil))
		return
	}

	if h.trustedDeviceSvc != nil {
		if err := h.trustedDeviceSvc.Delete(c.Request.Context(), id, p.UserID); err != nil {
			dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
			return
		}
	}

	dto.Write(c, dto.Ok(traceID, struct{}{}))
}

func isLocalAuthProvider(p string) bool {
	return p == "" || p == "local"
}

func normalizedAuthProvider(p string) string {
	if p == "" {
		return "local"
	}
	return p
}
