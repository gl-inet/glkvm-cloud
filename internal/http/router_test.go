package http

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"rttys/internal/domain/access"
	"rttys/internal/domain/device"
	"rttys/internal/domain/devicelog"
	"rttys/internal/domain/notification"
	"rttys/internal/domain/permission"
	"rttys/internal/domain/user"
	"rttys/internal/http/dto"
	"rttys/internal/http/middleware"
	"rttys/internal/pkg/pki"
	"rttys/internal/store/memory"
	"rttys/internal/store/sqlite"
	"rttys/xconfig"

	"github.com/gin-gonic/gin"
)

type apiFixture struct {
	r      *gin.Engine
	access *sqlite.AccessRepo
}

// newAPI serves the API routes over a temp database with one admin (session
// "sid-admin") and one regular user ("sid-alice").
func newAPI(t *testing.T, cfg *xconfig.Config) *apiFixture {
	t.Helper()
	ctx := context.Background()
	db, err := sqlite.Open(ctx, sqlite.Options{DSN: filepath.Join(t.TempDir(), "api.db")})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
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
	accessRepo := sqlite.NewAccessRepo(g)
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
		AccessRepo:        accessRepo,
		Cfg:               cfg,
	})
	return &apiFixture{r: r, access: accessRepo}
}

type envelope struct {
	Code string          `json:"code"`
	Data json.RawMessage `json:"data"`
}

func (f *apiFixture) call(t *testing.T, method, path, sid string, body any) envelope {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, rdr)
	req.AddCookie(&http.Cookie{Name: "sid", Value: sid})
	f.r.ServeHTTP(w, req)
	var env envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("%s %s: decode %q: %v", method, path, w.Body.String(), err)
	}
	return env
}

// script-info returns the device token and TURN credentials, so only roles with
// device.write (admin) may read it.
func TestScriptInfoRequiresDeviceWrite(t *testing.T) {
	f := newAPI(t, &xconfig.Config{Token: "device-token"})

	var data map[string]any
	env := f.call(t, http.MethodGet, "/api/script-info", "sid-alice", nil)
	_ = json.Unmarshal(env.Data, &data)
	if env.Code != dto.CodeForbidden || data["token"] != nil {
		t.Errorf("regular user: code=%q token=%v, want %q and no token", env.Code, data["token"], dto.CodeForbidden)
	}

	env = f.call(t, http.MethodGet, "/api/script-info", "sid-admin", nil)
	data = nil
	_ = json.Unmarshal(env.Data, &data)
	if env.Code != dto.CodeOK || data["token"] != "device-token" {
		t.Errorf("admin: code=%q token=%v, want %q and the device token", env.Code, data["token"], dto.CodeOK)
	}
}

func TestRegistrationCodesAPI(t *testing.T) {
	d, err := pki.LoadOrCreate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := newAPI(t, &xconfig.Config{WebrtcIP: "192.0.2.10", AddrDevMTLS: ":5915", DeviceCAFile: d.CAFile()})

	// Admin only.
	if env := f.call(t, http.MethodPost, "/api/registration-codes", "sid-alice", nil); env.Code != dto.CodeForbidden {
		t.Errorf("regular user create: code=%q, want %q", env.Code, dto.CodeForbidden)
	}
	if env := f.call(t, http.MethodPost, "/api/registration-codes", "sid-admin", map[string]int{"count": access.MaxCodesPerRequest + 1}); env.Code != dto.CodeInvalidArgument {
		t.Errorf("too many codes: code=%q, want %q", env.Code, dto.CodeInvalidArgument)
	}

	env := f.call(t, http.MethodPost, "/api/registration-codes", "sid-admin", map[string]int{"count": 2})
	var created dto.CreateRegistrationCodesResp
	if err := json.Unmarshal(env.Data, &created); err != nil || env.Code != dto.CodeOK || len(created.Items) != 2 {
		t.Fatalf("create: code=%q items=%d err=%v", env.Code, len(created.Items), err)
	}

	// The access code tells the device where to connect and what to trust.
	ac, err := access.ParseAccessCode(created.Items[0].AccessCode)
	if err != nil {
		t.Fatal(err)
	}
	if ac.Addr != "192.0.2.10:5915" {
		t.Errorf("access code address = %q", ac.Addr)
	}
	if ca, err := x509.ParseCertificate(ac.CA); err != nil || !ca.IsCA {
		t.Errorf("access code CA does not parse as a CA: %v", err)
	}
	if ttl := created.Items[0].ExpiresAt - time.Now().Unix(); ttl < 23*3600 || ttl > 24*3600 {
		t.Errorf("code lifetime = %ds, want 24h", ttl)
	}

	// Redeem the first code, revoke the second.
	if _, err := f.access.Claim(context.Background(), access.HashCode(ac.Code), "9483c4b71fb1", "zh71fb1", time.Now().Unix()); err != nil {
		t.Fatalf("access code's code did not redeem: %v", err)
	}
	second := created.Items[1].ID
	if env := f.call(t, http.MethodDelete, "/api/registration-codes/"+strconv.FormatInt(second, 10), "sid-admin", nil); env.Code != dto.CodeOK {
		t.Errorf("revoke: code=%q", env.Code)
	}
	if env := f.call(t, http.MethodDelete, "/api/registration-codes/"+strconv.FormatInt(second, 10), "sid-admin", nil); env.Code != dto.CodeConflict {
		t.Errorf("revoke twice: code=%q, want %q", env.Code, dto.CodeConflict)
	}

	env = f.call(t, http.MethodGet, "/api/registration-codes", "sid-admin", nil)
	var list dto.ListRegistrationCodesResp
	_ = json.Unmarshal(env.Data, &list)
	status := map[int64]dto.RegistrationCode{}
	for _, it := range list.Items {
		status[it.ID] = it
	}
	if got := status[created.Items[0].ID]; got.Status != "used" || got.Devid != "zh71fb1" {
		t.Errorf("first code = %+v, want used by zh71fb1", got)
	}
	if got := status[second]; got.Status != "revoked" {
		t.Errorf("second code = %+v, want revoked", got)
	}
}

func TestRegistrationCodesNeedAccessAddress(t *testing.T) {
	d, _ := pki.LoadOrCreate(t.TempDir())
	f := newAPI(t, &xconfig.Config{AddrDevMTLS: ":5915", DeviceCAFile: d.CAFile()})
	if env := f.call(t, http.MethodPost, "/api/registration-codes", "sid-admin", nil); env.Code != dto.CodeValidationFailed {
		t.Errorf("no access address: code=%q, want %q", env.Code, dto.CodeValidationFailed)
	}
}
