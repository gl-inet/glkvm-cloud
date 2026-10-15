package server

import (
	"context"
	"errors"
	"net"
	"strings"

	"rttys/internal/store/sqlite"
	"rttys/xconfig"

	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
)

// ctrlEnvelope is an rtty-ctrl message, carried in msgTypeCtrl frames. Only
// rtty-go understands them, so they go to mTLS devices only.
type ctrlEnvelope struct {
	V    int      `json:"v"`
	T    string   `json:"t"` // "req", "resp" or "evt"
	ID   string   `json:"id,omitempty"`
	Cmd  string   `json:"cmd,omitempty"`
	Data any      `json:"data,omitempty"`
	OK   bool     `json:"ok,omitempty"`
	Err  *ctrlErr `json:"err,omitempty"`
}

type ctrlErr struct {
	Code string `json:"code"`
	Msg  string `json:"msg,omitempty"`
}

// cloudInfo is what the platform tells a device once it is registered, for the
// device's local web to show: its ID here, the address of the web UI and when
// it first joined.
type cloudInfo struct {
	Devid    string `json:"devid"`
	WebURL   string `json:"webUrl,omitempty"`
	JoinedAt int64  `json:"joinedAt,omitempty"`
}

func (dev *Device) sendCtrl(e ctrlEnvelope) {
	e.V = 1
	b, err := jsoniter.Marshal(e)
	if err != nil {
		return
	}
	if err := dev.WriteMsg(msgTypeCtrl, "", b); err != nil {
		log.Warn().Err(err).Str("devid", dev.id).Str("cmd", e.Cmd).Msg("send control message to device")
	}
}

func (dev *Device) sendCloudInfo(cfg *xconfig.Config) {
	dev.sendCtrl(ctrlEnvelope{T: "evt", Cmd: "cloud.info", Data: cloudInfo{
		Devid:    dev.id,
		WebURL:   deviceWebURL(cfg),
		JoinedAt: dev.joinedAt,
	}})
}

// turnConfig is the TURN server the device's web relays the KVM stream
// through, as the device keeps it in /tmp/turnserver.json.
type turnConfig struct {
	Username string   `json:"username"`
	TTL      int      `json:"ttl"`
	Password string   `json:"password"`
	URIs     []string `json:"uris"`
}

// deviceTurnConfig is the platform's TURN server as given to devices, nil
// when it has none.
func deviceTurnConfig(cfg *xconfig.Config) *turnConfig {
	host, port := strings.TrimSpace(cfg.WebrtcIP), strings.TrimSpace(cfg.WebrtcPort)
	if host == "" || port == "" || cfg.WebrtcUsername == "" || cfg.WebrtcPassword == "" {
		return nil
	}
	addr := net.JoinHostPort(host, port)
	return &turnConfig{
		Username: cfg.WebrtcUsername,
		TTL:      864000, // as the old self-hosted script wrote it
		Password: cfg.WebrtcPassword,
		URIs:     []string{"turn:" + addr + "?transport=udp", "turn:" + addr + "?transport=tcp"},
	}
}

// sendTurnConfig gives the device the platform's TURN server; before access
// codes, the old self-hosted script wrote it onto the device.
func (dev *Device) sendTurnConfig(cfg *xconfig.Config) {
	if tc := deviceTurnConfig(cfg); tc != nil {
		dev.sendCtrl(ctrlEnvelope{T: "evt", Cmd: "turn.config", Data: tc})
	}
}

// replyCtrl answers request id: success when errCode is "".
func (dev *Device) replyCtrl(id string, errCode, errMsg string) {
	e := ctrlEnvelope{T: "resp", ID: id, OK: errCode == ""}
	if errCode != "" {
		e.Err = &ctrlErr{Code: errCode, Msg: errMsg}
	}
	dev.sendCtrl(e)
}

// errDeviceLeft ends the connection of a device that has left the platform.
var errDeviceLeft = errors.New("device left the platform")

// handleCtrlMsg serves the requests a device sends on the control channel.
// Anything else is ignored: a control message never costs a device its link.
func handleCtrlMsg(dev *Device, data []byte) error {
	var e struct {
		T   string `json:"t"`
		ID  string `json:"id"`
		Cmd string `json:"cmd"`
	}
	if err := jsoniter.Unmarshal(data, &e); err != nil || e.T != "req" {
		return nil
	}
	switch e.Cmd {
	case "cloud.leave":
		return dev.leave(e.ID)
	default:
		dev.replyCtrl(e.ID, "unknown_cmd", "unknown command: "+e.Cmd)
		return nil
	}
}

// leave takes a device off the platform at its own request, before it moves to
// another platform or back to the official cloud: as when an admin deletes
// it, its whitelist entry and device record go and its connection is dropped.
// Only an mTLS device, which proves who it is, can do this.
func (dev *Device) leave(id string) error {
	if dev.certMAC == "" {
		dev.replyCtrl(id, "not_supported", "only devices on the mTLS port can leave")
		return nil
	}
	cont := sqlite.TryContainer()
	if cont == nil || cont.Access == nil {
		dev.replyCtrl(id, "internal", "")
		return nil
	}
	ctx := context.Background()
	if _, err := cont.Access.RemoveByMAC(ctx, dev.certMAC); err != nil {
		log.Error().Err(err).Str("mac", dev.certMAC).Msg("mtls: removing a leaving device failed")
		dev.replyCtrl(id, "internal", "")
		return nil
	}

	log.Info().Str("devid", dev.id).Str("mac", dev.certMAC).Msg("mtls: device left the platform")
	if cont.DeviceLogSvc != nil {
		cont.DeviceLogSvc.RecordDeviceLeft(ctx, dev.id, dev.certMAC, remoteIP(dev.conn))
	}
	dev.replyCtrl(id, "", "")
	return errDeviceLeft
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
