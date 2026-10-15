package sqlite

import (
    "context"
    "strings"

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
    Client        string `gorm:"column:client"`
    Model         string `gorm:"column:model"`
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
            Client:        row.Client,
            Model:         row.Model,
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
            Client:        row.Client,
            Model:         row.Model,
            DeviceGroupID: row.DeviceGroupID,
            Status:        device.Status(row.Status),
            LastSeenAt:    row.LastSeenAt,
        })
    }
    return out, nil
}

// deviceListRow carries the device columns plus the joined group name.
type deviceListRow struct {
    ID            int64  `gorm:"column:id"`
    Ddns          string `gorm:"column:ddns"`
    Mac           string `gorm:"column:mac"`
    Name          string `gorm:"column:name"`
    Description   string `gorm:"column:description"`
    IP            string `gorm:"column:ip"`
    Client        string `gorm:"column:client"`
    Model         string `gorm:"column:model"`
    DeviceGroupID *int64 `gorm:"column:device_group_id"`
    Status        string `gorm:"column:status"`
    LastSeenAt    *int64 `gorm:"column:last_seen_at"`
    GroupName     string `gorm:"column:group_name"`
}

// sortColumns whitelists the user-supplied sortBy to a safe SQL expression so
// the value is never interpolated as raw column input.
var sortColumns = map[string]string{
    "id":              "d.id",
    "ip":              "d.ip",
    "mac":             "d.mac",
    "ddns":            "d.ddns",
    "description":     "d.description",
    "connectedTime":   "COALESCE(d.last_seen_at, 0)",
    "deviceGroupName": "COALESCE(dg.name, '')",
}

func (r *DeviceRepo) ListPaged(ctx context.Context, q device.ListQuery) ([]device.ListItem, int64, error) {
    base := r.db.WithContext(ctx).
        Table("devices AS d").
        Joins("LEFT JOIN device_groups AS dg ON dg.id = d.device_group_id")

    if q.RestrictGroups != nil {
        base = base.Where("d.device_group_id IN ?", q.RestrictGroups)
    }
    if q.Unassigned {
        base = base.Where("d.device_group_id IS NULL")
    }
    if q.Status != "" {
        base = base.Where("d.status = ?", q.Status)
    }
    if q.Search != "" {
        like := "%" + q.Search + "%"
        base = base.Where(
            "lower(d.ddns) LIKE ? OR lower(d.mac) LIKE ? OR lower(d.ip) LIKE ? OR lower(d.description) LIKE ?",
            like, like, like, like)
    }

    var total int64
    if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
        return nil, 0, err
    }

    // ORDER BY: online devices first (always), then the requested field, then id
    // as a stable tie-breaker (mirrors the previous in-memory SliceStable order).
    expr, ok := sortColumns[q.SortBy]
    if !ok {
        expr = "d.ddns"
    }
    dir := "ASC"
    if strings.EqualFold(q.Order, "desc") {
        dir = "DESC"
    }
    orderBy := "(d.status = 'online') DESC, " + expr + " " + dir + ", d.id ASC"

    tx := base.Session(&gorm.Session{}).
        Select("d.*, COALESCE(dg.name, '') AS group_name").
        Order(orderBy)
    if q.PageSize > 0 {
        page := q.Page
        if page < 1 {
            page = 1
        }
        tx = tx.Limit(q.PageSize).Offset((page - 1) * q.PageSize)
    }

    var rows2 []deviceListRow
    if err := tx.Scan(&rows2).Error; err != nil {
        return nil, 0, err
    }

    out2 := make([]device.ListItem, 0, len(rows2))
    for _, row := range rows2 {
        out2 = append(out2, device.ListItem{
            Device: device.Device{
                ID:            row.ID,
                Ddns:          row.Ddns,
                Mac:           row.Mac,
                Name:          row.Name,
                Description:   row.Description,
                IP:            row.IP,
                Client:        row.Client,
                Model:         row.Model,
                DeviceGroupID: row.DeviceGroupID,
                Status:        device.Status(row.Status),
                LastSeenAt:    row.LastSeenAt,
            },
            GroupName: row.GroupName,
        })
    }
    return out2, total, nil
}
