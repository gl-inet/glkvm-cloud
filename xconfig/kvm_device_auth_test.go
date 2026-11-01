package xconfig

import "testing"

func clearKvmEnv(t *testing.T) {
    t.Helper()
    for _, k := range []string{"KVM_DEVICE_USER", "KVM_DEVICE_PASSWORD", "KVM_DEVICE_TOTP_SECRET", "KVM_DEVICE_AUTH_SCOPE"} {
        t.Setenv(k, "")
    }
}

func TestKvmDeviceAuthDefaultsOff(t *testing.T) {
    clearKvmEnv(t)
    cfg := &Config{}
    if err := applyKvmDeviceAuthEnv(cfg); err != nil {
        t.Fatal(err)
    }
    if cfg.KvmDeviceAuthEnabled() {
        t.Fatal("enabled without KVM_DEVICE_PASSWORD")
    }
    if cfg.KvmDeviceUser != "admin" || cfg.KvmDeviceAuthScope != KvmDeviceAuthScopeAll {
        t.Fatalf("defaults: user=%q scope=%q", cfg.KvmDeviceUser, cfg.KvmDeviceAuthScope)
    }
}

func TestKvmDeviceAuthFromEnv(t *testing.T) {
    clearKvmEnv(t)
    t.Setenv("KVM_DEVICE_PASSWORD", " keep spaces ")
    t.Setenv("KVM_DEVICE_TOTP_SECRET", "jbsw y3dp ehpk 3pxp")
    t.Setenv("KVM_DEVICE_AUTH_SCOPE", "Admin")
    cfg := &Config{}
    if err := applyKvmDeviceAuthEnv(cfg); err != nil {
        t.Fatal(err)
    }
    if !cfg.KvmDeviceAuthEnabled() || cfg.KvmDevicePassword != " keep spaces " {
        t.Fatalf("password = %q", cfg.KvmDevicePassword)
    }
    if cfg.KvmDeviceTOTPSecret != "JBSWY3DPEHPK3PXP" {
        t.Fatalf("totp secret = %q", cfg.KvmDeviceTOTPSecret)
    }
    if cfg.KvmDeviceAuthScope != KvmDeviceAuthScopeAdmin {
        t.Fatalf("scope = %q", cfg.KvmDeviceAuthScope)
    }
}

func TestKvmDeviceAuthRejectsBadValues(t *testing.T) {
    cases := map[string]map[string]string{
        "bad scope":          {"KVM_DEVICE_PASSWORD": "pw", "KVM_DEVICE_AUTH_SCOPE": "everyone"},
        "totp without pass":  {"KVM_DEVICE_TOTP_SECRET": "JBSWY3DPEHPK3PXP"},
        "totp not base32":    {"KVM_DEVICE_PASSWORD": "pw", "KVM_DEVICE_TOTP_SECRET": "not-base32!"},
    }
    for name, env := range cases {
        t.Run(name, func(t *testing.T) {
            clearKvmEnv(t)
            for k, v := range env {
                t.Setenv(k, v)
            }
            if err := applyKvmDeviceAuthEnv(&Config{}); err == nil {
                t.Fatal("expected error")
            }
        })
    }
}
