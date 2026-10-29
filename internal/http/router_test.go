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
	r            *gin.Engine
	access       *sqlite.AccessRepo
	db           *sqlite.AppDB
	disconnected []string // devices DisconnectDevice was called for
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
	f := &apiFixture{r: r, access: sqlite.NewAccessRepo(g), db: db}
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
		AccessRepo:        f.access,
		DisconnectDevice:  func(devid string) { f.disconnected = append(f.disconnected, devid) },
		Cfg:               cfg,
	})
	return f
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

	// The add device dialog polls just the codes it generated.
	env = f.call(t, http.MethodGet, "/api/registration-codes?ids="+strconv.FormatInt(second, 10), "sid-admin", nil)
	list = dto.ListRegistrationCodesResp{}
	_ = json.Unmarshal(env.Data, &list)
	if env.Code != dto.CodeOK || len(list.Items) != 1 || list.Items[0].ID != second || list.Items[0].Status != "revoked" {
		t.Errorf("by id: code=%q items=%+v, want just the revoked second code", env.Code, list.Items)
	}
	for _, bad := range []string{"x", "1,,2", "0", "-3"} {
		if env := f.call(t, http.MethodGet, "/api/registration-codes?ids="+bad, "sid-admin", nil); env.Code != dto.CodeInvalidArgument {
			t.Errorf("ids=%q: code=%q, want %q", bad, env.Code, dto.CodeInvalidArgument)
		}
	}
}

func TestDeviceAccessAPI(t *testing.T) {
	f := newAPI(t, &xconfig.Config{WebrtcIP: "192.0.2.10", AddrDevMTLS: ":15915", OutdatedBundle: true})
	if env := f.call(t, http.MethodGet, "/api/device-access", "sid-alice", nil); env.Code != dto.CodeForbidden {
		t.Errorf("regular user: code=%q, want %q", env.Code, dto.CodeForbidden)
	}
	env := f.call(t, http.MethodGet, "/api/device-access", "sid-admin", nil)
	var got dto.DeviceAccessResp
	if err := json.Unmarshal(env.Data, &got); err != nil || env.Code != dto.CodeOK {
		t.Fatalf("admin: code=%q err=%v", env.Code, err)
	}
	if want := (dto.DeviceAccessResp{Addr: "192.0.2.10:15915", Port: "15915", OutdatedBundle: true}); got != want {
		t.Errorf("device access = %+v, want %+v", got, want)
	}

	// No access address yet: the port is still known.
	f = newAPI(t, &xconfig.Config{AddrDevMTLS: ":5915"})
	env = f.call(t, http.MethodGet, "/api/device-access", "sid-admin", nil)
	got = dto.DeviceAccessResp{}
	_ = json.Unmarshal(env.Data, &got)
	if want := (dto.DeviceAccessResp{Port: "5915"}); got != want {
		t.Errorf("no address: device access = %+v, want %+v", got, want)
	}
}

func TestRegistrationCodesNeedAccessAddress(t *testing.T) {
	d, _ := pki.LoadOrCreate(t.TempDir())
	f := newAPI(t, &xconfig.Config{AddrDevMTLS: ":5915", DeviceCAFile: d.CAFile()})
	if env := f.call(t, http.MethodPost, "/api/registration-codes", "sid-admin", nil); env.Code != dto.CodeValidationFailed {
		t.Errorf("no access address: code=%q, want %q", env.Code, dto.CodeValidationFailed)
	}
}

func TestDeleteDeviceRevokesAccess(t *testing.T) {
	f := newAPI(t, &xconfig.Config{})
	ctx := context.Background()
	g := f.db.Gorm()
	now := time.Now().Unix()

	if err := g.Exec(`INSERT INTO registration_codes (code_hash, created_at, expires_at) VALUES (?, ?, ?)`,
		access.HashCode("c"), now, now+60).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := f.access.Claim(ctx, access.HashCode("c"), "9483c4b71fb1", "zh71fb1", now); err != nil {
		t.Fatal(err)
	}
	if err := g.Exec(`INSERT INTO devices (ddns, mac) VALUES ('zh71fb1', '9483c4b71fb1')`).Error; err != nil {
		t.Fatal(err)
	}
	var id int64
	g.Raw(`SELECT id FROM devices WHERE ddns = 'zh71fb1'`).Scan(&id)
	path := "/api/devices/" + strconv.FormatInt(id, 10)

	if env := f.call(t, http.MethodDelete, path, "sid-alice", nil); env.Code != dto.CodeForbidden {
		t.Errorf("regular user delete: code=%q, want %q", env.Code, dto.CodeForbidden)
	}
	if env := f.call(t, http.MethodDelete, path, "sid-admin", nil); env.Code != dto.CodeOK {
		t.Fatalf("admin delete: code=%q", env.Code)
	}

	if e, _ := f.access.GetByMAC(ctx, "9483c4b71fb1"); e != nil {
		t.Error("deleted device is still whitelisted")
	}
	if len(f.disconnected) != 1 || f.disconnected[0] != "zh71fb1" {
		t.Errorf("disconnected = %v, want [zh71fb1]", f.disconnected)
	}
	var actor string
	g.Raw(`SELECT actor_name FROM device_event_logs WHERE event_type = 'device_removed' AND device_id = 'zh71fb1'`).Scan(&actor)
	if actor != "admin" {
		t.Errorf("device_removed actor = %q, want admin", actor)
	}

	if env := f.call(t, http.MethodDelete, path, "sid-admin", nil); env.Code != dto.CodeNotFound {
		t.Errorf("second delete: code=%q, want %q", env.Code, dto.CodeNotFound)
	}
}
