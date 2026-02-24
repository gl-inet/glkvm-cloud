package server

import (
    "context"
    "encoding/base64"
    "encoding/json"
    "fmt"
    oidc "github.com/coreos/go-oidc/v3/oidc"
    "github.com/gin-gonic/gin"
    "github.com/gorilla/sessions"
    "github.com/rs/zerolog/log"
    "io"
    "math/rand"
    "net/http"
    "net/url"
    "rttys/internal/domain/user"
    "rttys/internal/pkg/randtoken"
    "rttys/xconfig"
    "strings"
    "time"
)

var (
    // Session store for OAuth flow
    oauthStore *sessions.CookieStore
    // Global OIDC verifier
    oidcVerifier *oidc.IDTokenVerifier
)

// Register OIDC routes
func RegisterOIDCRoutes(r *gin.Engine, cfg *xconfig.Config, userSvc *user.Service) {
    if !cfg.OIDCEnabled {
        return
    }

    // ===== Initialize OIDC provider & ID token verifier =====
    issuer := strings.TrimSpace(cfg.OIDCGenericIssuer)
    if issuer == "" {
        log.Error().Msg("OIDC is enabled but issuer (OIDCGenericIssuer) is empty")
        return
    }

    ctx := context.Background()
    //// Normalize issuer to always end with a single '/'
    //issuer = strings.TrimRight(issuer, "/") + "/"
    cfg.OIDCGenericIssuer = issuer
    provider, err := oidc.NewProvider(ctx, issuer)
    if err != nil {
        log.Error().Err(err).Msg("Failed to initialize OIDC provider")
        return
    }

    oidcVerifier = provider.Verifier(&oidc.Config{
        ClientID: cfg.OIDCGenericClientID,
        // You can set SkipIssuerCheck, SkipClientIDCheck here if needed, but not recommended.
    })

    // Initialize session store
    sessionSecret := generateRandomString(32)
    oauthStore = sessions.NewCookieStore([]byte(sessionSecret))

    // Configure session options
    oauthStore.Options = &sessions.Options{
        Path:     "/",
        MaxAge:   300, // 5 minutes, enough to complete the OAuth flow
        HttpOnly: true,
        SameSite: http.SameSiteLaxMode, // Important: allow cookies on cross-site navigation (OIDC redirect)
        Domain:   "",                   // Empty means current host/domain will be used
    }

    // OIDC auth routes (public, no existing auth required)
    r.GET("/auth/oidc/login", oidcLoginHandler(cfg))
    r.GET("/auth/oidc/callback", oidcCallbackHandler(cfg, userSvc))
}

// Start OIDC login
func oidcLoginHandler(cfg *xconfig.Config) gin.HandlerFunc {
    return func(c *gin.Context) {

        // Generate state and nonce
        nonce := generateRandomString(32)
        state := generateRandomString(32)

        log.Info().Msgf("OIDC login initiated: nonce=%s, state=%s...", nonce[:10], state[:10])

        // Save to session
        session, _ := oauthStore.Get(c.Request, "oidc-session")
        session.Values["state"] = state
        session.Values["nonce"] = nonce

        if err := session.Save(c.Request, c.Writer); err != nil {
            log.Error().Err(err).Msg("Failed to save OIDC session")
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
            return
        }

        // Build authorization URL
        params := url.Values{}
        params.Add("client_id", cfg.OIDCGenericClientID)
        params.Add("redirect_uri", cfg.OIDCGenericRedirectURL)
        params.Add("response_type", "code")
        params.Add("scope", strings.Join(cfg.OIDCGenericScopes, " "))
        params.Add("state", state)
        params.Add("nonce", nonce)

        authURL := cfg.OIDCGenericAuthURL + "?" + params.Encode()
        log.Info().Msgf("OIDC login redirect=%s", authURL[:100]+"...")
        c.Redirect(http.StatusFound, authURL)
    }
}

// Handle OIDC callback
func oidcCallbackHandler(cfg *xconfig.Config, userSvc *user.Service) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Get session
        session, err := oauthStore.Get(c.Request, "oidc-session")
        if err != nil {
            log.Error().Err(err).Msg("Failed to get OIDC session")
            c.Redirect(http.StatusFound, "/?error=session_error")
            return
        }

        // Validate state
        state := c.Query("state")
        savedState, ok := session.Values["state"].(string)
        if !ok || state != savedState {
            log.Warn().Msg("OIDC state mismatch")
            c.Redirect(http.StatusFound, "/?error=invalid_state")
            return
        }

        // Check OAuth error
        if errorMsg := c.Query("error"); errorMsg != "" {
            errorDesc := c.Query("error_description")
            log.Warn().Msgf("OIDC error: %s - %s", errorMsg, errorDesc)
            c.Redirect(http.StatusFound, "/?error="+errorMsg)
            return
        }

        // Read authorization code
        code := c.Query("code")
        if code == "" {
            c.Redirect(http.StatusFound, "/?error=no_code")
            return
        }

        // Exchange authorization code for tokens
        tokens, err := exchangeCodeForTokens(cfg, code)
        if err != nil {
            log.Error().Err(err).Msg("Failed to exchange code for tokens")
            c.Redirect(http.StatusFound, "/?error=token_exchange_failed")
            return
        }

        // Extract user info
        var userEmail string
        var userName string

        // Standard OIDC verify and parse ID token
        rawIDToken, ok := tokens["id_token"].(string)
        if !ok {
            log.Error().Msg("No ID token in response")
            c.Redirect(http.StatusFound, "/?error=no_id_token")
            return
        }

        if oidcVerifier == nil {
            log.Error().Msg("OIDC verifier is not initialized")
            c.Redirect(http.StatusFound, "/?error=server_config")
            return
        }

        // ==== Signature + standard claims verification ====
        idToken, err := oidcVerifier.Verify(c.Request.Context(), rawIDToken)
        if err != nil {
            log.Error().Err(err).Msg("Failed to verify ID token signature/claims")
            c.Redirect(http.StatusFound, "/?error=invalid_token")
            return
        }

        // Decode claims into a map
        claims := map[string]interface{}{}
        if err := idToken.Claims(&claims); err != nil {
            log.Error().Err(err).Msg("Failed to parse ID token claims")
            c.Redirect(http.StatusFound, "/?error=invalid_token")
            return
        }

        // Pretty-print all claims as JSON for debugging
        if claimsJSON, err := json.MarshalIndent(claims, "", "  "); err == nil {
            log.Info().Msg("========== OIDC ID Token Claims ==========")
            log.Info().Msg(string(claimsJSON))
            log.Info().Msg("==========================================")
        } else {
            log.Warn().Err(err).Msg("Failed to marshal OIDC claims to JSON")
        }

        // Validate nonce
        if savedNonce, ok := session.Values["nonce"].(string); ok {
            if claims["nonce"] != savedNonce {
                log.Warn().Msg("OIDC nonce mismatch")
                c.Redirect(http.StatusFound, "/?error=invalid_nonce")
                return
            }
        }

        sub, _ := claims["sub"].(string)
        userEmail, _ = claims["email"].(string)
        userName, _ = claims["name"].(string)
        if userName == "" {
            // Fallback: use email or sub as display name
            if userEmail != "" {
                userName = userEmail
            } else {
                userName = sub
            }
        }

        // sub is required by OIDC spec and should never be empty
        if sub == "" {
            log.Error().Msg("OIDC token is missing 'sub' claim")
            c.Redirect(http.StatusFound, "/?error=user_info_failed")
            return
        }

        log.Info().Msgf("OIDC login successful:  email=%s", userEmail)

        // ====== OIDC whitelist enforcement ======
        if !isOIDCUserAllowed(cfg, claims) {
            log.Warn().Msgf("OIDC user not allowed by whitelist rules, sub=%v, email=%v", claims["sub"], claims["email"])
            c.Redirect(http.StatusFound, "/?error=authorization")
            return
        }

        // ==== Create application session (new session_store, same as LDAP) ====
        sid, err := randtoken.New() // randtoken.New()
        if err != nil {
            log.Error().Err(err).Msg("Failed to create session token")
            c.Redirect(http.StatusFound, "/?error=internal_error")
            return
        }

        sysAdmin, err := userSvc.GetSystemAdmin(c.Request.Context())
        if err != nil {
            log.Error().Err(err).Msg("Failed to find system admin user")
            c.Redirect(http.StatusFound, "/?error=internal_error")
            return
        }
        sessionStore.Create(sid, sysAdmin.ID)

        c.SetCookie("sid", sid, 0, "/", "", cfg.SslCert != "", false)

        // Clean up OAuth session
        session.Options.MaxAge = -1
        session.Save(c.Request, c.Writer)

        // Redirect to home page
        c.Redirect(http.StatusFound, "/")
    }
}

// Exchange authorization code for tokens
func exchangeCodeForTokens(cfg *xconfig.Config, code string) (map[string]interface{}, error) {
    data := url.Values{}
    data.Set("code", code)
    data.Set("client_id", cfg.OIDCGenericClientID)
    data.Set("client_secret", cfg.OIDCGenericClientSecret)
    data.Set("redirect_uri", cfg.OIDCGenericRedirectURL)
    data.Set("grant_type", "authorization_code")

    req, err := http.NewRequest("POST", cfg.OIDCGenericTokenURL, strings.NewReader(data.Encode()))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.Header.Set("Accept", "application/json")

    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("token request failed: %s", string(body))
    }

    var result map[string]interface{}
    if err := json.Unmarshal(body, &result); err != nil {
        return nil, err
    }

    return result, nil
}

// Generate a random string with given length
func generateRandomString(length int) string {
    b := make([]byte, length)
    rand.Read(b)
    return base64.URLEncoding.EncodeToString(b)[:length]
}

func isOIDCUserAllowed(cfg *xconfig.Config, claims map[string]interface{}) bool {
    email, _ := claims["email"].(string)
    sub, _ := claims["sub"].(string)
    preferredUsername, _ := claims["preferred_username"].(string)
    name, _ := claims["name"].(string)

    // 1) Email whitelist
    if len(cfg.OIDCGenericAllowedUsers) > 0 {
        if !isEmailAllowed(cfg.OIDCGenericAllowedUsers, email) {
            return false
        }
    }

    // 2) Sub (subject) whitelist
    if len(cfg.OIDCGenericAllowedSubs) > 0 {
        if !contains(cfg.OIDCGenericAllowedSubs, sub) {
            return false
        }
    }

    // 3) Username whitelist (preferred_username > name)
    if len(cfg.OIDCGenericAllowedUsernames) > 0 {
        u := preferredUsername
        if u == "" {
            u = name
        }
        if !contains(cfg.OIDCGenericAllowedUsernames, u) {
            return false
        }
    }

    // 4) Groups whitelist
    if len(cfg.OIDCGenericAllowedGroups) > 0 {
        groups := extractStringSlice(claims["groups"])
        if !intersects(groups, cfg.OIDCGenericAllowedGroups) {
            return false
        }
    }

    return true
}

// Email whitelist check rules:
// - Exact match (case-insensitive)
// - Simple domain match: entries starting with "@example.com" or "*@example.com"
//   mean "allow all users under this domain"
func isEmailAllowed(allowed []string, email string) bool {
    if len(allowed) == 0 {
        return true
    }
    e := strings.ToLower(strings.TrimSpace(email))
    for _, raw := range allowed {
        a := strings.ToLower(strings.TrimSpace(raw))
        if a == "" {
            continue
        }
        if strings.HasPrefix(a, "*@") || strings.HasPrefix(a, "@") {
            // Domain match
            dom := strings.TrimPrefix(a, "*@")
            dom = strings.TrimPrefix(dom, "@")
            if strings.HasSuffix(e, "@"+dom) {
                return true
            }
            continue
        }
        // Exact match
        if e == a {
            return true
        }
    }
    return false
}

// contains reports whether slice contains the given value v.
// Comparison is done after trimming spaces on both sides.
func contains(slice []string, v string) bool {
    v = strings.TrimSpace(v)
    if v == "" {
        return false
    }
    for _, s := range slice {
        if strings.TrimSpace(s) == v {
            return true
        }
    }
    return false
}

// extractStringSlice tries to normalize a generic claim value into a []string.
//
// It supports:
//   - []string
//   - []interface{} (only string elements are kept)
//   - string (split by comma and/or whitespace)
// Any other type will result in an empty slice.
func extractStringSlice(v interface{}) []string {
    if v == nil {
        return nil
    }

    switch vv := v.(type) {
    case []string:
        // Return a shallow copy to avoid accidental modification.
        out := make([]string, 0, len(vv))
        for _, s := range vv {
            s = strings.TrimSpace(s)
            if s != "" {
                out = append(out, s)
            }
        }
        return out

    case []interface{}:
        out := make([]string, 0, len(vv))
        for _, item := range vv {
            s, ok := item.(string)
            if !ok {
                continue
            }
            s = strings.TrimSpace(s)
            if s != "" {
                out = append(out, s)
            }
        }
        return out

    case string:
        // Allow comma- or whitespace-separated group lists.
        s := strings.TrimSpace(vv)
        if s == "" {
            return nil
        }
        // Replace commas with spaces, then split on whitespace.
        s = strings.ReplaceAll(s, ",", " ")
        parts := strings.Fields(s)
        out := make([]string, 0, len(parts))
        for _, p := range parts {
            p = strings.TrimSpace(p)
            if p != "" {
                out = append(out, p)
            }
        }
        return out

    default:
        return nil
    }
}

// intersects reports whether slice a and b share at least one common element.
// Matching is done after trimming spaces on both sides.
func intersects(a, b []string) bool {
    if len(a) == 0 || len(b) == 0 {
        return false
    }

    m := make(map[string]struct{}, len(a))
    for _, s := range a {
        s = strings.TrimSpace(s)
        if s == "" {
            continue
        }
        m[s] = struct{}{}
    }

    for _, s := range b {
        s = strings.TrimSpace(s)
        if s == "" {
            continue
        }
        if _, ok := m[s]; ok {
            return true
        }
    }

    return false
}
