package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"rttys/internal/domain/access"
)

func newAccessFixture(t *testing.T) (*AccessRepo, *AppDB) {
	t.Helper()
	ctx := context.Background()
	db, err := Open(ctx, Options{DSN: filepath.Join(t.TempDir(), "access.db")})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := InitSchema(ctx, db.SQL(), "schema.sql"); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	return NewAccessRepo(db.Gorm()), db
}

const now = int64(1_800_000_000)

func addCode(t *testing.T, db *AppDB, code string, expiresAt int64, revoked bool) {
	t.Helper()
	var rev any
	if revoked {
		rev = now - 1
	}
	if err := db.Gorm().Exec(
		`INSERT INTO registration_codes (code_hash, created_at, expires_at, revoked_at) VALUES (?, ?, ?, ?)`,
		access.HashCode(code), now-10, expiresAt, rev,
	).Error; err != nil {
		t.Fatal(err)
	}
}

func codeUsed(t *testing.T, db *AppDB, code string) bool {
	t.Helper()
	var n int64
	db.Gorm().Raw(`SELECT COUNT(1) FROM registration_codes WHERE code_hash = ? AND used_at IS NOT NULL`, access.HashCode(code)).Scan(&n)
	return n == 1
}

func TestClaimJoinsWhitelist(t *testing.T) {
	repo, db := newAccessFixture(t)
	ctx := context.Background()
	addCode(t, db, "code-a", now+3600, false)

	devid, err := repo.Claim(ctx, access.HashCode("code-a"), "94:83:C4:B7:1F:B1", "zh71fb1", now)
	if err != nil || devid != "zh71fb1" {
		t.Fatalf("claim = (%q, %v), want zh71fb1", devid, err)
	}
	e, err := repo.GetByMAC(ctx, "9483c4b71fb1")
	if err != nil || e == nil || e.Devid != "zh71fb1" || e.CodeID == 0 {
		t.Fatalf("whitelist entry = %+v, %v", e, err)
	}
	if !codeUsed(t, db, "code-a") {
		t.Error("code was not marked used")
	}

	// One-time: the same code cannot admit another device.
	if _, err := repo.Claim(ctx, access.HashCode("code-a"), "9483c4e1ddff", "dl1ddff", now); !errors.Is(err, access.ErrCodeInvalid) {
		t.Errorf("reused code: err = %v, want ErrCodeInvalid", err)
	}
}

func TestClaimRejectsBadCodes(t *testing.T) {
	repo, db := newAccessFixture(t)
	ctx := context.Background()
	addCode(t, db, "expired", now-1, false)
	addCode(t, db, "revoked", now+3600, true)

	for _, code := range []string{"expired", "revoked", "never-issued"} {
		if _, err := repo.Claim(ctx, access.HashCode(code), "9483c4b71fb1", "zh71fb1", now); !errors.Is(err, access.ErrCodeInvalid) {
			t.Errorf("%s: err = %v, want ErrCodeInvalid", code, err)
		}
	}
	if e, _ := repo.GetByMAC(ctx, "9483c4b71fb1"); e != nil {
		t.Errorf("rejected claims still listed the device: %+v", e)
	}
}

func TestClaimDeviceIDOwnership(t *testing.T) {
	repo, db := newAccessFixture(t)
	ctx := context.Background()
	g := db.Gorm()

	// zh71fb1 is already whitelisted for another MAC.
	addCode(t, db, "first", now+3600, false)
	if _, err := repo.Claim(ctx, access.HashCode("first"), "aa0000000001", "zh71fb1", now); err != nil {
		t.Fatal(err)
	}
	addCode(t, db, "second", now+3600, false)
	if _, err := repo.Claim(ctx, access.HashCode("second"), "aa0000000002", "zh71fb1", now); !errors.Is(err, access.ErrDevidTaken) {
		t.Errorf("whitelisted ID: err = %v, want ErrDevidTaken", err)
	}
	if codeUsed(t, db, "second") {
		t.Error("a rejected claim consumed the code")
	}

	// tokdev is a token-era device record with another MAC.
	if err := g.Exec(`INSERT INTO devices (ddns, mac) VALUES ('tokdev', 'aa0000000003')`).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Claim(ctx, access.HashCode("second"), "aa0000000002", "tokdev", now); !errors.Is(err, access.ErrDevidTaken) {
		t.Errorf("device record of another MAC: err = %v, want ErrDevidTaken", err)
	}

	// The same MAC connected before with a token under "oldid": it keeps that
	// record even though it now registers as "newid".
	if err := g.Exec(`INSERT INTO devices (ddns, mac) VALUES ('oldid', 'aa0000000004')`).Error; err != nil {
		t.Fatal(err)
	}
	addCode(t, db, "third", now+3600, false)
	devid, err := repo.Claim(ctx, access.HashCode("third"), "aa:00:00:00:00:04", "newid", now)
	if err != nil || devid != "oldid" {
		t.Errorf("token-era device: claim = (%q, %v), want oldid", devid, err)
	}
}

func TestCodesLifecycle(t *testing.T) {
	repo, _ := newAccessFixture(t)
	ctx := context.Background()

	ids, err := repo.CreateCodes(ctx, []string{access.HashCode("one"), access.HashCode("two")}, 7, now, now+3600)
	if err != nil || len(ids) != 2 || ids[0] == ids[1] {
		t.Fatalf("CreateCodes = (%v, %v)", ids, err)
	}

	if _, err := repo.Claim(ctx, access.HashCode("one"), "9483c4b71fb1", "zh71fb1", now+5); err != nil {
		t.Fatal(err)
	}
	if err := repo.RevokeCode(ctx, ids[1], now+5); err != nil {
		t.Fatalf("revoke unused: %v", err)
	}

	codes, err := repo.ListCodes(ctx, 50)
	if err != nil || len(codes) != 2 {
		t.Fatalf("ListCodes = (%v, %v)", codes, err)
	}
	byID := map[int64]access.RegistrationCode{codes[0].ID: codes[0], codes[1].ID: codes[1]}
	used, revoked := byID[ids[0]], byID[ids[1]]
	if used.Status(now+5) != access.CodeUsed || used.Devid != "zh71fb1" || used.UsedMAC != "9483c4b71fb1" || used.CreatedBy != 7 {
		t.Errorf("used code = %+v", used)
	}
	if revoked.Status(now+5) != access.CodeRevoked {
		t.Errorf("revoked code = %+v", revoked)
	}

	if err := repo.RevokeCode(ctx, ids[0], now+5); !errors.Is(err, access.ErrCodeNotRevocable) {
		t.Errorf("revoke used: %v, want ErrCodeNotRevocable", err)
	}
	if err := repo.RevokeCode(ctx, 9999, now+5); !errors.Is(err, access.ErrCodeNotFound) {
		t.Errorf("revoke unknown: %v, want ErrCodeNotFound", err)
	}

	late, _ := repo.CreateCodes(ctx, []string{access.HashCode("three")}, 7, now, now+10)
	if err := repo.RevokeCode(ctx, late[0], now+20); !errors.Is(err, access.ErrCodeNotRevocable) {
		t.Errorf("revoke expired: %v, want ErrCodeNotRevocable", err)
	}
}

func TestRemoveDevice(t *testing.T) {
	repo, db := newAccessFixture(t)
	ctx := context.Background()
	addCode(t, db, "c", now+3600, false)
	if _, err := repo.Claim(ctx, access.HashCode("c"), "9483c4b71fb1", "zh71fb1", now); err != nil {
		t.Fatal(err)
	}
	if err := db.Gorm().Exec(`INSERT INTO devices (ddns, mac) VALUES ('zh71fb1', '9483c4b71fb1')`).Error; err != nil {
		t.Fatal(err)
	}
	var id int64
	db.Gorm().Raw(`SELECT id FROM devices WHERE ddns = 'zh71fb1'`).Scan(&id)

	devid, mac, err := repo.RemoveDevice(ctx, id)
	if err != nil || devid != "zh71fb1" || mac != "9483c4b71fb1" {
		t.Fatalf("RemoveDevice = (%q, %q, %v)", devid, mac, err)
	}
	if e, _ := repo.GetByMAC(ctx, mac); e != nil {
		t.Error("whitelist entry survived the delete")
	}
	if _, _, err := repo.RemoveDevice(ctx, id); !errors.Is(err, access.ErrDeviceNotFound) {
		t.Errorf("second delete: %v, want ErrDeviceNotFound", err)
	}
}

// An upgraded deployment keeps the schema.sql of its old release, which has no
// access tables, so InitSchema has to create them itself.
func TestInitSchemaCreatesAccessTables(t *testing.T) {
	_, db := newAccessFixture(t)
	for _, table := range []string{"device_whitelist", "registration_codes"} {
		var n int64
		db.Gorm().Raw(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&n)
		if n != 1 {
			t.Errorf("table %s was not created", table)
		}
	}
}
