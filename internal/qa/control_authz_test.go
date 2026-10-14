package qa

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

// Device CONTROL plane authorization (/connect, /cmd, /web) and credential
// exposure via /api/script-info.
//
// A logged-in session alone must NOT be enough to drive a device: the caller
// also has to be allowed to see that specific device. The device LIST already
// enforced this (covered by TestE2E_GroupsRBAC); these endpoints did not, so a
// regular user could reach any device — including ungrouped ones — by id, and
// any logged-in user could read the device token and TURN credentials. This
// suite pins both boundaries so they cannot regress.
//
// Run (same env vars as the rest of the qa suite):
//
//	QA_TARGET=https://<host> QA_USER=admin QA_PASS='<password>' \
//	QA_DEV_ADDR=<host>:5912 QA_DEV_TOKEN='<rtty token>' \
//	go test ./internal/qa -run TestE2E_DeviceControlAuthz -v

// ctrl issues a request to a device control endpoint and returns the status
// code. These routes live outside /api and authenticate via the `sid` cookie
// (not the Bearer header), and they answer with redirects on success, so
// redirects are left unfollowed and the status code is the assertion.
func (c *client) ctrl(t *testing.T, method, path string, body any) int {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.cfg.target+path, rdr)
	if err != nil {
		t.Fatalf("ctrl %s %s: %v", method, path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.AddCookie(&http.Cookie{Name: "sid", Value: c.token})
	}
	hc := &http.Client{
		Timeout:       15 * time.Second,
		Transport:     &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := hc.Do(req)
	if err != nil {
		t.Fatalf("ctrl %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return resp.StatusCode
}

// controlProbes returns the three control endpoints for devid. The /cmd probe
// deliberately carries an empty body: authorization is checked before the body
// is parsed, so an allowed caller gets "invalid format" back and no command is
// ever dispatched to the device.
func controlProbes(devid string) []struct{ name, method, path string } {
	return []struct{ name, method, path string }{
		{"connect", http.MethodGet, "/connect/" + devid},
		{"cmd", http.MethodPost, "/cmd/" + devid},
		{"web", http.MethodGet, "/web/" + devid + "/http/127.0.0.1:80/"},
	}
}

func TestE2E_DeviceControlAuthz(t *testing.T) {
	cfg := loadConfig(t)
	if cfg.pass == "" {
		t.Skip("QA_PASS required")
	}
	if cfg.devAddr == "" {
		t.Skip("QA_DEV_ADDR required to register fixture devices")
	}

	admin := newClient(cfg)
	admin.login(t)

	admin.cleanupQADevices(t)
	defer admin.cleanupQADevices(t)

	dgID := admin.createDeviceGroup(t, "qae2e_authz_dg")
	ugID := admin.createUserGroup(t, "qae2e_authz_ug")
	admin.linkUGtoDG(t, ugID, []int64{dgID})
	uname, upass := "qae2e_authz_user", "Qae2ePass123!"
	admin.createNormalUser(t, uname, upass, []int64{ugID})
	defer func() {
		if uid := admin.findUserID(t, uname); uid > 0 {
			admin.del(t, fmt.Sprintf("/api/users/%d", uid))
		}
		admin.del(t, fmt.Sprintf("/api/user-groups/%d", ugID))
		admin.del(t, fmt.Sprintf("/api/device-groups/%d", dgID))
	}()

	// Two throwaway devices: one the user is entitled to, one they are not.
	// The out-of-scope device is left ungrouped, which is the case the device
	// list never showed them and the control endpoints used to allow anyway.
	inConn, inID := admin.registerQAOnline(t, "qae2e_authz_in", "02ffqaeauz01")
	defer inConn.Close()
	outConn, _ := admin.registerQAOnline(t, "qae2e_authz_out", "02ffqaeauz02")
	defer outConn.Close()
	admin.assignDevicesToGroup(t, dgID, []int64{inID})

	user := loginClient(t, cfg, uname, upass)
	if user == nil {
		t.Fatal("normal user login failed")
	}

	// Sanity: the fixture really is in/out of scope on the management plane.
	vl := user.listDevices(t, "pageSize=200")
	if findItem(vl.Items, "qae2e_authz_in") == nil {
		t.Fatal("fixture broken: user cannot see their own group's device")
	}
	if findItem(vl.Items, "qae2e_authz_out") != nil {
		t.Fatal("fixture broken: user can see the out-of-scope device")
	}

	t.Run("in_scope_device_allowed", func(t *testing.T) {
		for _, p := range controlProbes("qae2e_authz_in") {
			if got := user.ctrl(t, p.method, p.path, nil); got == http.StatusForbidden {
				t.Errorf("%s: user was denied their OWN group's device (403) — fix is too strict", p.name)
			} else {
				t.Logf("%s: http=%d (allowed)", p.name, got)
			}
		}
	})

	t.Run("out_of_scope_device_denied", func(t *testing.T) {
		for _, p := range controlProbes("qae2e_authz_out") {
			got := user.ctrl(t, p.method, p.path, nil)
			if got != http.StatusForbidden {
				t.Errorf("%s: out-of-scope device returned http=%d, want 403", p.name, got)
			} else {
				t.Logf("%s: http=403 (denied)", p.name)
			}
		}
	})

	t.Run("unknown_device_denied", func(t *testing.T) {
		for _, p := range controlProbes("qae2e_authz_nosuchdev") {
			if got := user.ctrl(t, p.method, p.path, nil); got != http.StatusForbidden {
				t.Errorf("%s: unknown device returned http=%d, want 403", p.name, got)
			}
		}
	})

	t.Run("admin_reaches_both", func(t *testing.T) {
		for _, devid := range []string{"qae2e_authz_in", "qae2e_authz_out"} {
			for _, p := range controlProbes(devid) {
				if got := admin.ctrl(t, p.method, p.path, nil); got == http.StatusForbidden {
					t.Errorf("%s on %s: admin was denied (403) — fix broke admin access", p.name, devid)
				}
			}
		}
	})

	t.Run("no_session_denied", func(t *testing.T) {
		anon := newClient(cfg) // no token => no sid cookie
		for _, p := range controlProbes("qae2e_authz_in") {
			if got := anon.ctrl(t, p.method, p.path, nil); got != http.StatusUnauthorized {
				t.Errorf("%s: anonymous request returned http=%d, want 401", p.name, got)
			}
		}
	})

	// /api/script-info hands out the device registration token and the TURN
	// credentials, so it must be restricted to roles that may add devices.
	t.Run("script_info_admin_only", func(t *testing.T) {
		_, env, err := user.do("GET", "/api/script-info", nil, true)
		if err != nil {
			t.Fatalf("script-info as user: %v", err)
		}
		if env.OK {
			t.Errorf("script-info: regular user got code=%s, want a denial", env.Code)
		} else {
			t.Logf("script-info as user: code=%s (denied)", env.Code)
		}

		_, env, err = admin.do("GET", "/api/script-info", nil, true)
		if err != nil {
			t.Fatalf("script-info as admin: %v", err)
		}
		if !env.OK {
			t.Errorf("script-info: admin got code=%s, want it to still work", env.Code)
		}
	})
}
