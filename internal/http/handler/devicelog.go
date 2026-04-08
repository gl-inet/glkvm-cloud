package handler

import (
	"strconv"
	"strings"

	"rttys/internal/domain/devicelog"
	"rttys/internal/http/dto"
	"rttys/internal/http/middleware"

	"github.com/gin-gonic/gin"
)

// DeviceLogHandler exposes /api/device-event-logs for admins.
type DeviceLogHandler struct {
	svc *devicelog.Service
}

func NewDeviceLogHandler(svc *devicelog.Service) *DeviceLogHandler {
	return &DeviceLogHandler{svc: svc}
}

// GET /api/device-event-logs?mac=&types=device_online,remote_ssh&from=&to=&page=1&pageSize=20
func (h *DeviceLogHandler) List(c *gin.Context) {
	traceID := middleware.GetTraceID(c)

	if h.svc == nil {
		dto.Write(c, dto.Ok(traceID, dto.ListDeviceEventLogsResp{
			Items: []dto.DeviceEventLog{}, Total: 0, Page: 1, PageSize: 20,
		}))
		return
	}

	q := devicelog.Query{
		Mac:        strings.TrimSpace(c.Query("mac")),
		EventTypes: parseEventTypes(c.Query("types")),
		From:       parseInt64(c.Query("from")),
		To:         parseInt64(c.Query("to")),
		Page:       parseInt(c.Query("page")),
		PageSize:   parseInt(c.Query("pageSize")),
	}

	rows, total, err := h.svc.Query(c.Request.Context(), q)
	if err != nil {
		dto.Write(c, dto.Err(traceID, dto.CodeInternalError, "Internal error", nil))
		return
	}

	out := make([]dto.DeviceEventLog, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.DeviceEventLog{
			ID:        r.ID,
			DeviceMac: r.DeviceMac,
			EventType: string(r.EventType),
			ActorName: r.ActorName,
			ClientIP:  r.ClientIP,
			Detail:    r.Detail,
			CreatedAt: r.CreatedAt,
			EndedAt:   r.EndedAt,
		})
	}

	page := q.Page
	if page < 1 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	dto.Write(c, dto.Ok(traceID, dto.ListDeviceEventLogsResp{
		Items:    out,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}))
}

func parseEventTypes(raw string) []devicelog.EventType {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]devicelog.EventType, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		switch devicelog.EventType(p) {
		case devicelog.EventDeviceOnline,
			devicelog.EventDeviceOffline,
			devicelog.EventRemoteSSH,
			devicelog.EventRemoteWeb,
			devicelog.EventRemoteControl:
			out = append(out, devicelog.EventType(p))
		}
	}
	return out
}

func parseInt64(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

func parseInt(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, _ := strconv.Atoi(s)
	return v
}
