package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"rttys/internal/domain/user"
	"rttys/internal/store/memory"
	"rttys/internal/store/sqlite"
	"rttys/xconfig"

	"github.com/gin-gonic/gin"
)

// setupAccessFixture installs a temp DB as the global container plus a fresh
// session store, with one session per user (sid "sid-<username>"):
//
//	admin (admin)       alice (user, ug1)    bob (user, no group)    carol (user, ug1, disabled)
//	ug1 -> dg1          dev-g1 in dg1        dev-g2 in dg2 (not linked)    dev-none ungrouped
func setupAccessFixture(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	db, err := sqlite.Open(ctx, sqlite.Options{DSN: filepath.Join(t.TempDir(), "access.db")})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := sqlite.InitSchema(ctx, db.SQL(), "../../docker-compose/database/schema.sql"); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	g := db.Gorm()

	stmts := []string{
		`INSERT INTO users (id, username, password_hash, role, status) VALUES
			(1, 'admin', 'x', 'admin', 'active'),
			(2, 'alice', 'x', 'user',  'active'),
			(3, 'bob',   'x', 'user',  'active'),
			(4, 'carol', 'x', 'user',  'disabled')`,
		`INSERT INTO user_groups (id, name) VALUES (1, 'ug1')`,
		`INSERT INTO user_group_members (user_id, group_id) VALUES (2, 1), (4, 1)`,
		`INSERT INTO device_groups (id, name) VALUES (1, 'dg1'), (2, 'dg2')`,
		`INSERT INTO user_group_device_group_links (user_group_id, device_group_id) VALUES (1, 1)`,
		`INSERT INTO devices (ddns, mac, device_group_id) VALUES
			('dev-g1',   'aa0000000001', 1),
			('dev-g2',   'aa0000000002', 2),
			('dev-none', 'aa0000000003', NULL)`,
	}
	for _, s := range stmts {
		if err := g.Exec(s).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, s)
		}
	}

	prevStore, prevCont := sessionStore, sqlite.TryContainer()
	sessionStore = memory.NewSessionStore(time.Hour)
	for id, name := range map[int64]string{1: "admin", 2: "alice", 3: "bob", 4: "carol"} {
		sessionStore.Create("sid-"+name, id)
	}
	sqlite.SetContainer(&sqlite.Container{Gorm: g, UserSvc: user.NewService(sqlite.NewUserRepo(g))})

	t.Cleanup(func() {
		sessionStore = prevStore
		sqlite.SetContainer(prevCont)
		db.Close()
	})
}

func ctxWithSid(sid string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/connect/x", nil)
	if sid != "" {
		req.AddCookie(&http.Cookie{Name: "sid", Value: sid})
	}
	c.Request = req
	return c, w
}

func TestCanAccessDevice(t *testing.T) {
	setupAccessFixture(t)

	cases := []struct {
		sid, devid string
		want       bool
	}{
		{"sid-admin", "dev-g1", true},
		{"sid-admin", "dev-g2", true},
		{"sid-admin", "dev-none", true}, // admin also reaches ungrouped devices
		{"sid-alice", "dev-g1", true},   // in a device group linked to her user group
		{"sid-alice", "dev-g2", false},  // device group not linked to her
		{"sid-alice", "dev-none", false},
		{"sid-alice", "dev-unknown", false},
		{"sid-alice", " ", false},
		{"sid-bob", "dev-g1", false},   // no user group at all
		{"sid-carol", "dev-g1", false}, // disabled account
		{"sid-bogus", "dev-g1", false}, // unknown session
		{"", "dev-g1", false},          // no session
	}
	for _, tc := range cases {
		c, _ := ctxWithSid(tc.sid)
		if got := canAccessDevice(c, tc.devid); got != tc.want {
			t.Errorf("canAccessDevice(sid=%q, devid=%q) = %v, want %v", tc.sid, tc.devid, got, tc.want)
		}
	}
}

func TestHttpAuthAlwaysRequiresSession(t *testing.T) {
	setupAccessFixture(t)

	// Settings that used to skip authentication must no longer do so.
	cfg := &xconfig.Config{Password: "", LocalAuth: false}

	c, _ := ctxWithSid("")
	c.Request.RemoteAddr = "127.0.0.1:50000"
	if httpAuth(cfg, c) {
		t.Error("httpAuth allowed a loopback request without a session")
	}

	c, _ = ctxWithSid("sid-bogus")
	if httpAuth(cfg, c) {
		t.Error("httpAuth allowed an unknown session")
	}

	c, _ = ctxWithSid("sid-alice")
	if !httpAuth(cfg, c) {
		t.Error("httpAuth rejected a valid session")
	}
}

func TestForbidDeviceAccess(t *testing.T) {
	c, w := ctxWithSid("")
	forbidDeviceAccess(c)
	if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "<html") {
		t.Errorf("page request: code=%d, want 403 with the HTML error page", w.Code)
	}

	c, w = ctxWithSid("")
	c.Request.Header.Set("Upgrade", "websocket")
	forbidDeviceAccess(c)
	if w.Code != http.StatusForbidden || w.Body.Len() != 0 {
		t.Errorf("websocket request: code=%d body=%d bytes, want bare 403", w.Code, w.Body.Len())
	}
}
