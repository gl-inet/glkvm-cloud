package sqlite

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"rttys/internal/domain/trusteddevice"
)

type TrustedDeviceRepo struct{ db *gorm.DB }

func NewTrustedDeviceRepo(db *gorm.DB) *TrustedDeviceRepo { return &TrustedDeviceRepo{db: db} }

type trustedDeviceRow struct {
	ID         int64  `gorm:"column:id;primaryKey"`
	UserID     int64  `gorm:"column:user_id"`
	Token      string `gorm:"column:token"`
	DeviceName string `gorm:"column:device_name"`
	IP         string `gorm:"column:ip"`
	CreatedAt  int64  `gorm:"column:created_at"`
	LastUsedAt int64  `gorm:"column:last_used_at"`
	ExpiresAt  int64  `gorm:"column:expires_at"`
}

func (trustedDeviceRow) TableName() string { return "user_trusted_devices" }

func (r trustedDeviceRow) toDomain() *trusteddevice.Device {
	return &trusteddevice.Device{
		ID:         r.ID,
		UserID:     r.UserID,
		Token:      r.Token,
		DeviceName: r.DeviceName,
		IP:         r.IP,
		CreatedAt:  r.CreatedAt,
		LastUsedAt: r.LastUsedAt,
		ExpiresAt:  r.ExpiresAt,
	}
}

func (r *TrustedDeviceRepo) Create(ctx context.Context, d *trusteddevice.Device) (int64, error) {
	row := trustedDeviceRow{
		UserID:     d.UserID,
		Token:      d.Token,
		DeviceName: d.DeviceName,
		IP:         d.IP,
		CreatedAt:  d.CreatedAt,
		LastUsedAt: d.LastUsedAt,
		ExpiresAt:  d.ExpiresAt,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

func (r *TrustedDeviceRepo) FindByToken(ctx context.Context, token string) (*trusteddevice.Device, error) {
	var row trustedDeviceRow
	err := r.db.WithContext(ctx).
		Where("token = ?", token).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *TrustedDeviceRepo) ListByUserID(ctx context.Context, userID int64) ([]trusteddevice.Device, error) {
	var rows []trustedDeviceRow
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("last_used_at DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]trusteddevice.Device, 0, len(rows))
	for _, row := range rows {
		out = append(out, *row.toDomain())
	}
	return out, nil
}

func (r *TrustedDeviceRepo) Delete(ctx context.Context, id, userID int64) error {
	return r.db.WithContext(ctx).
		Exec("DELETE FROM user_trusted_devices WHERE id = ? AND user_id = ?", id, userID).Error
}

func (r *TrustedDeviceRepo) DeleteByUserID(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).
		Exec("DELETE FROM user_trusted_devices WHERE user_id = ?", userID).Error
}

func (r *TrustedDeviceRepo) TouchLastUsed(ctx context.Context, id int64, ts int64) error {
	return r.db.WithContext(ctx).
		Exec("UPDATE user_trusted_devices SET last_used_at = ? WHERE id = ?", ts, id).Error
}

func (r *TrustedDeviceRepo) DeleteExpired(ctx context.Context, before int64) error {
	return r.db.WithContext(ctx).
		Exec("DELETE FROM user_trusted_devices WHERE expires_at < ?", before).Error
}
