package sqlite

import (
    "context"

    "gorm.io/gorm"

    "rttys/internal/domain/device"
)

type DeviceRepo struct{ db *gorm.DB }

func NewDeviceRepo(db *gorm.DB) *DeviceRepo { return &DeviceRepo{db: db} }

// 用于 DB 行映射
type deviceRow struct {
    ID            int64  `gorm:"column:id"`
    Ddns          string `gorm:"column:ddns"`
    Mac           string `gorm:"column:mac"`
    Name          string `gorm:"column:name"`
    Description   string `gorm:"column:description"`
    IP            string `gorm:"column:ip"`
    DeviceGroupID *int64 `gorm:"column:device_group_id"` // NULL => nil
    Status        string `gorm:"column:status"`
    LastSeenAt    *int64 `gorm:"column:last_seen_at"` // NULL => nil
}

func (deviceRow) TableName() string { return "devices" }

func (r *DeviceRepo) ListAll(ctx context.Context) ([]device.Device, error) {
    var rows []deviceRow
    err := r.db.WithContext(ctx).
        Order("id").
        Find(&rows).Error
    if err != nil {
        return nil, err
    }

    out := make([]device.Device, 0, len(rows))
    for _, row := range rows {
        out = append(out, device.Device{
            ID:            row.ID,
            Ddns:          row.Ddns,
            Mac:           row.Mac,
            Name:          row.Name,
            Description:   row.Description,
            IP:            row.IP,
            DeviceGroupID: row.DeviceGroupID,
            Status:        device.Status(row.Status),
            LastSeenAt:    row.LastSeenAt,
        })
    }
    return out, nil
}

func (r *DeviceRepo) ListByDeviceGroupIDs(ctx context.Context, groupIDs []int64) ([]device.Device, error) {
    if len(groupIDs) == 0 {
        return []device.Device{}, nil
    }

    var rows []deviceRow
    err := r.db.WithContext(ctx).
        Where("device_group_id IN ?", groupIDs).
        Order("id").
        Find(&rows).Error
    if err != nil {
        return nil, err
    }

    out := make([]device.Device, 0, len(rows))
    for _, row := range rows {
        out = append(out, device.Device{
            ID:            row.ID,
            Ddns:          row.Ddns,
            Mac:           row.Mac,
            Name:          row.Name,
            Description:   row.Description,
            IP:            row.IP,
            DeviceGroupID: row.DeviceGroupID,
            Status:        device.Status(row.Status),
            LastSeenAt:    row.LastSeenAt,
        })
    }
    return out, nil
}
