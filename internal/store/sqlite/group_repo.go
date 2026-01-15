package sqlite

import (
	"context"
	"database/sql"

	"rttys/internal/domain/devicegroup"
	"rttys/internal/domain/group"
)

type GroupRepo struct{ db *sql.DB }

func NewGroupRepo(db *sql.DB) *GroupRepo { return &GroupRepo{db: db} }

func (r *GroupRepo) ListUserGroupsVisibleToUser(ctx context.Context, userID int64, isAdmin bool) ([]group.UserGroup, error) {
	if isAdmin {
		rows, err := r.db.QueryContext(ctx, `SELECT id,name,description FROM user_groups ORDER BY id`)
		if err != nil { return nil, err }
		defer rows.Close()
		var out []group.UserGroup
		for rows.Next() {
			var g group.UserGroup
			if err := rows.Scan(&g.ID,&g.Name,&g.Description); err != nil { return nil, err }
			out = append(out, g)
		}
		return out, rows.Err()
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT ug.id,ug.name,ug.description
FROM user_groups ug
JOIN user_group_members ugm ON ugm.group_id=ug.id
WHERE ugm.user_id=?
ORDER BY ug.id`, userID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []group.UserGroup
	for rows.Next() {
		var g group.UserGroup
		if err := rows.Scan(&g.ID,&g.Name,&g.Description); err != nil { return nil, err }
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *GroupRepo) ListDeviceGroupIDsByUser(ctx context.Context, userID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT DISTINCT l.device_group_id
FROM user_group_members ugm
JOIN user_group_device_group_links l ON l.user_group_id=ugm.group_id
WHERE ugm.user_id=?
ORDER BY l.device_group_id`, userID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil { return nil, err }
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r *GroupRepo) ListDeviceGroupsVisibleToUser(ctx context.Context, userID int64, isAdmin bool) ([]devicegroup.DeviceGroup, error) {
	if isAdmin {
		rows, err := r.db.QueryContext(ctx, `SELECT id,name,description FROM device_groups ORDER BY id`)
		if err != nil { return nil, err }
		defer rows.Close()
		var out []devicegroup.DeviceGroup
		for rows.Next() {
			var dg devicegroup.DeviceGroup
			if err := rows.Scan(&dg.ID,&dg.Name,&dg.Description); err != nil { return nil, err }
			out = append(out, dg)
		}
		return out, rows.Err()
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT DISTINCT dg.id,dg.name,dg.description
FROM device_groups dg
JOIN user_group_device_group_links l ON l.device_group_id=dg.id
JOIN user_group_members ugm ON ugm.group_id=l.user_group_id
WHERE ugm.user_id=?
ORDER BY dg.id`, userID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []devicegroup.DeviceGroup
	for rows.Next() {
		var dg devicegroup.DeviceGroup
		if err := rows.Scan(&dg.ID,&dg.Name,&dg.Description); err != nil { return nil, err }
		out = append(out, dg)
	}
	return out, rows.Err()
}

func (r *GroupRepo) CreateUserGroup(ctx context.Context, name, description string) (int64, error) {
    res, err := r.db.ExecContext(ctx, `INSERT INTO user_groups(name, description) VALUES (?,?)`, name, description)
    if err != nil {
        return 0, err
    }
    return res.LastInsertId()
}

func (r *GroupRepo) UpdateUserGroup(ctx context.Context, id int64, name, description string) error {
    _, err := r.db.ExecContext(ctx, `UPDATE user_groups SET name=?, description=? WHERE id=?`, name, description, id)
    return err
}

func (r *GroupRepo) DeleteUserGroup(ctx context.Context, id int64) error {
    _, err := r.db.ExecContext(ctx, `DELETE FROM user_groups WHERE id=?`, id)
    return err
}

func (r *GroupRepo) CreateDeviceGroup(ctx context.Context, name, description string) (int64, error) {
    res, err := r.db.ExecContext(ctx, `INSERT INTO device_groups(name, description) VALUES (?,?)`, name, description)
    if err != nil {
        return 0, err
    }
    return res.LastInsertId()
}

func (r *GroupRepo) UpdateDeviceGroup(ctx context.Context, id int64, name, description string) error {
    _, err := r.db.ExecContext(ctx, `UPDATE device_groups SET name=?, description=? WHERE id=?`, name, description, id)
    return err
}

func (r *GroupRepo) DeleteDeviceGroup(ctx context.Context, id int64) error {
    _, err := r.db.ExecContext(ctx, `DELETE FROM device_groups WHERE id=?`, id)
    return err
}
