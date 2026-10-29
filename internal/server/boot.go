package server

import (
    "encoding/json"
    "fmt"
    "os"
    "runtime"

    xlog "rttys/log"
    "rttys/xconfig"

    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
)

func RunFromEnv() error {
    defaultLogPath := "/var/log/rttys.log"
    if runtime.GOOS == "windows" {
        defaultLogPath = "rttys.log"
    }

    defer LogPanic()

    cfg := xconfig.Config{
        AddrDev:     ":5912",
        AddrDevMTLS: ":5915",
        AddrUser:    ":5913",
        LocalAuth:   true,
        LogPath:     defaultLogPath,
        LogLevel:    "info",

        LegacyTokenAccess: true,
    }

    confPath := os.Getenv("RTTYS_CONF")
    if confPath == "" {
        if _, err := os.Stat("rttys.conf"); err == nil {
            confPath = "rttys.conf"
        }
    }

    if err := cfg.Load(confPath); err != nil {
        return err
    }

    xlog.SetPath(cfg.LogPath)

    // Logging is set up first: the file hook has no path before SetPath, so an
    // earlier Fatal would be reported as a failed log write instead of this.
    if err := validateConfig(&cfg); err != nil {
        return err
    }

    switch cfg.LogLevel {
    case "debug":
        zerolog.SetGlobalLevel(zerolog.DebugLevel)
    case "warn":
        zerolog.SetGlobalLevel(zerolog.WarnLevel)
    case "error":
        zerolog.SetGlobalLevel(zerolog.ErrorLevel)
    default:
        zerolog.SetGlobalLevel(zerolog.InfoLevel)
    }

    if cfg.Verbose {
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
        go StartSignalHandler()
    }

    if err := setupPKI(&cfg, xconfig.PKIDir); err != nil {
        return err
    }

    for _, s := range publicSecrets(&cfg) {
        log.Warn().Msgf("%s is the example value published in .env.example: %s", s.setting, s.advice)
    }
    if cfg.OutdatedBundle {
        log.Warn().Msg("the deployment files predate v3.0: their docker-compose.yml does not publish the device port for access codes, so devices cannot join; update docker-compose.yml, templates and scripts, keeping .env and database")
    }
    if cfg.WebrtcUsername == "" || cfg.WebrtcPassword == "" {
        log.Warn().Msg("TURN_USER or TURN_PASS is not set: devices get no TURN server, so remote control needs a direct connection")
    }

    {
        importJSON, _ := json.MarshalIndent(loggableConfig(cfg), "", "  ")
        log.Info().Msg("==== Loaded Configuration ====")
        log.Info().Msg(string(importJSON))
        log.Info().Msg("==============================")
    }

    xconfig.InitGlobal(&cfg)

    srv := New(cfg)
    return srv.Run()
}

// validateConfig refuses settings that would leave the platform unusable or
// open: no password (nobody could sign in), and legacy token access with no
// token (any device could register).
func validateConfig(cfg *xconfig.Config) error {
    if cfg.Password == "" {
        return fmt.Errorf("password is not set: set 'password' in the config file or RTTYS_PASS")
    }
    if cfg.LegacyTokenAccess && cfg.Token == "" {
        return fmt.Errorf("token is not set: set 'token' (RTTYS_TOKEN), or turn off legacy token access with 'legacy-token-access: false' (RTTYS_LEGACY_TOKEN_ACCESS)")
    }
    return nil
}

// publicSecret is a setting that still holds a value anyone can look up.
type publicSecret struct{ setting, advice string }

// publicSecrets lists the settings left at the example values the .env
// examples used to ship with. They are only warned about: devices on old
// firmware were given the token and TURN account, and changing those cuts
// them off.
func publicSecrets(cfg *xconfig.Config) []publicSecret {
    var found []publicSecret
    if cfg.Password == "StrongP@ssw0rd" {
        found = append(found, publicSecret{"RTTYS_PASS", "change it"})
    }
    if cfg.LegacyTokenAccess && cfg.Token == "DeviceTokenYouCanChangeMe" {
        found = append(found, publicSecret{"RTTYS_TOKEN", "devices on old firmware use it, so turn off legacy token access once they all join with access codes"})
    }
    if cfg.WebrtcPassword == "AnotherS3cret" {
        found = append(found, publicSecret{"TURN_PASS", "change it for both services; devices added with the old script keep the old one until they join with an access code"})
    }
    return found
}

// loggableConfig is cfg with its secrets masked for the startup log. An unset
// secret stays empty, so the log still shows it is missing.
func loggableConfig(cfg xconfig.Config) xconfig.Config {
    for _, s := range []*string{&cfg.Token, &cfg.Password, &cfg.WebrtcPassword, &cfg.LdapBindPassword, &cfg.OIDCGenericClientSecret} {
        if *s != "" {
            *s = "******"
        }
    }
    return cfg
}
