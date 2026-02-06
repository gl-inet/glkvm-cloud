package app

import (
    "context"
    "database/sql"
    "rttys/internal/pkg/password"
)

type App struct {
    srv *Server
}

func (a *App) Start(ctx context.Context) error    { return a.srv.Run(ctx) }
func (a *App) Shutdown(ctx context.Context) error { return a.srv.Shutdown(ctx) }

func SeedIfEmpty(ctx context.Context, db *sql.DB) error {
    var n int
    if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM users`).Scan(&n); err != nil {
        return err
    }
    if n > 0 {
        return nil
    }

    // users
    adminHash, err := password.HashPassword("admin")
    if err != nil {
        return err
    }
    userHash, err := password.HashPassword("user")
    if err != nil {
        return err
    }

    if _, err := db.ExecContext(ctx, `
INSERT INTO users(username, description, password_hash, role, status, is_system) VALUES
('admin','Admin', ?, 'admin', 'active', 1),
('user1','User One', ?, 'user', 'active', 0)`,
        adminHash, userHash,
    ); err != nil {
        return err
    }

    // device groups
    if _, err := db.ExecContext(ctx, `
INSERT INTO device_groups(name, description) VALUES
('dg1','Device Group 1'),
('dg2','Device Group 2')`); err != nil {
        return err
    }

    // user groups
    if _, err := db.ExecContext(ctx, `
INSERT INTO user_groups(name, description) VALUES
('ug1','User Group 1'),
('ug2','User Group 2')`); err != nil {
        return err
    }

    // memberships: user1 in ug1
    if _, err := db.ExecContext(ctx, `
INSERT INTO user_group_members(user_id, group_id) VALUES (2, 1)`); err != nil {
        return err
    }

    // links: ug1 -> dg1
    if _, err := db.ExecContext(ctx, `
INSERT INTO user_group_device_group_links(user_group_id, device_group_id) VALUES (1, 1)`); err != nil {
        return err
    }

    // devices: one in dg1, one in dg2, one ungrouped
    if _, err := db.ExecContext(ctx, `
INSERT INTO devices(ddns, mac, name, description, device_group_id, status, last_seen_at) VALUES
('dev-001','00:11:22:33:44:55','Device 1','in dg1', 1, 'online', unixepoch()),
('dev-002','00:11:22:33:44:66','Device 2','in dg2', 2, 'offline', unixepoch()),
('dev-003','00:11:22:33:44:77','Device 3','ungrouped', NULL, 'online', unixepoch())`); err != nil {
        return err
    }

    return nil
}
