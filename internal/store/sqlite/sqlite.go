package sqlite

import (
	"context"
	"database/sql"
	"os"

	_ "modernc.org/sqlite"
)

type DB struct{ *sql.DB }

func Open(dsn string) (*DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return &DB{DB: db}, nil
}

func InitSchema(ctx context.Context, db *sql.DB, schemaPath string) error {
	b, err := os.ReadFile(schemaPath)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, string(b))
	return err
}
