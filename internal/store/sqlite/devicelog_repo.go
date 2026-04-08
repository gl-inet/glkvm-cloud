package sqlite

import (
	"context"

	"gorm.io/gorm"

	"rttys/internal/domain/devicelog"
)

type DeviceLogRepo struct{ db *gorm.DB }

func NewDeviceLogRepo(db *gorm.DB) *DeviceLogRepo { return &DeviceLogRepo{db: db} }

type deviceLogRow struct {
	ID          int64  `gorm:"column:id;primaryKey"`
	DeviceID    string `gorm:"column:device_id"`
	DeviceMac   string `gorm:"column:device_mac"`
	EventType   string `gorm:"column:event_type"`
	ActorUserID int64  `gorm:"column:actor_user_id"`
	ActorName   string `gorm:"column:actor_name"`
	ClientIP    string `gorm:"column:client_ip"`
	Detail      string `gorm:"column:detail"`
	CreatedAt   int64  `gorm:"column:created_at"`
	EndedAt     int64  `gorm:"column:ended_at"`
}

func (deviceLogRow) TableName() string { return "device_event_logs" }

func (r deviceLogRow) toDomain() devicelog.Log {
	return devicelog.Log{
		ID:          r.ID,
		DeviceID:    r.DeviceID,
		DeviceMac:   r.DeviceMac,
		EventType:   devicelog.EventType(r.EventType),
		ActorUserID: r.ActorUserID,
		ActorName:   r.ActorName,
		ClientIP:    r.ClientIP,
		Detail:      r.Detail,
		CreatedAt:   r.CreatedAt,
		EndedAt:     r.EndedAt,
	}
}

func (r *DeviceLogRepo) Create(ctx context.Context, l *devicelog.Log) (int64, error) {
	row := deviceLogRow{
		DeviceID:    l.DeviceID,
		DeviceMac:   l.DeviceMac,
		EventType:   string(l.EventType),
		ActorUserID: l.ActorUserID,
		ActorName:   l.ActorName,
		ClientIP:    l.ClientIP,
		Detail:      l.Detail,
		CreatedAt:   l.CreatedAt,
		EndedAt:     l.EndedAt,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return 0, err
	}
	return row.ID, nil
}

func (r *DeviceLogRepo) UpdateEndedAt(ctx context.Context, id int64, ts int64) error {
	return r.db.WithContext(ctx).
		Exec(`UPDATE device_event_logs SET ended_at = ? WHERE id = ?`, ts, id).Error
}

func (r *DeviceLogRepo) List(ctx context.Context, q devicelog.Query) ([]devicelog.Log, int64, error) {
	tx := r.db.WithContext(ctx).Model(&deviceLogRow{})

	if q.Mac != "" {
		tx = tx.Where("device_mac LIKE ?", "%"+q.Mac+"%")
	}
	if len(q.EventTypes) > 0 {
		types := make([]string, 0, len(q.EventTypes))
		for _, t := range q.EventTypes {
			types = append(types, string(t))
		}
		tx = tx.Where("event_type IN ?", types)
	}
	if q.From > 0 {
		tx = tx.Where("created_at >= ?", q.From)
	}
	if q.To > 0 {
		tx = tx.Where("created_at <= ?", q.To)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []deviceLogRow
	if err := tx.
		Order("created_at DESC").
		Offset((q.Page - 1) * q.PageSize).
		Limit(q.PageSize).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	out := make([]devicelog.Log, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.toDomain())
	}
	return out, total, nil
}
