package sqlite

import (
    "context"

    "gorm.io/gorm"

    "rttys/internal/domain/devicegroup"
    "rttys/internal/domain/group"
)

type GroupRepo struct{ db *gorm.DB }

func NewGroupRepo(db *gorm.DB) *GroupRepo { return &GroupRepo{db: db} }

// --- row mapping (only for scan) ---

type userGroupRow struct {
    ID          int64  `gorm:"column:id"`
    Name        string `gorm:"column:name"`
    Description string `gorm:"column:description"`
}

type deviceGroupRow struct {
    ID          int64  `gorm:"column:id"`
    Name        string `gorm:"column:name"`
    Description string `gorm:"column:description"`
}

// -----------------------------
// Visible query
// -----------------------------

func (r *GroupRepo) ListUserGroupsVisibleToUser(ctx context.Context, userID int64, isAdmin bool) ([]group.UserGroup, error) {
    if isAdmin {
        var rows []userGroupRow
        err := r.db.WithContext(ctx).
            Raw(`SELECT id,name,description FROM user_groups ORDER BY id`).
            Scan(&rows).Error
        if err != nil {
            return nil, err
        }
        return mapUserGroups(rows), nil
    }

    var rows []userGroupRow
    err := r.db.WithContext(ctx).
        Raw(`
SELECT ug.id,ug.name,ug.description
FROM user_groups ug
JOIN user_group_members ugm ON ugm.group_id=ug.id
WHERE ugm.user_id=?
ORDER BY ug.id`, userID).
        Scan(&rows).Error
    if err != nil {
        return nil, err
    }

    return mapUserGroups(rows), nil
}

func (r *GroupRepo) ListDeviceGroupIDsByUser(ctx context.Context, userID int64) ([]int64, error) {
    var ids []int64
    err := r.db.WithContext(ctx).
        Raw(`
SELECT DISTINCT l.device_group_id
FROM user_group_members ugm
JOIN user_group_device_group_links l ON l.user_group_id=ugm.group_id
WHERE ugm.user_id=?
ORDER BY l.device_group_id`, userID).
        Scan(&ids).Error
    if err != nil {
        return nil, err
    }
    return ids, nil
}

func (r *GroupRepo) ListDeviceGroupsVisibleToUser(ctx context.Context, userID int64, isAdmin bool) ([]devicegroup.DeviceGroup, error) {
    if isAdmin {
        var rows []deviceGroupRow
        err := r.db.WithContext(ctx).
            Raw(`SELECT id,name,description FROM device_groups ORDER BY id`).
            Scan(&rows).Error
        if err != nil {
            return nil, err
        }
        return mapDeviceGroups(rows), nil
    }

    var rows []deviceGroupRow
    err := r.db.WithContext(ctx).
        Raw(`
SELECT DISTINCT dg.id,dg.name,dg.description
FROM device_groups dg
JOIN user_group_device_group_links l ON l.device_group_id=dg.id
JOIN user_group_members ugm ON ugm.group_id=l.user_group_id
WHERE ugm.user_id=?
ORDER BY dg.id`, userID).
        Scan(&rows).Error
    if err != nil {
        return nil, err
    }

    return mapDeviceGroups(rows), nil
}

// -----------------------------
// CRUD - use Exec (keeps behavior close to original)
// -----------------------------

func (r *GroupRepo) CreateUserGroup(ctx context.Context, name, description string) (int64, error) {
    res := r.db.WithContext(ctx).Exec(
        `INSERT INTO user_groups(name, description) VALUES (?,?)`,
        name, description,
    )
    if res.Error != nil {
        return 0, res.Error
    }
    return res.RowsAffected, nil // 注意：RowsAffected 不是 LastInsertId
}

// 如果你需要“返回新ID”，推荐用下面这个版本（见下方“返回 LastInsertId”方案）

func (r *GroupRepo) UpdateUserGroup(ctx context.Context, id int64, name, description string) error {
    return r.db.WithContext(ctx).Exec(
        `UPDATE user_groups SET name=?, description=? WHERE id=?`,
        name, description, id,
    ).Error
}

func (r *GroupRepo) DeleteUserGroup(ctx context.Context, id int64) error {
    return r.db.WithContext(ctx).Exec(
        `DELETE FROM user_groups WHERE id=?`,
        id,
    ).Error
}

func (r *GroupRepo) CreateDeviceGroup(ctx context.Context, name, description string) (int64, error) {
    res := r.db.WithContext(ctx).Exec(
        `INSERT INTO device_groups(name, description) VALUES (?,?)`,
        name, description,
    )
    if res.Error != nil {
        return 0, res.Error
    }
    return res.RowsAffected, nil
}

func (r *GroupRepo) UpdateDeviceGroup(ctx context.Context, id int64, name, description string) error {
    return r.db.WithContext(ctx).Exec(
        `UPDATE device_groups SET name=?, description=? WHERE id=?`,
        name, description, id,
    ).Error
}

func (r *GroupRepo) DeleteDeviceGroup(ctx context.Context, id int64) error {
    return r.db.WithContext(ctx).Exec(
        `DELETE FROM device_groups WHERE id=?`,
        id,
    ).Error
}

// -----------------------------
// mapping helpers
// -----------------------------

func mapUserGroups(rows []userGroupRow) []group.UserGroup {
    out := make([]group.UserGroup, 0, len(rows))
    for _, r := range rows {
        out = append(out, group.UserGroup{
            ID:          r.ID,
            Name:        r.Name,
            Description: r.Description,
        })
    }
    return out
}

func mapDeviceGroups(rows []deviceGroupRow) []devicegroup.DeviceGroup {
    out := make([]devicegroup.DeviceGroup, 0, len(rows))
    for _, r := range rows {
        out = append(out, devicegroup.DeviceGroup{
            ID:          r.ID,
            Name:        r.Name,
            Description: r.Description,
        })
    }
    return out
}
