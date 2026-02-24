package sqlite

import (
    "context"
    "errors"
    "fmt"
    "rttys/model"

    "gorm.io/gorm"

    "rttys/utils"
)

type DeviceMetaRepo struct {
    db *gorm.DB
}

func NewDeviceMetaRepo(db *gorm.DB) *DeviceMetaRepo {
    return &DeviceMetaRepo{db: db}
}

func (r *DeviceMetaRepo) SaveOrUpdate(ctx context.Context, deviceID, mac, description, ip string) error {
    if r.db == nil {
        return fmt.Errorf("gorm db is nil")
    }

    normMac := utils.NormalizeMac(mac)
    return r.db.WithContext(ctx).Exec(
        `INSERT INTO devices (ddns, mac, description, ip, status, last_seen_at)
         VALUES (?, ?, ?, ?, 'online', unixepoch())
         ON CONFLICT(ddns) DO UPDATE SET
           mac=excluded.mac,
           description=excluded.description,
           ip=excluded.ip,
           status='online',
           last_seen_at=unixepoch()`,
        deviceID, normMac, description, ip,
    ).Error
}

func (r *DeviceMetaRepo) UpdateClient(ctx context.Context, deviceID, client string) error {
    if r.db == nil {
        return fmt.Errorf("gorm db is nil")
    }
    return r.db.WithContext(ctx).Exec(
        `UPDATE devices SET client=? WHERE ddns=?`,
        client, deviceID,
    ).Error
}

func (r *DeviceMetaRepo) UpdateDescriptionIfEmpty(ctx context.Context, deviceID, description string) error {
    if r.db == nil {
        return fmt.Errorf("gorm db is nil")
    }
    return r.db.WithContext(ctx).Exec(
        `UPDATE devices SET description=? WHERE ddns=? AND (description IS NULL OR description='')`,
        description, deviceID,
    ).Error
}

func (r *DeviceMetaRepo) GetByDeviceID(ctx context.Context, deviceID string) (*model.DeviceMeta, error) {
    var meta model.DeviceMeta
    err := r.db.WithContext(ctx).Where("ddns = ?", deviceID).First(&meta).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, nil
    }
    return &meta, err
}

func (r *DeviceMetaRepo) GetByMac(ctx context.Context, mac string) (*model.DeviceMeta, error) {
    normMac := utils.NormalizeMac(mac)

    var meta model.DeviceMeta
    err := r.db.WithContext(ctx).Where("mac = ?", normMac).First(&meta).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, nil
    }
    return &meta, err
}

func (r *DeviceMetaRepo) List(ctx context.Context, keyword string) ([]model.DeviceMeta, error) {
    var list []model.DeviceMeta

    q := r.db.WithContext(ctx).Model(&model.DeviceMeta{})
    if keyword != "" {
        normMac := utils.NormalizeMac(keyword)
        likeDesc := "%" + keyword + "%"
        q = q.Where("ddns = ? OR mac = ? OR description LIKE ? OR ip = ?", keyword, normMac, likeDesc, keyword)
    }

    if err := q.Order("id ASC").Find(&list).Error; err != nil {
        return nil, err
    }
    return list, nil
}

func (r *DeviceMetaRepo) ListByDeviceIDs(ctx context.Context, deviceIDs []string) ([]model.DeviceMeta, error) {
    if len(deviceIDs) == 0 {
        return []model.DeviceMeta{}, nil
    }

    var list []model.DeviceMeta
    if err := r.db.WithContext(ctx).
        Where("ddns IN ?", deviceIDs).
        Find(&list).Error; err != nil {
        return nil, err
    }
    return list, nil
}

func (r *DeviceMetaRepo) DeleteByDeviceID(ctx context.Context, deviceID string) error {
    res := r.db.WithContext(ctx).Where("ddns = ?", deviceID).Delete(&model.DeviceMeta{})
    if res.Error != nil {
        return res.Error
    }
    if res.RowsAffected == 0 {
        return gorm.ErrRecordNotFound
    }
    return nil
}

func (r *DeviceMetaRepo) MarkOffline(ctx context.Context, deviceID string) error {
    return r.db.WithContext(ctx).Exec(
        `UPDATE devices SET status='offline' WHERE ddns=?`,
        deviceID,
    ).Error
}
