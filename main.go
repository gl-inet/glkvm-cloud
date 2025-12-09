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
    "context"
    "encoding/json"
    _ "net/http/pprof"
    "os"
    "rttys/db"
    "runtime"
    "runtime/debug"

    xlog "rttys/log"

    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
    "github.com/urfave/cli/v3"
)

const RttysVersion = "5.2.0"

var (
    GitCommit = ""
    BuildTime = ""
)

func main() {
    defaultLogPath := "/var/log/rttys.log"
    if runtime.GOOS == "windows" {
        defaultLogPath = "rttys.log"
    }

    cmd := &cli.Command{
        Name:    "rttys",
        Usage:   "The server side for rtty",
        Version: RttysVersion,
        Flags: []cli.Flag{
            &cli.StringFlag{
                Name:  "log",
                Value: defaultLogPath,
                Usage: "log file path",
            },
            &cli.StringFlag{
                Name:  "log-level",
                Value: "info",
                Usage: "log level(debug, info, warn, error)",
            },
            &cli.StringFlag{
                Name:    "conf",
                Aliases: []string{"c"},
                Usage:   "config file to load",
            },
            &cli.StringFlag{
                Name:  "addr-dev",
                Value: ":5912",
                Usage: "address to listen device",
            },
            &cli.StringFlag{
                Name:  "addr-user",
                Value: ":5913",
                Usage: "address to listen user",
            },
            &cli.StringFlag{
                Name:  "addr-http-proxy",
                Usage: "address to listen for HTTP proxy (default auto)",
            },
            &cli.StringFlag{
                Name:  "http-proxy-redir-url",
                Usage: "url to redirect for HTTP proxy",
            },
            &cli.StringFlag{
                Name:  "http-proxy-redir-domain",
                Usage: "domain for HTTP proxy set cookie",
            },
            &cli.StringFlag{
                Name:    "token",
                Aliases: []string{"t"},
                Usage:   "token to use",
            },
            &cli.StringFlag{
                Name:  "dev-hook-url",
                Usage: "called when the device is connected",
            },
            &cli.StringFlag{
                Name:  "user-hook-url",
                Usage: "called when user accesses /connect/:devid, /cmd/:devid, /web/, or /web2/ APIs",
            },
            &cli.BoolFlag{
                Name:  "local-auth",
                Value: true,
                Usage: "need auth for local",
            },
            &cli.StringFlag{
                Name:  "password",
                Usage: "web management password",
            },
            &cli.BoolFlag{
                Name:  "allow-origins",
                Usage: "allow all origins for cross-domain request",
            },
            &cli.BoolFlag{
                Name:  "ldap-enabled",
                Usage: "enable LDAP authentication",
            },
            &cli.StringFlag{
                Name:  "ldap-server",
                Usage: "LDAP server hostname or IP",
            },
            &cli.IntFlag{
                Name:  "ldap-port",
                Value: 389,
                Usage: "LDAP server port",
            },
            &cli.BoolFlag{
                Name:  "ldap-use-tls",
                Usage: "use TLS/SSL for LDAP connection",
            },
            &cli.StringFlag{
                Name:  "ldap-bind-dn",
                Usage: "LDAP bind DN for service account",
            },
            &cli.StringFlag{
                Name:  "ldap-bind-password",
                Usage: "LDAP bind password for service account",
            },
            &cli.StringFlag{
                Name:  "ldap-base-dn",
                Usage: "LDAP base DN for user searches",
            },
            &cli.StringFlag{
                Name:  "ldap-user-filter",
                Value: "(uid=%s)",
                Usage: "LDAP user filter",
            },
            &cli.StringFlag{
                Name:  "ldap-allowed-groups",
                Usage: "comma-separated list of allowed LDAP groups",
            },
            &cli.StringFlag{
                Name:  "ldap-allowed-users",
                Usage: "comma-separated list of allowed LDAP users",
            },
            &cli.StringFlag{
                Name:  "pprof",
                Usage: "enable pprof and listen on specified address (e.g. localhost:6060)",
            },

            // ---- OIDC Authentication (generic OIDC provider) ----
            &cli.BoolFlag{
                Name:  "oidc-enabled",
                Usage: "enable OIDC authentication (OpenID Connect)",
            },
            &cli.StringFlag{
                Name:  "oidc-generic-client-id",
                Usage: "OIDC client ID (issued by the identity provider)",
            },
            &cli.StringFlag{
                Name:  "oidc-generic-client-secret",
                Usage: "OIDC client secret (read from OIDC_GENERIC_CLIENT_SECRET env by default)",
            },
            &cli.StringFlag{
                Name:  "oidc-generic-auth-url",
                Usage: "OIDC authorization endpoint URL",
            },
            &cli.StringFlag{
                Name:  "oidc-generic-token-url",
                Usage: "OIDC token endpoint URL",
            },
            &cli.StringFlag{
                Name:  "oidc-generic-redirect-url",
                Usage: "OIDC redirect/callback URL (must match one registered in IdP)",
            },
            &cli.StringFlag{
                Name:  "oidc-generic-scopes",
                Value: "openid profile email",
                Usage: "space-separated list of OIDC scopes",
            },
            &cli.StringFlag{
                Name:  "oidc-generic-allowed-users",
                Usage: "optional email whitelist for OIDC logins (exact emails or @domain, space/comma-separated)",
            },
            &cli.StringFlag{
                Name:  "oidc-generic-allowed-subs",
                Usage: "optional subject (sub) whitelist for OIDC logins (space/comma-separated)",
            },
            &cli.StringFlag{
                Name:  "oidc-generic-allowed-usernames",
                Usage: "optional username whitelist for OIDC logins (preferred_username/name, space/comma-separated)",
            },
            &cli.StringFlag{
                Name:  "oidc-generic-allowed-groups",
                Usage: "optional groups whitelist for OIDC logins (space/comma-separated)",
            },

            &cli.BoolFlag{
                Name:    "verbose",
                Aliases: []string{"V"},
                Usage:   "more detailed output",
            },
        },
        Action: cmdAction,
    }

    err := cmd.Run(context.Background(), os.Args)
    if err != nil {
        log.Fatal().Msg(err.Error())
    }
}

func cmdAction(c context.Context, cmd *cli.Command) error {
    defer logPanic()

    xlog.SetPath(cmd.String("log"))

    switch cmd.String("log-level") {
    case "debug":
        zerolog.SetGlobalLevel(zerolog.DebugLevel)
    case "warn":
        zerolog.SetGlobalLevel(zerolog.WarnLevel)
    case "error":
        zerolog.SetGlobalLevel(zerolog.ErrorLevel)
    default:
        zerolog.SetGlobalLevel(zerolog.InfoLevel)
    }

    if cmd.Bool("verbose") {
        xlog.Verbose()
    }

    log.Info().Msg("Go Version: " + runtime.Version())
    log.Info().Msgf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)

    log.Info().Msg("Rttys Version: " + RttysVersion)

    if GitCommit != "" {
        log.Info().Msg("Git Commit: " + GitCommit)
    }

    if BuildTime != "" {
        log.Info().Msg("Build Time: " + BuildTime)
    }

    if runtime.GOOS != "windows" {
        go signalHandle()
    }

    cfg := Config{
        AddrDev:   ":5912",
        AddrUser:  ":5913",
        LocalAuth: true,
    }

    err := cfg.Parse(cmd)
    if err != nil {
        return err
    }

    // ===== 打印完整配置（验证配置是否加载正确） =====
    {
        importJSON, _ := json.MarshalIndent(cfg, "", "  ")
        log.Info().Msg("==== Loaded Configuration ====")
        log.Info().Msg(string(importJSON))
        log.Info().Msg("==============================")
    }

    // Initialize the SQLite database connection
    db.Init()

    srv := &RttyServer{cfg: cfg}

    return srv.Run()
}

func logPanic() {
    if r := recover(); r != nil {
        saveCrashLog(r, debug.Stack())
        os.Exit(2)
    }
}

func saveCrashLog(p any, stack []byte) {
    log.Error().Msgf("%v", p)
    log.Error().Msg(string(stack))
}
