// Package devicelog records four kinds of device events: online, offline,
// remote SSH session and remote Web session. Records are queryable by
// MAC, event type and time range.
//
// The service deliberately swallows errors so that logging never blocks the
// device runtime — failures are reported via the standard logger.
package devicelog

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// startupGraceWindow is how long after the service boots we silently drop
// device online/offline events. The goal is to avoid the reconnect storm
// that happens right after a server restart from filling the log with
// noise. SSH/Web events are user-initiated and never suppressed.
const startupGraceWindow = 60 * time.Second

// Detail field length cap to keep rows bounded against malicious input.
const maxDetailLen = 2000

type Service struct {
	repo        Repository
	startupTime time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, startupTime: time.Now()}
}

// normalizeMac strips colons and lowercases, matching the format in the
// devices table so that MAC-based searches work correctly.
func normalizeMac(mac string) string {
	return strings.ReplaceAll(strings.ToLower(mac), ":", "")
}

// inGracePeriod reports whether we are still inside the post-startup quiet
// window during which device on/off events are dropped.
func (s *Service) inGracePeriod() bool {
	return time.Since(s.startupTime) < startupGraceWindow
}

// RecordDeviceOnline records a device-online event. Dropped during the
// startup grace window.
func (s *Service) RecordDeviceOnline(ctx context.Context, deviceID, mac, ip string) {
	if s == nil || s.repo == nil {
		return
	}
	if s.inGracePeriod() {
		return
	}
	if _, err := s.repo.Create(ctx, &Log{
		DeviceID:  deviceID,
		DeviceMac: normalizeMac(mac),
		EventType: EventDeviceOnline,
		ClientIP:  ip,
		CreatedAt: time.Now().Unix(),
	}); err != nil {
		log.Warn().Err(err).Str("device", deviceID).Msg("devicelog: record online failed")
	}
}

// RecordDeviceOffline records a device-offline event. Dropped during the
// startup grace window.
func (s *Service) RecordDeviceOffline(ctx context.Context, deviceID, mac, ip string) {
	if s == nil || s.repo == nil {
		return
	}
	if s.inGracePeriod() {
		return
	}
	if _, err := s.repo.Create(ctx, &Log{
		DeviceID:  deviceID,
		DeviceMac: normalizeMac(mac),
		EventType: EventDeviceOffline,
		ClientIP:  ip,
		CreatedAt: time.Now().Unix(),
	}); err != nil {
		log.Warn().Err(err).Str("device", deviceID).Msg("devicelog: record offline failed")
	}
}

// StartRemoteSSHSession records the start of an SSH session and returns
// the row ID so the caller can later mark it ended via EndSession.
// Returns 0 if recording failed.
func (s *Service) StartRemoteSSHSession(ctx context.Context, deviceID, mac string, userID int64, userName, ip string) int64 {
	if s == nil || s.repo == nil {
		return 0
	}
	id, err := s.repo.Create(ctx, &Log{
		DeviceID:    deviceID,
		DeviceMac:   normalizeMac(mac),
		EventType:   EventRemoteSSH,
		ActorUserID: userID,
		ActorName:   userName,
		ClientIP:    ip,
		CreatedAt:   time.Now().Unix(),
	})
	if err != nil {
		log.Warn().Err(err).Str("device", deviceID).Msg("devicelog: start ssh session failed")
		return 0
	}
	return id
}

// StartRemoteWebSession records the start of a web-proxy session.
// addr/proto are stored as a small JSON detail blob.
func (s *Service) StartRemoteWebSession(ctx context.Context, deviceID, mac string, userID int64, userName, ip, addr, proto string) int64 {
	if s == nil || s.repo == nil {
		return 0
	}
	detail := encodeDetail(map[string]string{"addr": addr, "proto": proto})
	id, err := s.repo.Create(ctx, &Log{
		DeviceID:    deviceID,
		DeviceMac:   normalizeMac(mac),
		EventType:   EventRemoteWeb,
		ActorUserID: userID,
		ActorName:   userName,
		ClientIP:    ip,
		Detail:      detail,
		CreatedAt:   time.Now().Unix(),
	})
	if err != nil {
		log.Warn().Err(err).Str("device", deviceID).Msg("devicelog: start web session failed")
		return 0
	}
	return id
}

// StartRemoteControlSession records the start of a remote-control session
// (KVM web UI). No detail blob is stored for this event type.
func (s *Service) StartRemoteControlSession(ctx context.Context, deviceID, mac string, userID int64, userName, ip string) int64 {
	if s == nil || s.repo == nil {
		return 0
	}
	id, err := s.repo.Create(ctx, &Log{
		DeviceID:    deviceID,
		DeviceMac:   normalizeMac(mac),
		EventType:   EventRemoteControl,
		ActorUserID: userID,
		ActorName:   userName,
		ClientIP:    ip,
		CreatedAt:   time.Now().Unix(),
	})
	if err != nil {
		log.Warn().Err(err).Str("device", deviceID).Msg("devicelog: start control session failed")
		return 0
	}
	return id
}

// EndSession stamps ended_at on a session row. Safe to call with id == 0
// (no-op) so callers can write `defer logSvc.EndSession(ctx, id)` without
// branching on whether the start succeeded.
func (s *Service) EndSession(ctx context.Context, id int64) {
	if s == nil || s.repo == nil || id <= 0 {
		return
	}
	if err := s.repo.UpdateEndedAt(ctx, id, time.Now().Unix()); err != nil {
		log.Warn().Err(err).Int64("id", id).Msg("devicelog: end session failed")
	}
}

// Query lists logs matching the filter. Page/PageSize are normalized:
// page defaults to 1, pageSize is clamped to [1, 200].
func (s *Service) Query(ctx context.Context, q Query) ([]Log, int64, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	if q.PageSize > 200 {
		q.PageSize = 200
	}
	return s.repo.List(ctx, q)
}

func encodeDetail(m map[string]string) string {
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	if len(b) > maxDetailLen {
		return string(b[:maxDetailLen])
	}
	return string(b)
}
