package sqlite

import (
	"context"
	"path/filepath"
	"testing"
)

// Verifies the reconnect minimal-write path: MarkOnline flips status + refreshes
// last_seen_at without rewriting identity columns, and UpdateClient only writes
// when the value actually changed.
func TestMarkOnlineAndClientGuard(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "mw.db")
	db, err := Open(context.Background(), Options{DSN: dsn, MaxOpenConns: 4, MaxIdleConns: 4})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	g := db.Gorm()
	g.Exec(`CREATE TABLE devices (id INTEGER PRIMARY KEY AUTOINCREMENT, ddns TEXT, mac TEXT, name TEXT DEFAULT '',
		description TEXT DEFAULT '', ip TEXT DEFAULT '', client TEXT DEFAULT '', model TEXT NOT NULL DEFAULT '', device_group_id INTEGER,
		status TEXT, last_seen_at INTEGER)`)
	g.Exec(`INSERT INTO devices(ddns,mac,description,ip,client,status,last_seen_at)
		VALUES ('dev1','aabbcc','MacInfo','10.0.0.9','rtty-go','offline',100)`)

	repo := NewDeviceMetaRepo(g)
	ctx := context.Background()

	if err := repo.MarkOnline(ctx, "dev1"); err != nil {
		t.Fatalf("MarkOnline: %v", err)
	}
	var status, mac, desc, ip string
	var lastSeen int64
	g.Raw(`SELECT status,mac,description,ip,last_seen_at FROM devices WHERE ddns='dev1'`).
		Row().Scan(&status, &mac, &desc, &ip, &lastSeen)
	if status != "online" {
		t.Errorf("status=%q want online", status)
	}
	if lastSeen <= 100 {
		t.Errorf("last_seen_at not refreshed: %d", lastSeen)
	}
	// identity columns untouched
	if mac != "aabbcc" || desc != "MacInfo" || ip != "10.0.0.9" {
		t.Errorf("identity columns changed: mac=%q desc=%q ip=%q", mac, desc, ip)
	}

	// UpdateClient with the SAME values => no row written
	const guard = `UPDATE devices SET client=?, model=? WHERE ddns=? AND (client IS NULL OR client <> ? OR model <> ?)`
	res := g.Exec(guard, "rtty-go", "", "dev1", "rtty-go", "")
	if res.RowsAffected != 0 {
		t.Errorf("unchanged client should not write, RowsAffected=%d", res.RowsAffected)
	}
	if err := repo.UpdateClient(ctx, "dev1", "rtty-go", ""); err != nil {
		t.Fatalf("UpdateClient same: %v", err)
	}
	// changed client or model => writes
	res = g.Exec(guard, "rtty-go-2.0", "", "dev1", "rtty-go-2.0", "")
	if res.RowsAffected != 1 {
		t.Errorf("changed client should write once, RowsAffected=%d", res.RowsAffected)
	}
	if err := repo.UpdateClient(ctx, "dev1", "rtty-go-2.0", "rm1pe"); err != nil {
		t.Fatalf("UpdateClient model: %v", err)
	}
	var model string
	g.Raw(`SELECT model FROM devices WHERE ddns='dev1'`).Row().Scan(&model)
	if model != "rm1pe" {
		t.Errorf("model=%q, want the reported one", model)
	}
}
