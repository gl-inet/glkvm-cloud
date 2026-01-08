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

package main

import (
    "fmt"
    "os"
    "strconv"
    "strings"

    "github.com/kylelemons/go-gypsy/yaml"
    "github.com/urfave/cli/v3"
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
    SslCert              string
    SslKey               string
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

func (cfg *Config) Parse(c *cli.Command) error {
    conf := c.String("conf")
    if conf != "" {
        err := parseYamlCfg(cfg, conf)
        if err != nil {
            return err
        }
    }

    getFlagOpt(c, "addr-dev", &cfg.AddrDev)
    getFlagOpt(c, "addr-user", &cfg.AddrUser)
    getFlagOpt(c, "addr-http-proxy", &cfg.AddrHttpProxy)
    getFlagOpt(c, "http-proxy-redir-url", &cfg.HttpProxyRedirURL)
    getFlagOpt(c, "http-proxy-redir-domain", &cfg.HttpProxyRedirDomain)
    getFlagOpt(c, "dev-hook-url", &cfg.DevHookUrl)
    getFlagOpt(c, "user-hook-url", &cfg.UserHookUrl)
    getFlagOpt(c, "local-auth", &cfg.LocalAuth)
    getFlagOpt(c, "token", &cfg.Token)
    getFlagOpt(c, "password", &cfg.Password)
    getFlagOpt(c, "allow-origins", &cfg.AllowOrigins)
    getFlagOpt(c, "pprof", &cfg.PprofAddr)

    cfg.SslCert = SslCert
    cfg.SslKey = SslKey

    getFlagOpt(c, "webrtc-ip", &cfg.WebrtcIP)
    getFlagOpt(c, "webrtc-port", &cfg.WebrtcPort)
    getFlagOpt(c, "webrtc-username", &cfg.WebrtcUsername)
    getFlagOpt(c, "webrtc-password", &cfg.WebrtcPassword)

    // LDAP configuration flags
    getFlagOpt(c, "ldap-enabled", &cfg.LdapEnabled)
    getFlagOpt(c, "ldap-server", &cfg.LdapServer)
    getFlagOpt(c, "ldap-port", &cfg.LdapPort)
    getFlagOpt(c, "ldap-use-tls", &cfg.LdapUseTLS)
    getFlagOpt(c, "ldap-bind-dn", &cfg.LdapBindDN)
    getFlagOpt(c, "ldap-bind-password", &cfg.LdapBindPassword)
    getFlagOpt(c, "ldap-base-dn", &cfg.LdapBaseDN)
    getFlagOpt(c, "ldap-user-filter", &cfg.LdapUserFilter)
    getFlagOpt(c, "ldap-allowed-groups", &cfg.LdapAllowedGroups)
    getFlagOpt(c, "ldap-allowed-users", &cfg.LdapAllowedUsers)

    // Generic OIDC Provider (standard OIDC provider flags)
    getFlagOpt(c, "oidc-enabled", &cfg.OIDCEnabled)
    getFlagOpt(c, "oidc-generic-issuer", &cfg.OIDCGenericIssuer)
    getFlagOpt(c, "oidc-generic-client-id", &cfg.OIDCGenericClientID)
    getFlagOpt(c, "oidc-generic-client-secret", &cfg.OIDCGenericClientSecret)
    getFlagOpt(c, "oidc-generic-auth-url", &cfg.OIDCGenericAuthURL)
    getFlagOpt(c, "oidc-generic-token-url", &cfg.OIDCGenericTokenURL)
    getFlagOpt(c, "oidc-generic-redirect-url", &cfg.OIDCGenericRedirectURL)
    getFlagOpt(c, "oidc-generic-scopes", &cfg.OIDCGenericScopes)
    getFlagOpt(c, "oidc-generic-allowed-users", &cfg.OIDCGenericAllowedUsers)
    getFlagOpt(c, "oidc-generic-allowed-subs", &cfg.OIDCGenericAllowedSubs)
    getFlagOpt(c, "oidc-generic-allowed-usernames", &cfg.OIDCGenericAllowedUsernames)
    getFlagOpt(c, "oidc-generic-allowed-groups", &cfg.OIDCGenericAllowedGroups)

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

    // LDAP password is always read from environment variable to avoid YAML special character parsing issues and for security.
    if envPassword := os.Getenv("LDAP_BIND_PASSWORD"); envPassword != "" {
        cfg.LdapBindPassword = envPassword
    }

    // ===== OIDC configuration (generic OIDC provider) =====
    // Switch and basic endpoints
    getConfigOpt(yamlCfg, "oidc-enabled", &cfg.OIDCEnabled)
    getConfigOpt(yamlCfg, "oidc-generic-issuer", &cfg.OIDCGenericIssuer)
    getConfigOpt(yamlCfg, "oidc-generic-client-id", &cfg.OIDCGenericClientID)

    // Note: oidc-generic-client-secret is intentionally not read from YAML
    // to avoid checking secrets into config files and leaking in logs.
    // It is always read directly from the OIDC_CLIENT_SECRET environment variable below.
    if envSecret := os.Getenv("OIDC_CLIENT_SECRET"); envSecret != "" {
        cfg.OIDCGenericClientSecret = envSecret
    }

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

func getFlagOpt(c *cli.Command, name string, opt any) {
    if !c.IsSet(name) {
        return
    }

    switch opt := opt.(type) {
    case *string:
        *opt = c.String(name)
    case *int:
        *opt = c.Int(name)
    case *bool:
        *opt = c.Bool(name)
    }
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
