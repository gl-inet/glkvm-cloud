package server

import (
    "encoding/json"
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
        AddrDev:   ":5912",
        AddrUser:  ":5913",
        LocalAuth: true,
        LogPath:   defaultLogPath,
        LogLevel:  "info",
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

    {
        importJSON, _ := json.MarshalIndent(cfg, "", "  ")
        log.Info().Msg("==== Loaded Configuration ====")
        log.Info().Msg(string(importJSON))
        log.Info().Msg("==============================")
    }

    xconfig.InitGlobal(&cfg)

    srv := New(cfg)
    return srv.Run()
}
