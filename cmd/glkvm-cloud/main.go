package main

import (
    _ "net/http/pprof"
    "rttys/internal/server"

    "github.com/rs/zerolog/log"
)

func main() {
    if err := server.RunFromEnv(); err != nil {
        log.Fatal().Msg(err.Error())
    }
}
