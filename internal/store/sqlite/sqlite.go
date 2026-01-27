package sqlite

import (
    "context"
    "database/sql"
    "fmt"
    "os"

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
    _, err = db.ExecContext(ctx, string(b))
    return err
}
