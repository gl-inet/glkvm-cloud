package sqlite

import (
    "context"
    "fmt"
    "strings"

    "gorm.io/gorm"
)

type RelationsRepo struct{ db *gorm.DB }

func NewRelationsRepo(db *gorm.DB) *RelationsRepo {
    return &RelationsRepo{db: db}
}

// ----------------------------------------------------
// user <-> user_groups  (cover / set)
// ----------------------------------------------------

func (r *RelationsRepo) SetUserGroups(ctx context.Context, userID int64, groupIDs []int64) error {
    return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        // 1) delete old relations
        if err := tx.Exec(
            `DELETE FROM user_group_members WHERE user_id=?`,
            userID,
        ).Error; err != nil {
            return err
        }

        // 2) insert new relations
        if len(groupIDs) == 0 {
            return nil
        }

        vals := make([]string, 0, len(groupIDs))
        args := make([]any, 0, len(groupIDs)*2)
        for _, gid := range groupIDs {
            vals = append(vals, "(?,?)")
            args = append(args, userID, gid)
        }

        q := fmt.Sprintf(
            `INSERT INTO user_group_members(user_id, group_id) VALUES %s`,
            strings.Join(vals, ","),
        )

        return tx.Exec(q, args...).Error
    })
}

// ----------------------------------------------------
// user_group <-> device_groups  (cover / set)
// ----------------------------------------------------

func (r *RelationsRepo) SetUserGroupDeviceGroups(ctx context.Context, userGroupID int64, deviceGroupIDs []int64) error {
    return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        if err := tx.Exec(
            `DELETE FROM user_group_device_group_links WHERE user_group_id=?`,
            userGroupID,
        ).Error; err != nil {
            return err
        }

        if len(deviceGroupIDs) == 0 {
            return nil
        }

        vals := make([]string, 0, len(deviceGroupIDs))
        args := make([]any, 0, len(deviceGroupIDs)*2)
        for _, dg := range deviceGroupIDs {
            vals = append(vals, "(?,?)")
            args = append(args, userGroupID, dg)
        }

        q := fmt.Sprintf(
            `INSERT INTO user_group_device_group_links(user_group_id, device_group_id) VALUES %s`,
            strings.Join(vals, ","),
        )

        return tx.Exec(q, args...).Error
    })
}

// ----------------------------------------------------
// device_group <-> devices  (cover / set, one device -> one group)
// ----------------------------------------------------

func (r *RelationsRepo) SetDeviceGroupDevices(ctx context.Context, deviceGroupID int64, deviceIDs []int64) error {
    return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

        // case 1: empty list => clear all devices in this group
        if len(deviceIDs) == 0 {
            return tx.Exec(
                `UPDATE devices SET device_group_id=NULL WHERE device_group_id=?`,
                deviceGroupID,
            ).Error
        }

        // 1) remove devices that are no longer in the group
        ph := make([]string, 0, len(deviceIDs))
        args := make([]any, 0, len(deviceIDs)+1)
        args = append(args, deviceGroupID)
        for _, id := range deviceIDs {
            ph = append(ph, "?")
            args = append(args, id)
        }

        qRemove := fmt.Sprintf(
            `UPDATE devices SET device_group_id=NULL
			 WHERE device_group_id=?
			   AND id NOT IN (%s)`,
            strings.Join(ph, ","),
        )

        if err := tx.Exec(qRemove, args...).Error; err != nil {
            return err
        }

        // 2) assign devices to this group (overwrite old group)
        ph2 := make([]string, 0, len(deviceIDs))
        args2 := make([]any, 0, len(deviceIDs)+1)
        args2 = append(args2, deviceGroupID)
        for _, id := range deviceIDs {
            ph2 = append(ph2, "?")
            args2 = append(args2, id)
        }

        qAssign := fmt.Sprintf(
            `UPDATE devices SET device_group_id=?
			 WHERE id IN (%s)`,
            strings.Join(ph2, ","),
        )

        res := tx.Exec(qAssign, args2...)
        if res.Error != nil {
            return res.Error
        }

        return nil
    })
}

// ----------------------------------------------------
// device_group <-> user_groups (cover / set)
// ----------------------------------------------------

func (r *RelationsRepo) SetDeviceGroupUserGroups(ctx context.Context, deviceGroupID int64, userGroupIDs []int64) error {
    return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        if err := tx.Exec(
            `DELETE FROM user_group_device_group_links WHERE device_group_id=?`,
            deviceGroupID,
        ).Error; err != nil {
            return err
        }

        if len(userGroupIDs) == 0 {
            return nil
        }

        vals := make([]string, 0, len(userGroupIDs))
        args := make([]any, 0, len(userGroupIDs)*2)
        for _, ugID := range userGroupIDs {
            vals = append(vals, "(?,?)")
            args = append(args, ugID, deviceGroupID)
        }

        q := fmt.Sprintf(
            `INSERT INTO user_group_device_group_links(user_group_id, device_group_id) VALUES %s`,
            strings.Join(vals, ","),
        )

        return tx.Exec(q, args...).Error
    })
}

// ----------------------------------------------------
// device_group <-> devices (add/remove)
// ----------------------------------------------------

func (r *RelationsRepo) AddDevicesToGroup(ctx context.Context, deviceGroupID int64, deviceIDs []int64) error {
    if len(deviceIDs) == 0 {
        return nil
    }

    ph := make([]string, 0, len(deviceIDs))
    args := make([]any, 0, len(deviceIDs)+1)
    args = append(args, deviceGroupID)
    for _, id := range deviceIDs {
        ph = append(ph, "?")
        args = append(args, id)
    }

    q := fmt.Sprintf(
        `UPDATE devices SET device_group_id=? WHERE id IN (%s)`,
        strings.Join(ph, ","),
    )

    return r.db.WithContext(ctx).Exec(q, args...).Error
}

func (r *RelationsRepo) RemoveDevicesFromGroup(ctx context.Context, deviceGroupID int64, deviceIDs []int64) error {
    if len(deviceIDs) == 0 {
        return nil
    }

    ph := make([]string, 0, len(deviceIDs))
    args := make([]any, 0, len(deviceIDs)+1)
    args = append(args, deviceGroupID)
    for _, id := range deviceIDs {
        ph = append(ph, "?")
        args = append(args, id)
    }

    q := fmt.Sprintf(
        `UPDATE devices SET device_group_id=NULL WHERE device_group_id=? AND id IN (%s)`,
        strings.Join(ph, ","),
    )

    return r.db.WithContext(ctx).Exec(q, args...).Error
}
