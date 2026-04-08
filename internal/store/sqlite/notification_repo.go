package sqlite

import (
	"context"
	"time"

	"gorm.io/gorm"

	"rttys/internal/domain/notification"
)

// ─── Repo ───────────────────────────────────────────────────────

type NotificationRepo struct{ db *gorm.DB }

func NewNotificationRepo(db *gorm.DB) *NotificationRepo {
	return &NotificationRepo{db: db}
}

// ─── SMTP Config ────────────────────────────────────────────────

type smtpConfigRow struct {
	ID         int64  `gorm:"column:id;primaryKey"`
	Host       string `gorm:"column:host"`
	Port       int    `gorm:"column:port"`
	Username   string `gorm:"column:username"`
	Password   string `gorm:"column:password"`
	FromEmail  string `gorm:"column:from_email"`
	Encryption string `gorm:"column:encryption"`
	Enabled    int    `gorm:"column:enabled"`
	UpdatedAt  int64  `gorm:"column:updated_at"`
}

func (smtpConfigRow) TableName() string { return "notification_smtp_config" }

func (r *NotificationRepo) GetSMTPConfig(ctx context.Context) (*notification.SMTPConfig, error) {
	var row smtpConfigRow
	err := r.db.WithContext(ctx).First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &notification.SMTPConfig{Port: 587, Encryption: "starttls"}, nil
		}
		return nil, err
	}
	return &notification.SMTPConfig{
		Host:       row.Host,
		Port:       row.Port,
		Username:   row.Username,
		Password:   row.Password,
		FromEmail:  row.FromEmail,
		Encryption: row.Encryption,
		Enabled:    row.Enabled == 1,
		UpdatedAt:  row.UpdatedAt,
	}, nil
}

func (r *NotificationRepo) SaveSMTPConfig(ctx context.Context, cfg *notification.SMTPConfig) error {
	enabled := 0
	if cfg.Enabled {
		enabled = 1
	}
	row := smtpConfigRow{
		ID:         1,
		Host:       cfg.Host,
		Port:       cfg.Port,
		Username:   cfg.Username,
		Password:   cfg.Password,
		FromEmail:  cfg.FromEmail,
		Encryption: cfg.Encryption,
		Enabled:    enabled,
		UpdatedAt:  cfg.UpdatedAt,
	}
	return r.db.WithContext(ctx).Save(&row).Error
}

// ─── Notify Rules ───────────────────────────────────────────────

type notifyRulesRow struct {
	ID            int64 `gorm:"column:id;primaryKey"`
	DeviceOnline  int   `gorm:"column:device_online"`
	DeviceOffline int   `gorm:"column:device_offline"`
	RemoteAccess  int   `gorm:"column:remote_access"`
	UpdatedAt     int64 `gorm:"column:updated_at"`
}

func (notifyRulesRow) TableName() string { return "notification_rules" }

func (r *NotificationRepo) GetNotifyRules(ctx context.Context) (*notification.NotifyRules, error) {
	var row notifyRulesRow
	err := r.db.WithContext(ctx).First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &notification.NotifyRules{}, nil
		}
		return nil, err
	}
	return &notification.NotifyRules{
		DeviceOnline:  row.DeviceOnline == 1,
		DeviceOffline: row.DeviceOffline == 1,
		RemoteAccess:  row.RemoteAccess == 1,
		UpdatedAt:     row.UpdatedAt,
	}, nil
}

func (r *NotificationRepo) SaveNotifyRules(ctx context.Context, rules *notification.NotifyRules) error {
	boolToInt := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}
	row := notifyRulesRow{
		ID:            1,
		DeviceOnline:  boolToInt(rules.DeviceOnline),
		DeviceOffline: boolToInt(rules.DeviceOffline),
		RemoteAccess:  boolToInt(rules.RemoteAccess),
		UpdatedAt:     rules.UpdatedAt,
	}
	return r.db.WithContext(ctx).Save(&row).Error
}

// ─── Recipients ─────────────────────────────────────────────────

type recipientRow struct {
	ID        int64  `gorm:"column:id;primaryKey"`
	Email     string `gorm:"column:email"`
	CreatedAt int64  `gorm:"column:created_at"`
}

func (recipientRow) TableName() string { return "notification_recipients" }

func (r *NotificationRepo) ListRecipients(ctx context.Context) ([]notification.Recipient, error) {
	var rows []recipientRow
	if err := r.db.WithContext(ctx).Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]notification.Recipient, 0, len(rows))
	for _, row := range rows {
		out = append(out, notification.Recipient{
			ID:        row.ID,
			Email:     row.Email,
			CreatedAt: row.CreatedAt,
		})
	}
	return out, nil
}

func (r *NotificationRepo) AddRecipient(ctx context.Context, email string) (*notification.Recipient, error) {
	row := recipientRow{
		Email:     email,
		CreatedAt: time.Now().Unix(),
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return &notification.Recipient{
		ID:        row.ID,
		Email:     row.Email,
		CreatedAt: row.CreatedAt,
	}, nil
}

func (r *NotificationRepo) RemoveRecipient(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&recipientRow{}, id).Error
}
