//go:build !windows
// +build !windows

package server

import (
	"os"
	"os/signal"
	"syscall"

	xlog "rttys/log"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func signalHandle() {
	c := make(chan os.Signal, 1)

	signal.Notify(c, syscall.SIGUSR1)

	for s := range c {
		switch s {
		case syscall.SIGUSR1:
			xlog.Verbose()
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
			log.Debug().Msg("Debug mode enabled")
		}
	}
}
