// Package notification provides email notification for device events.
// The service swallows errors so notifications never block the device runtime.
package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ─── SMTP config ────────────────────────────────────────────────

func (s *Service) GetSMTPConfig(ctx context.Context) (*SMTPConfig, error) {
	return s.repo.GetSMTPConfig(ctx)
}

func (s *Service) SaveSMTPConfig(ctx context.Context, cfg *SMTPConfig) error {
	cfg.UpdatedAt = time.Now().Unix()
	return s.repo.SaveSMTPConfig(ctx, cfg)
}

func (s *Service) TestSMTP(ctx context.Context, email string) error {
	cfg, err := s.repo.GetSMTPConfig(ctx)
	if err != nil {
		return fmt.Errorf("load smtp config: %w", err)
	}
	if cfg.Host == "" {
		return fmt.Errorf("SMTP is not configured")
	}
	subj, body := RenderTestEmail()
	return SendEmail(cfg, []string{email}, subj, body)
}

// ─── Notification rules ─────────────────────────────────────────

func (s *Service) GetNotifyRules(ctx context.Context) (*NotifyRules, error) {
	return s.repo.GetNotifyRules(ctx)
}

func (s *Service) SaveNotifyRules(ctx context.Context, rules *NotifyRules) error {
	rules.UpdatedAt = time.Now().Unix()
	return s.repo.SaveNotifyRules(ctx, rules)
}

// ─── Recipients ─────────────────────────────────────────────────

func (s *Service) ListRecipients(ctx context.Context) ([]Recipient, error) {
	return s.repo.ListRecipients(ctx)
}

func (s *Service) AddRecipient(ctx context.Context, email string) (*Recipient, error) {
	return s.repo.AddRecipient(ctx, email)
}

func (s *Service) RemoveRecipient(ctx context.Context, id int64) error {
	return s.repo.RemoveRecipient(ctx, id)
}

// ─── Event triggers (called from device runtime) ────────────────

// NotifyDeviceOnline sends a device-online notification if enabled.
func (s *Service) NotifyDeviceOnline(deviceID, mac string) {
	s.sendEventNotification("deviceOnline", "[GLKVM Cloud] Device Online", "Device Online", []EmailField{
		{Label: "Event", Value: "Device Online"},
		{Label: "Device ID", Value: deviceID},
		{Label: "MAC Address", Value: mac},
		{Label: "Time", Value: time.Now().UTC().Format("2006-01-02 15:04:05 UTC")},
	})
}

// NotifyDeviceOffline sends a device-offline notification if enabled.
func (s *Service) NotifyDeviceOffline(deviceID, mac string) {
	s.sendEventNotification("deviceOffline", "[GLKVM Cloud] Device Offline", "Device Offline", []EmailField{
		{Label: "Event", Value: "Device Offline"},
		{Label: "Device ID", Value: deviceID},
		{Label: "MAC Address", Value: mac},
		{Label: "Time", Value: time.Now().UTC().Format("2006-01-02 15:04:05 UTC")},
	})
}

// NotifyRemoteAccess sends a remote-access notification if enabled.
func (s *Service) NotifyRemoteAccess(accessType, deviceID, mac, actor, clientIP string) {
	s.sendEventNotification("remoteAccess", "[GLKVM Cloud] Remote Access: "+accessType, "Remote Access Detected", []EmailField{
		{Label: "Access Type", Value: accessType},
		{Label: "Device ID", Value: deviceID},
		{Label: "MAC Address", Value: mac},
		{Label: "Actor", Value: actor},
		{Label: "Client IP", Value: clientIP},
		{Label: "Time", Value: time.Now().UTC().Format("2006-01-02 15:04:05 UTC")},
	})
}

// sendEventNotification is the common helper: check rules → load recipients → send emails.
func (s *Service) sendEventNotification(ruleField, subject, title string, fields []EmailField) {
	if s == nil || s.repo == nil {
		return
	}
	go func() {
		ctx := context.Background()
		cfg, err := s.repo.GetSMTPConfig(ctx)
		if err != nil || cfg == nil || !cfg.Enabled || cfg.Host == "" {
			return
		}

		rules, err := s.repo.GetNotifyRules(ctx)
		if err != nil || rules == nil {
			return
		}
		if !s.ruleEnabled(rules, ruleField) {
			return
		}

		recipients, err := s.repo.ListRecipients(ctx)
		if err != nil || len(recipients) == 0 {
			return
		}

		to := make([]string, 0, len(recipients))
		for _, r := range recipients {
			to = append(to, r.Email)
		}

		body := RenderNotificationEmail(title, fields)
		if err := SendEmail(cfg, to, subject, body); err != nil {
			log.Warn().Err(err).Str("subject", subject).Msg("notification: send email failed")
		}
	}()
}

func (s *Service) ruleEnabled(rules *NotifyRules, field string) bool {
	switch field {
	case "deviceOnline":
		return rules.DeviceOnline
	case "deviceOffline":
		return rules.DeviceOffline
	case "remoteAccess":
		return rules.RemoteAccess
	default:
		return false
	}
}
