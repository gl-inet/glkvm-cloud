package server

import (
	"net"
	"strings"

	"rttys/xconfig"

	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
)

// ctrlEnvelope is an rtty-ctrl message, carried in msgTypeCtrl frames. Only
// rtty-go understands them, so they go to mTLS devices only.
type ctrlEnvelope struct {
	V    int    `json:"v"`
	T    string `json:"t"` // "req", "resp" or "evt"
	Cmd  string `json:"cmd,omitempty"`
	Data any    `json:"data,omitempty"`
}

// cloudInfo is what the platform tells a device once it is registered, for the
// device's local web to show: its ID here, the address of the web UI and when
// it first joined.
type cloudInfo struct {
	Devid    string `json:"devid"`
	WebURL   string `json:"webUrl,omitempty"`
	JoinedAt int64  `json:"joinedAt,omitempty"`
}

func (dev *Device) sendCloudInfo(cfg *xconfig.Config) {
	b, err := jsoniter.Marshal(ctrlEnvelope{V: 1, T: "evt", Cmd: "cloud.info", Data: cloudInfo{
		Devid:    dev.id,
		WebURL:   deviceWebURL(cfg),
		JoinedAt: dev.joinedAt,
	}})
	if err != nil {
		return
	}
	if err := dev.WriteMsg(msgTypeCtrl, "", b); err != nil {
		log.Warn().Err(err).Str("devid", dev.id).Msg("send cloud info to device")
	}
}

// deviceWebURL is the address of the web UI as given to devices. They have no
// request to take a host from, so it comes from the configuration:
// SELFHOST_WEBUI_URL, else the web port on the configured host.
func deviceWebURL(cfg *xconfig.Config) string {
	if u := strings.TrimSpace(cfg.SelfhostWebUIURL); u != "" {
		return u
	}
	host := cfg.WebUIHost
	if host == "" {
		host = cfg.DeviceAccessHost
	}
	if host == "" {
		host = cfg.WebrtcIP
	}
	if host == "" {
		return ""
	}
	if _, _, err := net.SplitHostPort(host); err == nil {
		return "https://" + host // already has a port
	}
	if _, port, err := net.SplitHostPort(cfg.AddrUser); err == nil && port != "" && port != "443" {
		return "https://" + net.JoinHostPort(host, port)
	}
	if strings.Contains(host, ":") {
		return "https://[" + host + "]"
	}
	return "https://" + host
}
