package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"rttys/internal/domain/device"
)

type DeviceRepo struct{ db *sql.DB }

func NewDeviceRepo(db *sql.DB) *DeviceRepo { return &DeviceRepo{db: db} }

func (r *DeviceRepo) ListAll(ctx context.Context) ([]device.Device, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id,device_uid,name,description,device_group_id,status,last_seen_at
FROM devices ORDER BY id`)
	if err != nil { return nil, err }
	defer rows.Close()

	var out []device.Device
	for rows.Next() {
		var d device.Device
		var dg sql.NullInt64
		var st string
		var last sql.NullInt64
		if err := rows.Scan(&d.ID,&d.DeviceUID,&d.Name,&d.Description,&dg,&st,&last); err != nil { return nil, err }
		if dg.Valid { v := dg.Int64; d.DeviceGroupID = &v }
		d.Status = device.Status(st)
		if last.Valid { v := last.Int64; d.LastSeenAt = &v }
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *DeviceRepo) ListByDeviceGroupIDs(ctx context.Context, groupIDs []int64) ([]device.Device, error) {
	if len(groupIDs)==0 { return []device.Device{}, nil }
	ph := make([]string,0,len(groupIDs))
	args := make([]any,0,len(groupIDs))
	for _, id := range groupIDs { ph=append(ph,"?"); args=append(args,id) }

	q := fmt.Sprintf(`
SELECT id,device_uid,name,description,device_group_id,status,last_seen_at
FROM devices
WHERE device_group_id IN (%s)
ORDER BY id`, strings.Join(ph, ","))

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil { return nil, err }
	defer rows.Close()

	var out []device.Device
	for rows.Next() {
		var d device.Device
		var dg sql.NullInt64
		var st string
		var last sql.NullInt64
		if err := rows.Scan(&d.ID,&d.DeviceUID,&d.Name,&d.Description,&dg,&st,&last); err != nil { return nil, err }
		if dg.Valid { v := dg.Int64; d.DeviceGroupID = &v }
		d.Status = device.Status(st)
		if last.Valid { v := last.Int64; d.LastSeenAt = &v }
		out = append(out, d)
	}
	return out, rows.Err()
}
