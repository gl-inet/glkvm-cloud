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
    "encoding/json"
    _ "net/http/pprof"
    "os"
    "rttys/xconfig"
    "runtime"
    "runtime/debug"

    xlog "rttys/log"

    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
)

const RttysVersion = "5.2.0"
const KVMCloudVersion = "v2.0.0"

var (
    GitCommit = ""
    BuildTime = ""
)

func main() {
    defaultLogPath := "/var/log/rttys.log"
    if runtime.GOOS == "windows" {
        defaultLogPath = "rttys.log"
    }

    defer logPanic()

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
        log.Fatal().Msg(err.Error())
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
        go signalHandle()
    }

    // ===== 打印完整配置 =====
    {
        importJSON, _ := json.MarshalIndent(cfg, "", "  ")
        log.Info().Msg("==== Loaded Configuration ====")
        log.Info().Msg(string(importJSON))
        log.Info().Msg("==============================")
    }
    xconfig.InitGlobal(&cfg)

    srv := &RttyServer{cfg: cfg}

    if err := srv.Run(); err != nil {
        log.Fatal().Msg(err.Error())
    }
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
