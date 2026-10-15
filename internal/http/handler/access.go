package handler

import (
	"encoding/pem"
	"errors"
	"io"
	"net"
	"os"
	"strconv"
	"time"

	"rttys/internal/domain/access"
	"rttys/internal/http/dto"
	"rttys/internal/http/middleware"
	"rttys/internal/store/sqlite"
	"rttys/xconfig"

	"github.com/gin-gonic/gin"
)

// AccessHandler hands out registration codes, packed into access codes that
// devices paste to join over mTLS.
type AccessHandler struct {
	repo *sqlite.AccessRepo
	addr string // host:port devices dial for the mTLS port; "" = not configured
	ca   []byte // DER of the deployment CA
}

func NewAccessHandler(repo *sqlite.AccessRepo, cfg *xconfig.Config) *AccessHandler {
	h := &AccessHandler{repo: repo}

	host := cfg.DeviceAccessHost
	if host == "" {
		host = cfg.WebrtcIP
	}
	if _, port, err := net.SplitHostPort(cfg.AddrDevMTLS); err == nil && host != "" && port != "" {
		h.addr = net.JoinHostPort(host, port)
	}
	if b, err := os.ReadFile(cfg.DeviceCAFile); err == nil {
		if blk, _ := pem.Decode(b); blk != nil {
			h.ca = blk.Bytes
		}
	}
	return h
}

// POST /api/registration-codes
func (h *AccessHandler) CreateCodes(c *gin.Context) {
	traceID := middleware.GetTraceID(c)
	p := middleware.MustPrincipal(c)

	var req dto.CreateRegistrationCodesReq
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", nil))
		return
	}
	if req.Count == 0 {
		req.Count = 1
	}
	if req.Count < 1 || req.Count > access.MaxCodesPerRequest {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{
			"field": "count", "max": access.MaxCodesPerRequest,
		}))
		return
	}
	if h.addr == "" || len(h.ca) == 0 {
		dto.Write(c, dto.Err(traceID, dto.CodeValidationFailed,
			"Device access address is not configured: set GLKVM_ACCESS_IP or DEVICE_ACCESS_HOST", nil))
		return
	}

	codes := make([]string, req.Count)
	hashes := make([]string, req.Count)
	for i := range codes {
		code, err := access.NewCode()
		if err != nil {
			dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
			return
		}
		codes[i], hashes[i] = code, access.HashCode(code)
	}

	now := time.Now().Unix()
	expiresAt := now + int64(access.CodeTTL/time.Second)
	ids, err := h.repo.CreateCodes(c.Request.Context(), hashes, p.UserID, now, expiresAt)
	if err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
		return
	}

	out := make([]dto.CreatedRegistrationCode, len(ids))
	for i, id := range ids {
		out[i] = dto.CreatedRegistrationCode{
			ID:         id,
			AccessCode: access.AccessCode{Addr: h.addr, CA: h.ca, Code: codes[i]}.String(),
			ExpiresAt:  expiresAt,
		}
	}
	dto.Write(c, dto.Ok(traceID, dto.CreateRegistrationCodesResp{Items: out}))
}

// GET /api/registration-codes
func (h *AccessHandler) ListCodes(c *gin.Context) {
	traceID := middleware.GetTraceID(c)

	codes, err := h.repo.ListCodes(c.Request.Context(), 200)
	if err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
		return
	}
	now := time.Now().Unix()
	out := make([]dto.RegistrationCode, 0, len(codes))
	for _, rc := range codes {
		out = append(out, dto.RegistrationCode{
			ID:        rc.ID,
			Status:    string(rc.Status(now)),
			CreatedAt: rc.CreatedAt,
			ExpiresAt: rc.ExpiresAt,
			UsedAt:    rc.UsedAt,
			UsedMac:   rc.UsedMAC,
			Devid:     rc.Devid,
			RevokedAt: rc.RevokedAt,
		})
	}
	dto.Write(c, dto.Ok(traceID, dto.ListRegistrationCodesResp{Items: out}))
}

// DELETE /api/registration-codes/:id
func (h *AccessHandler) RevokeCode(c *gin.Context) {
	traceID := middleware.GetTraceID(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		dto.Write(c, dto.Err(traceID, dto.CodeInvalidArgument, "Invalid argument", map[string]any{"field": "id"}))
		return
	}
	switch err := h.repo.RevokeCode(c.Request.Context(), id, time.Now().Unix()); {
	case errors.Is(err, access.ErrCodeNotFound):
		dto.Write(c, dto.Err(traceID, dto.CodeNotFound, "Registration code not found", nil))
	case errors.Is(err, access.ErrCodeNotRevocable):
		dto.Write(c, dto.Err(traceID, dto.CodeConflict, "Registration code is already used, revoked or expired", nil))
	case err != nil:
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
	default:
		dto.Write(c, dto.Ok(traceID, struct{}{}))
	}
}
