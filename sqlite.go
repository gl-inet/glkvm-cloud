package main

import (
    "errors"
    "fmt"
    "gorm.io/gorm"
    "gorm.io/gorm/clause"
    "rttys/db"
    "rttys/utils"
    "time"
)

// SaveOrUpdateDeviceMeta inserts or updates device metadata in the database.
// It performs an UPSERT operation based on device_id.
func SaveOrUpdateDeviceMeta(deviceID, mac, description, ip string) error {
    deviceDB := db.GetDbClient()
    if deviceDB == nil {
        return fmt.Errorf("deviceDB is not initialized")
    }

    now := time.Now().Unix()

    meta := &db.DeviceMeta{
        DeviceID:    deviceID,
        Mac:         utils.NormalizeMac(mac),
        IP:          ip,
        Description: description,
        CreateTime:  now,
        UpdateTime:  now,
    }

    // Use device_id as the conflict key and update fields on conflict
    return deviceDB.Clauses(clause.OnConflict{
        Columns: []clause.Column{
            {Name: "device_id"},
        },
        DoUpdates: clause.Assignments(map[string]any{
            "mac":         meta.Mac,
            "ip":          meta.IP,
            "description": meta.Description,
            "update_time": now,
        }),
    }).Create(meta).Error
}

// GetDeviceMetaByDeviceID retrieves device metadata by device_id.
// It returns (nil, nil) if the record does not exist.
func GetDeviceMetaByDeviceID(deviceID string) (*db.DeviceMeta, error) {
    deviceDB := db.GetDbClient()
    if deviceDB == nil {
        return nil, fmt.Errorf("deviceDB is not initialized")
    }

    var meta db.DeviceMeta
    if err := deviceDB.
        Where("device_id = ?", deviceID).
        First(&meta).Error; err != nil {

        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil
        }
        return nil, err
    }

    return &meta, nil
}

// GetDeviceMetaByMac retrieves device metadata by MAC address.
// The MAC address is normalized before querying.
// It returns (nil, nil) if the record does not exist.
func GetDeviceMetaByMac(mac string) (*db.DeviceMeta, error) {
    deviceDB := db.GetDbClient()
    if deviceDB == nil {
        return nil, fmt.Errorf("deviceDB is not initialized")
    }

    normMac := utils.NormalizeMac(mac)

    var meta db.DeviceMeta
    if err := deviceDB.
        Where("mac = ?", normMac).
        First(&meta).Error; err != nil {

        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil
        }
        return nil, err
    }

    return &meta, nil
}

// GetAllDeviceMeta retrieves device metadata records from the database.
// If keyword is empty, it returns all records ordered by create_time ASC.
// If keyword is non-empty, it searches by device_id, normalized MAC, or description (fuzzy match).
func GetAllDeviceMeta(keyword string) ([]db.DeviceMeta, error) {
    deviceDB := db.GetDbClient()
    if deviceDB == nil {
        return nil, fmt.Errorf("deviceDB is not initialized")
    }

    var list []db.DeviceMeta

    query := deviceDB.Model(&db.DeviceMeta{})

    if keyword != "" {
        // Normalize MAC in case the keyword is a MAC address
        normMac := utils.NormalizeMac(keyword)
        likeDesc := "%" + keyword + "%"

        query = query.Where(
            "device_id = ? OR mac = ? OR description LIKE ?",
            keyword,
            normMac,
            likeDesc,
        )
    }

    if err := query.
        Order("create_time ASC").
        Find(&list).Error; err != nil {
        return nil, err
    }

    return list, nil
}

// DeleteDeviceMetaByDeviceID deletes device metadata by device_id.
// It returns gorm.ErrRecordNotFound if no record is deleted.
func DeleteDeviceMetaByDeviceID(deviceID string) error {
    deviceDB := db.GetDbClient()
    if deviceDB == nil {
        return fmt.Errorf("deviceDB is not initialized")
    }

    result := deviceDB.
        Where("device_id = ?", deviceID).
        Delete(&db.DeviceMeta{})

    if result.Error != nil {
        return result.Error
    }

    if result.RowsAffected == 0 {
        return gorm.ErrRecordNotFound
    }

    return nil
}
