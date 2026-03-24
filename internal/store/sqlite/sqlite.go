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
    return ensureExternalIdentityIndex(ctx, db)
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
