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

func (userGroupRow) TableName() string { return "user_groups" }

type deviceGroupRow struct {
    ID          int64  `gorm:"column:id"`
    Name        string `gorm:"column:name"`
    Description string `gorm:"column:description"`
}

func (deviceGroupRow) TableName() string { return "device_groups" }

type UserGroupBrief struct {
    ID   int64  `gorm:"column:id"`
    Name string `gorm:"column:name"`
}

type DeviceGroupBrief struct {
    ID   int64  `gorm:"column:id"`
    Name string `gorm:"column:name"`
}

type idCountRow struct {
    ID  int64 `gorm:"column:id"`
    Cnt int64 `gorm:"column:cnt"`
}

type deviceGroupUserGroupRow struct {
    DeviceGroupID int64  `gorm:"column:device_group_id"`
    UserGroupID   int64  `gorm:"column:user_group_id"`
    UserGroupName string `gorm:"column:user_group_name"`
}

type userGroupDeviceGroupRow struct {
    UserGroupID     int64  `gorm:"column:user_group_id"`
    DeviceGroupID   int64  `gorm:"column:device_group_id"`
    DeviceGroupName string `gorm:"column:device_group_name"`
}

type userGroupMemberRow struct {
    UserID        int64  `gorm:"column:user_id"`
    UserGroupID   int64  `gorm:"column:user_group_id"`
    UserGroupName string `gorm:"column:user_group_name"`
}

type DeviceGroupDetail struct {
    ID          int64
    Name        string
    Description string
    DeviceCount int64
    UserGroups  []UserGroupBrief
}

type UserGroupDetail struct {
    ID           int64
    Name         string
    Description  string
    UserCount    int64
    DeviceGroups []DeviceGroupBrief
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
    row := userGroupRow{
        Name:        name,
        Description: description,
    }
    if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
        return 0, err
    }
    return row.ID, nil
}

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
    row := deviceGroupRow{
        Name:        name,
        Description: description,
    }
    if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
        return 0, err
    }
    return row.ID, nil
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

// ----------------------------------------------------
// Extended listing helpers (counts + relations)
// ----------------------------------------------------

func (r *GroupRepo) ListDeviceGroupDetails(ctx context.Context, userID int64, isAdmin bool) ([]DeviceGroupDetail, error) {
    base, err := r.ListDeviceGroupsVisibleToUser(ctx, userID, isAdmin)
    if err != nil {
        return nil, err
    }
    if len(base) == 0 {
        return []DeviceGroupDetail{}, nil
    }

    ids := make([]int64, 0, len(base))
    detailMap := make(map[int64]*DeviceGroupDetail, len(base))
    for _, g := range base {
        ids = append(ids, g.ID)
        detailMap[g.ID] = &DeviceGroupDetail{
            ID:          g.ID,
            Name:        g.Name,
            Description: g.Description,
            DeviceCount: 0,
            UserGroups:  []UserGroupBrief{},
        }
    }

    var countRows []idCountRow
    if err := r.db.WithContext(ctx).
        Raw(`SELECT device_group_id AS id, COUNT(1) AS cnt
             FROM devices WHERE device_group_id IN ? GROUP BY device_group_id`, ids).
        Scan(&countRows).Error; err != nil {
        return nil, err
    }
    for _, row := range countRows {
        if d, ok := detailMap[row.ID]; ok {
            d.DeviceCount = row.Cnt
        }
    }

    var linkRows []deviceGroupUserGroupRow
    if err := r.db.WithContext(ctx).
        Raw(`SELECT l.device_group_id AS device_group_id,
                    ug.id AS user_group_id,
                    ug.name AS user_group_name
             FROM user_group_device_group_links l
             JOIN user_groups ug ON ug.id=l.user_group_id
             WHERE l.device_group_id IN ?
             ORDER BY l.device_group_id, ug.id`, ids).
        Scan(&linkRows).Error; err != nil {
        return nil, err
    }
    for _, row := range linkRows {
        if d, ok := detailMap[row.DeviceGroupID]; ok {
            d.UserGroups = append(d.UserGroups, UserGroupBrief{ID: row.UserGroupID, Name: row.UserGroupName})
        }
    }

    out := make([]DeviceGroupDetail, 0, len(base))
    for _, g := range base {
        out = append(out, *detailMap[g.ID])
    }
    return out, nil
}

func (r *GroupRepo) ListUserGroupDetails(ctx context.Context, userID int64, isAdmin bool) ([]UserGroupDetail, error) {
    base, err := r.ListUserGroupsVisibleToUser(ctx, userID, isAdmin)
    if err != nil {
        return nil, err
    }
    if len(base) == 0 {
        return []UserGroupDetail{}, nil
    }

    ids := make([]int64, 0, len(base))
    detailMap := make(map[int64]*UserGroupDetail, len(base))
    for _, g := range base {
        ids = append(ids, g.ID)
        detailMap[g.ID] = &UserGroupDetail{
            ID:           g.ID,
            Name:         g.Name,
            Description:  g.Description,
            UserCount:    0,
            DeviceGroups: []DeviceGroupBrief{},
        }
    }

    var countRows []idCountRow
    if err := r.db.WithContext(ctx).
        Raw(`SELECT group_id AS id, COUNT(1) AS cnt
             FROM user_group_members WHERE group_id IN ? GROUP BY group_id`, ids).
        Scan(&countRows).Error; err != nil {
        return nil, err
    }
    for _, row := range countRows {
        if d, ok := detailMap[row.ID]; ok {
            d.UserCount = row.Cnt
        }
    }

    var linkRows []userGroupDeviceGroupRow
    if err := r.db.WithContext(ctx).
        Raw(`SELECT l.user_group_id AS user_group_id,
                    dg.id AS device_group_id,
                    dg.name AS device_group_name
             FROM user_group_device_group_links l
             JOIN device_groups dg ON dg.id=l.device_group_id
             WHERE l.user_group_id IN ?
             ORDER BY l.user_group_id, dg.id`, ids).
        Scan(&linkRows).Error; err != nil {
        return nil, err
    }
    for _, row := range linkRows {
        if d, ok := detailMap[row.UserGroupID]; ok {
            d.DeviceGroups = append(d.DeviceGroups, DeviceGroupBrief{ID: row.DeviceGroupID, Name: row.DeviceGroupName})
        }
    }

    out := make([]UserGroupDetail, 0, len(base))
    for _, g := range base {
        out = append(out, *detailMap[g.ID])
    }
    return out, nil
}

func (r *GroupRepo) ListUserGroupsByUserIDs(ctx context.Context, userIDs []int64) (map[int64][]UserGroupBrief, error) {
    out := make(map[int64][]UserGroupBrief)
    if len(userIDs) == 0 {
        return out, nil
    }

    var rows []userGroupMemberRow
    if err := r.db.WithContext(ctx).
        Raw(`SELECT ugm.user_id AS user_id,
                    ug.id AS user_group_id,
                    ug.name AS user_group_name
             FROM user_group_members ugm
             JOIN user_groups ug ON ug.id=ugm.group_id
             WHERE ugm.user_id IN ?
             ORDER BY ugm.user_id, ug.id`, userIDs).
        Scan(&rows).Error; err != nil {
        return nil, err
    }

    for _, row := range rows {
        out[row.UserID] = append(out[row.UserID], UserGroupBrief{ID: row.UserGroupID, Name: row.UserGroupName})
    }
    return out, nil
}
