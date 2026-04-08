package sqlite

import (
    "context"
    "database/sql"
    "fmt"
    "os"
    "strings"

    gormsqlite "github.com/glebarez/sqlite"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
)

type AppDB struct {
    gorm *gorm.DB
    sql  *sql.DB
    dsn  string
}

func (a *AppDB) Gorm() *gorm.DB { return a.gorm }
func (a *AppDB) SQL() *sql.DB   { return a.sql }
func (a *AppDB) Close() error {
    if a.sql == nil {
        return nil
    }
    return a.sql.Close()
}

type Options struct {
    DSN          string // e.g. "/home/database/glkvm-cloud.db"
    MaxOpenConns int
    MaxIdleConns int
    LogSQL       bool
}

// Open opens sqlite via GORM (glebarez/sqlite) and exposes both *gorm.DB and *sql.DB.
func Open(ctx context.Context, opt Options) (*AppDB, error) {
    if opt.DSN == "" {
        return nil, fmt.Errorf("sqlite: DSN is empty")
    }
    if opt.MaxOpenConns == 0 {
        opt.MaxOpenConns = 1
    }
    if opt.MaxIdleConns == 0 {
        opt.MaxIdleConns = 1
    }

    gormCfg := &gorm.Config{}
    if opt.LogSQL {
        gormCfg.Logger = logger.Default.LogMode(logger.Info)
    }

    gdb, err := gorm.Open(gormsqlite.Open(opt.DSN), gormCfg)
    if err != nil {
        return nil, err
    }

    raw, err := gdb.DB()
    if err != nil {
        return nil, err
    }
    raw.SetMaxOpenConns(opt.MaxOpenConns)
    raw.SetMaxIdleConns(opt.MaxIdleConns)

    return &AppDB{
        gorm: gdb,
        sql:  raw,
        dsn:  opt.DSN,
    }, nil
}

func InitSchema(ctx context.Context, db *sql.DB, schemaPath string) error {
    b, err := os.ReadFile(schemaPath)
    if err != nil {
        return err
    }
    if _, err = db.ExecContext(ctx, string(b)); err != nil {
        return err
    }
    if err := ensureDeviceClientColumn(ctx, db); err != nil {
        return err
    }
    if err := ensureUserIsSystemColumn(ctx, db); err != nil {
        return err
    }
    if err := ensureAuthProviderColumn(ctx, db); err != nil {
        return err
    }
    if err := ensureExternalSubColumn(ctx, db); err != nil {
        return err
    }
    if err := ensureExternalIdentityIndex(ctx, db); err != nil {
        return err
    }
    if err := ensureUserLastLoginAtColumn(ctx, db); err != nil {
        return err
    }
    if err := ensureUserTotpSecretColumn(ctx, db); err != nil {
        return err
    }
    if err := ensureUserTotpEnabledColumn(ctx, db); err != nil {
        return err
    }
    if err := ensureTrustedDevicesTable(ctx, db); err != nil {
        return err
    }
    if err := ensureDeviceEventLogsTable(ctx, db); err != nil {
        return err
    }
    return ensureNotificationTables(ctx, db)
}

func ensureNotificationTables(ctx context.Context, db *sql.DB) error {
    if db == nil {
        return nil
    }
    if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS notification_smtp_config (
  id         INTEGER PRIMARY KEY CHECK (id = 1),
  host       TEXT    NOT NULL DEFAULT '',
  port       INTEGER NOT NULL DEFAULT 587,
  username   TEXT    NOT NULL DEFAULT '',
  password   TEXT    NOT NULL DEFAULT '',
  from_email TEXT    NOT NULL DEFAULT '',
  encryption TEXT    NOT NULL DEFAULT 'starttls',
  enabled    INTEGER NOT NULL DEFAULT 0,
  updated_at INTEGER NOT NULL DEFAULT 0
)`); err != nil {
        return err
    }
    if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS notification_rules (
  id             INTEGER PRIMARY KEY CHECK (id = 1),
  device_online  INTEGER NOT NULL DEFAULT 0,
  device_offline INTEGER NOT NULL DEFAULT 0,
  remote_access  INTEGER NOT NULL DEFAULT 0,
  updated_at     INTEGER NOT NULL DEFAULT 0
)`); err != nil {
        return err
    }
    if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS notification_recipients (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  email      TEXT    NOT NULL UNIQUE,
  created_at INTEGER NOT NULL DEFAULT (unixepoch())
)`); err != nil {
        return err
    }
    return nil
}

func ensureDeviceClientColumn(ctx context.Context, db *sql.DB) error {
    if db == nil {
        return nil
    }
    _, err := db.ExecContext(ctx, `ALTER TABLE devices ADD COLUMN client TEXT NOT NULL DEFAULT ''`)
    if err == nil {
        return nil
    }
    if strings.Contains(err.Error(), "duplicate column name") {
        return nil
    }
    return err
}

func ensureUserIsSystemColumn(ctx context.Context, db *sql.DB) error {
    if db == nil {
        return nil
    }
    _, err := db.ExecContext(ctx, `ALTER TABLE users ADD COLUMN is_system INTEGER NOT NULL DEFAULT 0`)
    if err == nil {
        return nil
    }
    if strings.Contains(err.Error(), "duplicate column name") {
        return nil
    }
    return err
}

func ensureAuthProviderColumn(ctx context.Context, db *sql.DB) error {
    if db == nil {
        return nil
    }
    _, err := db.ExecContext(ctx, `ALTER TABLE users ADD COLUMN auth_provider TEXT NOT NULL DEFAULT 'local'`)
    if err == nil {
        return nil
    }
    if strings.Contains(err.Error(), "duplicate column name") {
        return nil
    }
    return err
}

func ensureExternalSubColumn(ctx context.Context, db *sql.DB) error {
    if db == nil {
        return nil
    }
    _, err := db.ExecContext(ctx, `ALTER TABLE users ADD COLUMN external_sub TEXT NOT NULL DEFAULT ''`)
    if err == nil {
        return nil
    }
    if strings.Contains(err.Error(), "duplicate column name") {
        return nil
    }
    return err
}

func ensureExternalIdentityIndex(ctx context.Context, db *sql.DB) error {
    if db == nil {
        return nil
    }
    _, err := db.ExecContext(ctx,
        `CREATE UNIQUE INDEX IF NOT EXISTS idx_users_external_identity
         ON users(auth_provider, external_sub)
         WHERE external_sub != ''`)
    return err
}

func ensureUserLastLoginAtColumn(ctx context.Context, db *sql.DB) error {
    if db == nil {
        return nil
    }
    _, err := db.ExecContext(ctx, `ALTER TABLE users ADD COLUMN last_login_at INTEGER`)
    if err == nil {
        return nil
    }
    if strings.Contains(err.Error(), "duplicate column name") {
        return nil
    }
    return err
}

func ensureUserTotpSecretColumn(ctx context.Context, db *sql.DB) error {
    if db == nil {
        return nil
    }
    _, err := db.ExecContext(ctx, `ALTER TABLE users ADD COLUMN totp_secret TEXT NOT NULL DEFAULT ''`)
    if err == nil {
        return nil
    }
    if strings.Contains(err.Error(), "duplicate column name") {
        return nil
    }
    return err
}

func ensureUserTotpEnabledColumn(ctx context.Context, db *sql.DB) error {
    if db == nil {
        return nil
    }
    _, err := db.ExecContext(ctx, `ALTER TABLE users ADD COLUMN totp_enabled INTEGER NOT NULL DEFAULT 0`)
    if err == nil {
        return nil
    }
    if strings.Contains(err.Error(), "duplicate column name") {
        return nil
    }
    return err
}

func ensureTrustedDevicesTable(ctx context.Context, db *sql.DB) error {
    if db == nil {
        return nil
    }
    if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS user_trusted_devices (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id      INTEGER NOT NULL,
  token        TEXT    NOT NULL UNIQUE,
  device_name  TEXT    NOT NULL DEFAULT '',
  ip           TEXT    NOT NULL DEFAULT '',
  created_at   INTEGER NOT NULL DEFAULT (unixepoch()),
  last_used_at INTEGER NOT NULL DEFAULT (unixepoch()),
  expires_at   INTEGER NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
)`); err != nil {
        return err
    }
    if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_trusted_devices_user_id ON user_trusted_devices(user_id)`); err != nil {
        return err
    }
    if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_trusted_devices_token ON user_trusted_devices(token)`); err != nil {
        return err
    }
    return nil
}

func ensureDeviceEventLogsTable(ctx context.Context, db *sql.DB) error {
    if db == nil {
        return nil
    }
    if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS device_event_logs (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  device_id     TEXT    NOT NULL DEFAULT '',
  device_mac    TEXT    NOT NULL DEFAULT '',
  event_type    TEXT    NOT NULL,
  actor_user_id INTEGER NOT NULL DEFAULT 0,
  actor_name    TEXT    NOT NULL DEFAULT '',
  client_ip     TEXT    NOT NULL DEFAULT '',
  detail        TEXT    NOT NULL DEFAULT '',
  created_at    INTEGER NOT NULL DEFAULT (unixepoch()),
  ended_at      INTEGER NOT NULL DEFAULT 0
)`); err != nil {
        return err
    }
    if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_device_event_logs_mac ON device_event_logs(device_mac)`); err != nil {
        return err
    }
    if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_device_event_logs_type ON device_event_logs(event_type)`); err != nil {
        return err
    }
    if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_device_event_logs_created_at ON device_event_logs(created_at)`); err != nil {
        return err
    }
    return nil
}
