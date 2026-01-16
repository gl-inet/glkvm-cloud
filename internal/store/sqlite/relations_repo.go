package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type RelationsRepo struct{ db *sql.DB }

func NewRelationsRepo(db *sql.DB) *RelationsRepo { return &RelationsRepo{db: db} }

func (r *RelationsRepo) SetUserGroups(ctx context.Context, userID int64, groupIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM user_group_members WHERE user_id=?`, userID); err != nil {
		return err
	}

	if len(groupIDs) > 0 {
		vals := make([]string, 0, len(groupIDs))
		args := make([]any, 0, len(groupIDs)*2)
		for _, gid := range groupIDs {
			vals = append(vals, "(?,?)")
			args = append(args, userID, gid)
		}
		q := fmt.Sprintf(`INSERT INTO user_group_members(user_id, group_id) VALUES %s`, strings.Join(vals, ","))
		if _, err := tx.ExecContext(ctx, q, args...); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *RelationsRepo) SetUserGroupDeviceGroups(ctx context.Context, userGroupID int64, deviceGroupIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM user_group_device_group_links WHERE user_group_id=?`, userGroupID); err != nil {
		return err
	}

	if len(deviceGroupIDs) > 0 {
		vals := make([]string, 0, len(deviceGroupIDs))
		args := make([]any, 0, len(deviceGroupIDs)*2)
		for _, dg := range deviceGroupIDs {
			vals = append(vals, "(?,?)")
			args = append(args, userGroupID, dg)
		}
		q := fmt.Sprintf(`INSERT INTO user_group_device_group_links(user_group_id, device_group_id) VALUES %s`, strings.Join(vals, ","))
		if _, err := tx.ExecContext(ctx, q, args...); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *RelationsRepo) SetDeviceGroupDevices(ctx context.Context, deviceGroupID int64, deviceUIDs []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if len(deviceUIDs) == 0 {
		if _, err := tx.ExecContext(ctx, `UPDATE devices SET device_group_id=NULL WHERE device_group_id=?`, deviceGroupID); err != nil {
			return err
		}
		return tx.Commit()
	}

	ph := make([]string, 0, len(deviceUIDs))
	args := make([]any, 0, len(deviceUIDs)+1)
	args = append(args, deviceGroupID)
	for _, uid := range deviceUIDs {
		ph = append(ph, "?")
		args = append(args, uid)
	}
	qRemove := fmt.Sprintf(`UPDATE devices SET device_group_id=NULL WHERE device_group_id=? AND device_uid NOT IN (%s)`, strings.Join(ph, ","))
	if _, err := tx.ExecContext(ctx, qRemove, args...); err != nil {
		return err
	}

	ph2 := make([]string, 0, len(deviceUIDs))
	args2 := make([]any, 0, len(deviceUIDs)+1)
	args2 = append(args2, deviceGroupID)
	for _, uid := range deviceUIDs {
		ph2 = append(ph2, "?")
		args2 = append(args2, uid)
	}
	qAssign := fmt.Sprintf(`UPDATE devices SET device_group_id=? WHERE device_uid IN (%s)`, strings.Join(ph2, ","))
	res, err := tx.ExecContext(ctx, qAssign, args2...)
	if err != nil {
		return err
	}

	affected, _ := res.RowsAffected()
	if affected != int64(len(deviceUIDs)) {
		return fmt.Errorf("some deviceUids not found")
	}

	return tx.Commit()
}
