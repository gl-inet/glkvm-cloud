package app

import (
    "context"
    "database/sql"
    "log"

    "rttys/internal/config"
    "rttys/internal/domain/device"
    "rttys/internal/domain/permission"
    "rttys/internal/domain/user"
    httpx "rttys/internal/http"
    "rttys/internal/pkg/password"
    "rttys/internal/store/memory"
    "rttys/internal/store/sqlite"
)

type App struct {
    srv *Server
}

func Bootstrap(cfg config.Config) *App {
    ctx := context.Background()

    // --- DB ---
    db, err := sqlite.Open(cfg.DB.DSN)
    if err != nil {
        log.Fatalf("open sqlite: %v", err)
    }
    if err := sqlite.InitSchema(ctx, db.DB, "internal/store/sqlite/schema.sql"); err != nil {
        log.Fatalf("init schema: %v", err)
    }

    // Seed demo data if empty (optional)
    if err := seedIfEmpty(ctx, db.DB); err != nil {
        log.Fatalf("seed: %v", err)
    }

    // --- Repos & Services ---
    userRepo := sqlite.NewUserRepo(db.DB)
    groupRepo := sqlite.NewGroupRepo(db.DB)
    deviceRepo := sqlite.NewDeviceRepo(db.DB)
    relationsRepo := sqlite.NewRelationsRepo(db.DB)

    userSvc := user.NewService(userRepo)
    devSvc := device.NewService(deviceRepo, groupRepo)

    permRepo := memory.NewPermissionRepo() // permissions stay in-memory
    permSvc := permission.NewService(permRepo)

    sessionStore := memory.NewSessionStore(cfg.Auth.SessionTTL)

    router := httpx.NewRouter(httpx.Deps{
        UserSvc:       userSvc,
        PermSvc:       permSvc,
        DevSvc:        devSvc,
        GroupRepo:     groupRepo,
        SessionStore:  sessionStore,
        RelationsRepo: relationsRepo,
    })

    srv := NewServer(cfg.HTTP.Addr, router)
    return &App{srv: srv}
}

func (a *App) Start(ctx context.Context) error    { return a.srv.Run(ctx) }
func (a *App) Shutdown(ctx context.Context) error { return a.srv.Shutdown(ctx) }

func seedIfEmpty(ctx context.Context, db *sql.DB) error {
    var n int
    if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM users`).Scan(&n); err != nil {
        return err
    }
    if n > 0 {
        return nil
    }

    // users
    adminHash := password.HashDemoSHA256("admin")
    userHash := password.HashDemoSHA256("user")

    if _, err := db.ExecContext(ctx, `
INSERT INTO users(email, display_name, password_hash, role, status) VALUES
('admin@example.com','Admin', ?, 'admin', 'active'),
('user1@example.com','User One', ?, 'user', 'active')`,
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
INSERT INTO devices(device_uid, name, description, device_group_id, status) VALUES
('dev-001','Device 1','in dg1', 1, 'online'),
('dev-002','Device 2','in dg2', 2, 'offline'),
('dev-003','Device 3','ungrouped', NULL, 'online')`); err != nil {
        return err
    }

    return nil
}
