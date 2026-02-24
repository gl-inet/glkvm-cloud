/*
 * MIT License
 *
 * Copyright (c) 2019 Jianhui Zhao <zhaojh329@gmail.com>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package xconfig

import (
    "fmt"
    "os"
    "strconv"
    "strings"
    "time"

    "github.com/kylelemons/go-gypsy/yaml"
)

type Config struct {
    AddrDev              string
    AddrUser             string
    AddrHttpProxy        string
    HttpProxyRedirURL    string
    HttpProxyRedirDomain string
    Token                string
    DevHookUrl           string
    UserHookUrl          string
    LocalAuth            bool
    Password             string
    AllowOrigins         bool
    PprofAddr            string
    AuthSessionTTL       time.Duration
    SslCert              string
    SslKey               string
    LogPath              string
    LogLevel             string
    Verbose              bool
    WebrtcIP             string
    WebrtcPort           string
    WebrtcUsername       string
    WebrtcPassword       string
    // // LDAP Configuration
    LdapEnabled       bool
    LdapServer        string
    LdapPort          int
    LdapUseTLS        bool
    LdapBindDN        string
    LdapBindPassword  string
    LdapBaseDN        string
    LdapUserFilter    string
    LdapAllowedGroups string
    LdapAllowedUsers  string

    // Generic OIDC Provider (supports any standard OIDC provider)
    OIDCEnabled                 bool
    OIDCGenericIssuer           string
    OIDCGenericClientID         string
    OIDCGenericClientSecret     string
    OIDCGenericAuthURL          string
    OIDCGenericTokenURL         string
    OIDCGenericRedirectURL      string
    OIDCGenericScopes           []string
    OIDCGenericAllowedUsers     []string
    OIDCGenericAllowedSubs      []string
    OIDCGenericAllowedUsernames []string
    OIDCGenericAllowedGroups    []string

    // =====================================================
    // Reverse Proxy / Proxy Mode
    // =====================================================
    // Enable proxy mode (app is behind Nginx/Traefik/Caddy/Cloudflare)
    ReverseProxyEnabled bool

    // =====================================================
    // Device Remote Access
    // =====================================================
    // Host[:port] used to generate device remote access address:
    //   <deviceId>.<DEVICE_ENDPOINT_HOST>
    DeviceEndpointHost string

    // Platform access domain restriction.
    // When set, only requests with a matching domain are allowed to access the platform.
    WebUIHost string
}

// docker mode fixed path for reading certificate
const (
    SslCert = "/home/certificate/glkvm_cer"
    SslKey  = "/home/certificate/glkvm_key"
)

func (cfg *Config) Load(confPath string) error {
    cfg.SslCert = SslCert
    cfg.SslKey = SslKey
    if cfg.AuthSessionTTL == 0 {
        cfg.AuthSessionTTL = 24 * time.Hour
    }

    if strings.TrimSpace(confPath) != "" {
        if _, err := os.Stat(confPath); err == nil {
            if err := parseYamlCfg(cfg, confPath); err != nil {
                return err
            }
        } else if !os.IsNotExist(err) {
            return fmt.Errorf("read config file: %s", err.Error())
        }
    }

    if err := applyEnvCfg(cfg); err != nil {
        return err
    }

    return nil
}

func getConfigOpt(yamlCfg *yaml.File, name string, opt any) {
    val, err := yamlCfg.Get(name)
    if err != nil {
        return
    }

    switch opt := opt.(type) {
    case *string:
        *opt = val
    case *int:
        *opt, _ = strconv.Atoi(val)
    case *bool:
        *opt, _ = strconv.ParseBool(val)
    }
}

func parseYamlCfg(cfg *Config, conf string) error {
    yamlCfg, err := yaml.ReadFile(conf)
    if err != nil {
        return fmt.Errorf(`read config file: %s`, err.Error())
    }

    getConfigOpt(yamlCfg, "log", &cfg.LogPath)
    getConfigOpt(yamlCfg, "log-level", &cfg.LogLevel)
    getConfigOpt(yamlCfg, "verbose", &cfg.Verbose)

    getConfigOpt(yamlCfg, "addr-dev", &cfg.AddrDev)
    getConfigOpt(yamlCfg, "addr-user", &cfg.AddrUser)
    getConfigOpt(yamlCfg, "addr-http-proxy", &cfg.AddrHttpProxy)
    getConfigOpt(yamlCfg, "http-proxy-redir-url", &cfg.HttpProxyRedirURL)
    getConfigOpt(yamlCfg, "http-proxy-redir-domain", &cfg.HttpProxyRedirDomain)

    getConfigOpt(yamlCfg, "token", &cfg.Token)
    getConfigOpt(yamlCfg, "dev-hook-url", &cfg.DevHookUrl)
    getConfigOpt(yamlCfg, "user-hook-url", &cfg.UserHookUrl)
    getConfigOpt(yamlCfg, "local-auth", &cfg.LocalAuth)
    getConfigOpt(yamlCfg, "password", &cfg.Password)
    getConfigOpt(yamlCfg, "allow-origins", &cfg.AllowOrigins)

    if err := getDurationOpt(yamlCfg, "auth-session-ttl", &cfg.AuthSessionTTL); err != nil {
        return err
    }

    getConfigOpt(yamlCfg, "webrtc-ip", &cfg.WebrtcIP)
    getConfigOpt(yamlCfg, "webrtc-port", &cfg.WebrtcPort)
    getConfigOpt(yamlCfg, "webrtc-username", &cfg.WebrtcUsername)
    getConfigOpt(yamlCfg, "webrtc-password", &cfg.WebrtcPassword)

    // LDAP配置 (LDAP Configuration)
    getConfigOpt(yamlCfg, "ldap-enabled", &cfg.LdapEnabled)
    getConfigOpt(yamlCfg, "ldap-server", &cfg.LdapServer)
    getConfigOpt(yamlCfg, "ldap-port", &cfg.LdapPort)
    getConfigOpt(yamlCfg, "ldap-use-tls", &cfg.LdapUseTLS)
    getConfigOpt(yamlCfg, "ldap-bind-dn", &cfg.LdapBindDN)

    // Note: ldap-bind-password is intentionally not read from YAML to avoid special character parsing issues and for security.
    // It's always read directly from the LDAP_BIND_PASSWORD environment variable below
    getConfigOpt(yamlCfg, "ldap-base-dn", &cfg.LdapBaseDN)
    getConfigOpt(yamlCfg, "ldap-user-filter", &cfg.LdapUserFilter)
    getConfigOpt(yamlCfg, "ldap-allowed-groups", &cfg.LdapAllowedGroups)
    getConfigOpt(yamlCfg, "ldap-allowed-users", &cfg.LdapAllowedUsers)

    // ===== OIDC configuration (generic OIDC provider) =====
    // Switch and basic endpoints
    getConfigOpt(yamlCfg, "oidc-enabled", &cfg.OIDCEnabled)
    getConfigOpt(yamlCfg, "oidc-generic-issuer", &cfg.OIDCGenericIssuer)
    getConfigOpt(yamlCfg, "oidc-generic-client-id", &cfg.OIDCGenericClientID)

    getConfigOpt(yamlCfg, "oidc-generic-auth-url", &cfg.OIDCGenericAuthURL)
    getConfigOpt(yamlCfg, "oidc-generic-token-url", &cfg.OIDCGenericTokenURL)
    getConfigOpt(yamlCfg, "oidc-generic-redirect-url", &cfg.OIDCGenericRedirectURL)

    // OIDC scopes (string can be space- or comma-separated, parsed by splitScopes)
    if s, err := yamlCfg.Get("oidc-generic-scopes"); err == nil && strings.TrimSpace(s) != "" {
        cfg.OIDCGenericScopes = splitScopes(s)
    }
    // Default scopes (when OIDC is enabled but scopes are still empty)
    if cfg.OIDCEnabled && len(cfg.OIDCGenericScopes) == 0 {
        cfg.OIDCGenericScopes = []string{"openid", "profile", "email"}
    }

    // Whitelists for OIDC logins, all parsed via splitScopes (space/comma/newline separated)
    // 1) Email-based whitelist
    if s, err := yamlCfg.Get("oidc-generic-allowed-users"); err == nil && strings.TrimSpace(s) != "" {
        cfg.OIDCGenericAllowedUsers = splitScopes(s)
    }

    // 2) Subject (sub) whitelist
    if s, err := yamlCfg.Get("oidc-generic-allowed-subs"); err == nil && strings.TrimSpace(s) != "" {
        cfg.OIDCGenericAllowedSubs = splitScopes(s)
    }

    // 3) Username whitelist (preferred_username / name)
    if s, err := yamlCfg.Get("oidc-generic-allowed-usernames"); err == nil && strings.TrimSpace(s) != "" {
        cfg.OIDCGenericAllowedUsernames = splitScopes(s)
    }

    // 4) Groups whitelist
    if s, err := yamlCfg.Get("oidc-generic-allowed-groups"); err == nil && strings.TrimSpace(s) != "" {
        cfg.OIDCGenericAllowedGroups = splitScopes(s)
    }

    return nil
}

func applyEnvCfg(cfg *Config) error {
    if v := strings.TrimSpace(os.Getenv("RTTYS_LOG")); v != "" {
        cfg.LogPath = v
    }
    if v := strings.TrimSpace(os.Getenv("RTTYS_LOG_LEVEL")); v != "" {
        cfg.LogLevel = v
    }
    if v := strings.TrimSpace(os.Getenv("RTTYS_VERBOSE")); v != "" {
        if b, err := strconv.ParseBool(v); err == nil {
            cfg.Verbose = b
        }
    }
    if v := strings.TrimSpace(os.Getenv("RTTYS_SESSION_TTL")); v != "" {
        d, err := time.ParseDuration(v)
        if err != nil {
            return fmt.Errorf("invalid RTTYS_SESSION_TTL value %q: %w", v, err)
        }
        cfg.AuthSessionTTL = d
    }

    // LDAP password is always read from environment variable to avoid YAML special character parsing issues and for security.
    if envPassword := os.Getenv("LDAP_BIND_PASSWORD"); envPassword != "" {
        cfg.LdapBindPassword = envPassword
    }

    // Note: oidc-generic-client-secret is intentionally not read from YAML
    // to avoid checking secrets into config files and leaking in logs.
    // It is always read directly from the OIDC_CLIENT_SECRET environment variable below.
    if envSecret := os.Getenv("OIDC_CLIENT_SECRET"); envSecret != "" {
        cfg.OIDCGenericClientSecret = envSecret
    }

    // Reverse proxy mode is always read from environment variable
    // to avoid config drift when running behind different proxies per deployment.
    if v := strings.TrimSpace(os.Getenv("REVERSE_PROXY_ENABLED")); v != "" {
        // Accept common truthy values: "true/false", "1/0", "yes/no", "on/off"
        if b, err := strconv.ParseBool(v); err == nil {
            cfg.ReverseProxyEnabled = b
        } else {
            return fmt.Errorf("invalid REVERSE_PROXY_ENABLED value %q, expected boolean (true/false/1/0)", v)
        }
    }

    if v := strings.TrimSpace(os.Getenv("DEVICE_ENDPOINT_HOST")); v != "" {
        cleaned := v
        // 1. Remove scheme if present (http:// or https://)
        if idx := strings.Index(cleaned, "://"); idx != -1 {
            cleaned = cleaned[idx+3:]
        }

        // 2. Remove path/query/fragment if present
        //    Keep only host[:port]
        if idx := strings.IndexAny(cleaned, "/?#"); idx != -1 {
            cleaned = cleaned[:idx]
        }

        // 3. Final trim
        cleaned = strings.TrimSpace(cleaned)
        cfg.DeviceEndpointHost = cleaned
    }

    if v := strings.TrimSpace(os.Getenv("WEB_UI_HOST")); v != "" {
        cleaned := v
        // 1. Remove scheme if present (http:// or https://)
        if idx := strings.Index(cleaned, "://"); idx != -1 {
            cleaned = cleaned[idx+3:]
        }

        // 2. Remove path/query/fragment if present
        if idx := strings.IndexAny(cleaned, "/?#"); idx != -1 {
            cleaned = cleaned[:idx]
        }

        // 3. Final trim
        cleaned = strings.TrimSpace(cleaned)
        cfg.WebUIHost = cleaned
    }

    return nil
}

func getDurationOpt(yamlCfg *yaml.File, name string, opt *time.Duration) error {
    val, err := yamlCfg.Get(name)
    if err != nil {
        return nil
    }
    d, err := time.ParseDuration(strings.TrimSpace(val))
    if err != nil {
        return fmt.Errorf("invalid %s value %q: %w", name, val, err)
    }
    *opt = d
    return nil
}

// splitScopes splits a whitespace/comma/newline-separated scope string
// into a deduplicated, cleaned string slice.
func splitScopes(s string) []string {
    // Replace commas with spaces to unify delimiters
    s = strings.ReplaceAll(s, ",", " ")
    // Remove optional YAML list characters such as brackets (lenient parsing)
    s = strings.NewReplacer("[", " ", "]", " ").Replace(s)

    parts := strings.Fields(s)
    uniq := make([]string, 0, len(parts))
    seen := make(map[string]struct{}, len(parts))
    for _, p := range parts {
        p = strings.TrimSpace(p)
        if p == "" || p == "-" { // Support accidental "-" items
            continue
        }
        if _, ok := seen[p]; ok {
            continue
        }
        seen[p] = struct{}{}
        uniq = append(uniq, p)
    }
    return uniq
}
