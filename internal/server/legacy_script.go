package server

import (
	"encoding/json"
	"errors"
	"net"
	"os"

	"rttys/internal/legacyscript"
	"rttys/xconfig"
)

// legacyScript renders the add device script for KVMs on firmware below
// V1.11.0, which connect to the token port. reqHost is the host the admin
// reached the web UI on, used when no device access address is configured.
func legacyScript(cfg *xconfig.Config, reqHost string) (legacyscript.Script, error) {
	if !cfg.LegacyTokenAccess {
		return legacyscript.Script{}, errors.New("legacy token access is off")
	}

	host := cfg.DeviceAccessHost
	if host == "" {
		host = cfg.WebrtcIP
	}
	if host == "" {
		host = reqHost
		if h, _, err := net.SplitHostPort(reqHost); err == nil {
			host = h
		}
	}
	_, port, err := net.SplitHostPort(cfg.AddrDev)
	if err != nil || port == "" || host == "" {
		return legacyscript.Script{}, errors.New("no device access address")
	}

	ca, err := os.ReadFile(cfg.DeviceCAFile)
	if err != nil {
		return legacyscript.Script{}, err
	}

	turn := ""
	if tc := deviceTurnConfig(cfg); tc != nil {
		b, err := json.MarshalIndent(tc, "", "    ")
		if err != nil {
			return legacyscript.Script{}, err
		}
		turn = string(b)
	}

	web := deviceWebURL(cfg)
	if web == "" && reqHost != "" {
		web = "https://" + reqHost
	}

	return legacyscript.Render(legacyscript.Params{
		Server:   net.JoinHostPort(host, port),
		Token:    cfg.Token,
		CA:       string(ca),
		TURN:     turn,
		WebUIURL: web,
		Version:  KVMCloudVersion,
	})
}
