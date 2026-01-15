package main

import (
    "context"
    "os/signal"
    "syscall"

    "rttys/internal/app"
    "rttys/internal/config"
)

func main() {
    cfg := config.MustLoad()
    a := app.Bootstrap(cfg)

    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    if err := a.Start(ctx); err != nil {
        panic(err)
    }
}
