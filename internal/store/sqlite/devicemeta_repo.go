package sqlite

import (
    "context"
    "errors"
    "fmt"
    "rttys/model"
    "time"

    "gorm.io/gorm"
    "gorm.io/gorm/clause"

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

    now := time.Now().Unix()
    meta := &model.DeviceMeta{
        DeviceID:    deviceID,
        Mac:         utils.NormalizeMac(mac),
        IP:          ip,
        Description: description,
        CreateTime:  now,
        UpdateTime:  now,
    }

    return r.db.WithContext(ctx).Clauses(clause.OnConflict{
        Columns: []clause.Column{{Name: "device_id"}},
        DoUpdates: clause.Assignments(map[string]any{
            "mac":         meta.Mac,
            "ip":          meta.IP,
            "description": meta.Description,
            "update_time": now,
        }),
    }).Create(meta).Error
}

func (r *DeviceMetaRepo) GetByDeviceID(ctx context.Context, deviceID string) (*model.DeviceMeta, error) {
    var meta model.DeviceMeta
    err := r.db.WithContext(ctx).Where("device_id = ?", deviceID).First(&meta).Error
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
        q = q.Where("device_id = ? OR mac = ? OR description LIKE ?", keyword, normMac, likeDesc)
    }

    if err := q.Order("create_time ASC").Find(&list).Error; err != nil {
        return nil, err
    }
    return list, nil
}

func (r *DeviceMetaRepo) DeleteByDeviceID(ctx context.Context, deviceID string) error {
    res := r.db.WithContext(ctx).Where("device_id = ?", deviceID).Delete(&model.DeviceMeta{})
    if res.Error != nil {
        return res.Error
    }
    if res.RowsAffected == 0 {
        return gorm.ErrRecordNotFound
    }
    return nil
}
