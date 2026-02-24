package server

import (
    "os"
    "runtime/debug"

    "github.com/rs/zerolog/log"
)

func LogPanic() {
    if r := recover(); r != nil {
        SaveCrashLog(r, debug.Stack())
        os.Exit(2)
    }
}

func SaveCrashLog(p any, stack []byte) {
    log.Error().Msgf("%v", p)
    log.Error().Msg(string(stack))
}
