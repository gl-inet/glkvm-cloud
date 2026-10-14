package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"rttys/internal/domain/device"
	"rttys/internal/domain/devicelog"
	"rttys/internal/domain/notification"
	"rttys/internal/domain/permission"
	"rttys/internal/domain/user"
	"rttys/internal/http/dto"
	"rttys/internal/http/middleware"
	"rttys/internal/store/memory"
	"rttys/internal/store/sqlite"
	"rttys/xconfig"

	"github.com/gin-gonic/gin"
)

// script-info returns the device token and TURN credentials, so only roles with
// device.write (admin) may read it.
func TestScriptInfoRequiresDeviceWrite(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.Open(ctx, sqlite.Options{DSN: filepath.Join(t.TempDir(), "si.db")})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	if err := sqlite.InitSchema(ctx, db.SQL(), "../../docker-compose/database/schema.sql"); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	g := db.Gorm()
	if err := g.Exec(`INSERT INTO users (id, username, password_hash, role, status) VALUES
		(1, 'admin', 'x', 'admin', 'active'), (2, 'alice', 'x', 'user', 'active')`).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}

	store := memory.NewSessionStore(time.Hour)
	store.Create("sid-admin", 1)
	store.Create("sid-alice", 2)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.Trace())
	groupRepo := sqlite.NewGroupRepo(g)
	RegisterAPIRoutes(r, Deps{
		UserSvc:           user.NewService(sqlite.NewUserRepo(g)),
		PermSvc:           permission.NewService(memory.NewPermissionRepo()),
		DevSvc:            device.NewService(sqlite.NewDeviceRepo(g), groupRepo),
		GroupRepo:         groupRepo,
		SessionStore:      store,
		RelationsRepo:     sqlite.NewRelationsRepo(g),
		TrustedDeviceRepo: sqlite.NewTrustedDeviceRepo(g),
		DeviceLogSvc:      devicelog.NewService(sqlite.NewDeviceLogRepo(g)),
		NotificationSvc:   notification.NewService(sqlite.NewNotificationRepo(g)),
		Cfg:               &xconfig.Config{Token: "device-token"},
	})

	get := func(sid string) (string, map[string]any) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/script-info", nil)
		req.AddCookie(&http.Cookie{Name: "sid", Value: sid})
		r.ServeHTTP(w, req)
		var body struct {
			Code string         `json:"code"`
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode %q: %v", w.Body.String(), err)
		}
		return body.Code, body.Data
	}

	if code, data := get("sid-alice"); code != dto.CodeForbidden || data["token"] != nil {
		t.Errorf("regular user: code=%q token=%v, want %q and no token", code, data["token"], dto.CodeForbidden)
	}
	if code, data := get("sid-admin"); code != dto.CodeOK || data["token"] != "device-token" {
		t.Errorf("admin: code=%q token=%v, want %q and the device token", code, data["token"], dto.CodeOK)
	}
}
