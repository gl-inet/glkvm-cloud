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
    "bufio"
    "bytes"
    "context"
    "crypto/tls"
    "encoding/binary"
    "fmt"
    "io"
    "net"
    "net/http"
    "strings"
    "sync"
    "time"

    "rttys/utils"

    "github.com/gorilla/websocket"
    jsoniter "github.com/json-iterator/go"
    "github.com/rs/zerolog/log"
    "github.com/valyala/bytebufferpool"
)

type DeviceInfo struct {
    ID        string `json:"id"`
    Mac       string `json:"mac"`
    Connected uint32 `json:"connected"`
    Uptime    uint32 `json:"uptime"`
    Desc      string `json:"description"`
    Proto     uint8  `json:"proto"`
    IPaddr    string `json:"ipaddr"`
}

type Device struct {
    group     string
    id        string
    proto     uint8
    desc      string
    timestamp int64
    uptime    uint32
    token     string
    heartbeat time.Duration

    users    sync.Map
    pending  sync.Map
    commands sync.Map
    https    sync.Map

    conn    net.Conn
    br      *bufio.Reader
    readBuf []byte
    close   sync.Once
    ctx     context.Context
    cancel  context.CancelFunc
}

const (
    msgTypeRegister = byte(iota)
    msgTypeLogin
    msgTypeLogout
    msgTypeTermData
    msgTypeWinsize
    msgTypeCmd
    msgTypeHeartbeat
    msgTypeFile
    msgTypeHttp
    msgTypeAck
)

const (
    msgTypeFileSend = byte(iota)
    msgTypeFileRecv
    msgTypeFileInfo
    msgTypeFileData
    msgTypeFileAck
    msgTypeFileAbort
)

const (
    msgRegAttrHeartbeat = iota
    msgRegAttrDevid
    msgRegAttrDescription
    msgRegAttrToken
    msgRegAttrGroup
)

const (
    msgHeartbeatAttrUptime = iota
)

const (
    devRegErrUnsupportedProto = iota + 1
    devRegErrInvalidToken
    devRegErrHookFailed
    devRegErrIdConflicting
)

const (
    RttyProtoRequired uint8 = 3
    WaitRegistTimeout       = 5 * time.Second
    DefaultHeartbeat        = 5 * time.Second
    TermLoginTimeout        = 5 * time.Second
    CommandTimeout          = 30
)

var DevRegErrMsg = map[byte]string{
    0:                         "Success",
    devRegErrUnsupportedProto: "Unsupported protocol",
    devRegErrInvalidToken:     "Invalid token",
    devRegErrHookFailed:       "Hook failed",
    devRegErrIdConflicting:    "ID conflict",
}

var DeviceMsgHandlers = map[byte]func(*Device, []byte) error{
    msgTypeHeartbeat: handleHeartbeatMsg,
    msgTypeLogin:     handleLoginMsg,
    msgTypeLogout:    handleLogoutMsg,
    msgTypeTermData:  handleTermDataMsg,
    msgTypeFile:      handleFileMsg,
    msgTypeCmd:       handleCmdMsg,
    msgTypeHttp:      handleHttpMsg,
}

func (srv *RttyServer) ListenDevices() {
    cfg := &srv.cfg

    ln, err := net.Listen("tcp", cfg.AddrDev)
    if err != nil {
        log.Fatal().Msg(err.Error())
    }
    defer ln.Close()
    if cfg.SslCert != "" && cfg.SslKey != "" {
        crt, err := tls.LoadX509KeyPair(cfg.SslCert, cfg.SslKey)
        if err != nil {
            log.Fatal().Msg(err.Error())
        }

        tlsConfig := &tls.Config{
            GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
                // 忽略 SNI，始终返回唯一证书
                return &crt, nil
            },
        }

        ln = tls.NewListener(ln, tlsConfig)
    }

    log.Info().Msgf("Listen devices on: %s", ln.Addr().(*net.TCPAddr))

    for {
        conn, err := ln.Accept()
        if err != nil {
            log.Error().Msg(err.Error())
            continue
        }

        go handleDeviceConnection(srv, conn)
    }
}

func handleDeviceConnection(srv *RttyServer, conn net.Conn) {
    defer logPanic()

    dev := &Device{
        conn:      conn,
        heartbeat: DefaultHeartbeat,
        timestamp: time.Now().Unix(),
        br:        bufio.NewReader(conn),
    }
    defer dev.Close(srv)

    dev.ctx, dev.cancel = context.WithCancel(context.Background())

    log.Debug().Msgf("new device '%s' connected", conn.RemoteAddr())

    conn.SetReadDeadline(time.Now().Add(WaitRegistTimeout))

    typ, data, err := dev.ReadMsg()
    if err != nil {
        log.Error().Msgf("read register msg fail: %v", err)
        return
    }

    if typ != msgTypeRegister {
        log.Error().Msg("register msg expected first")
        return
    }

    if !dev.ParseRegister(data) {
        log.Error().Msg("invalid device info")
        return
    }

    code := dev.Register(srv)

    err = dev.WriteMsg(msgTypeRegister, "", append([]byte{code}, DevRegErrMsg[code]...))
    if err != nil {
        log.Printf("send register to device '%s' fail: %v", dev.id, err)
        return
    }

    if code != 0 {
        return
    }

    deviceRemoteIP := ""
    if addr, ok := dev.conn.RemoteAddr().(*net.TCPAddr); ok {
        deviceRemoteIP = addr.IP.String()
    } else if host, _, err := net.SplitHostPort(dev.conn.RemoteAddr().String()); err == nil {
        deviceRemoteIP = host
    }
    log.Info().Msgf("device '%s' registered, group '%s' proto %d, heartbeat %v, remoteIP '%s'",
        dev.id, dev.group, dev.proto, dev.heartbeat, deviceRemoteIP)

    // 2. Load existing metadata by device_id
    description := ""
    meta, err := GetDeviceMetaByDeviceID(dev.id)
    if err == nil && meta != nil {
        description = meta.Description
    }
    if err := SaveOrUpdateDeviceMeta(
        dev.id,
        dev.desc, // device register mac info with desc filed
        description,
        deviceRemoteIP,
    ); err != nil {
        return
    }

    for {
        conn.SetReadDeadline(time.Now().Add(dev.heartbeat * 3 / 2))

        typ, data, err = dev.ReadMsg()
        if err != nil {
            if err != io.EOF {
                log.Error().Msgf("read msg from device '%s' fail: %v", dev.id, err)
            }
            return
        }

        log.Debug().Msgf("device msg %s from device %s", msgTypeName(typ), dev.id)

        handler, ok := DeviceMsgHandlers[typ]
        if !ok {
            log.Error().Msgf("unexpected message '%s' from device '%s'", msgTypeName(typ), dev.id)
            return
        }

        err = handler(dev, data)
        if err != nil {
            log.Error().Msg(err.Error())
            return
        }
    }
}

func msgTypeName(typ byte) string {
    switch typ {
    case msgTypeRegister:
        return "register"
    case msgTypeLogin:
        return "login"
    case msgTypeLogout:
        return "logout"
    case msgTypeTermData:
        return "termdata"
    case msgTypeWinsize:
        return "winsize"
    case msgTypeCmd:
        return "cmd"
    case msgTypeHeartbeat:
        return "heartbeat"
    case msgTypeFile:
        return "file"
    case msgTypeHttp:
        return "http"
    case msgTypeAck:
        return "ack"
    default:
        return fmt.Sprintf("unknown(%d)", typ)
    }
}

func (dev *Device) ReadMsg() (byte, []byte, error) {
    head := make([]byte, 3)
    br := dev.br

    _, err := io.ReadFull(br, head)
    if err != nil {
        return 0, nil, err
    }

    typ := head[0]

    msgLen := binary.BigEndian.Uint16(head[1:])

    if cap(dev.readBuf) < int(msgLen) {
        dev.readBuf = make([]byte, msgLen)
    } else {
        dev.readBuf = dev.readBuf[:msgLen]
    }

    _, err = io.ReadFull(br, dev.readBuf)
    if err != nil {
        return 0, nil, err
    }

    return typ, dev.readBuf, nil
}

func (dev *Device) WriteMsg(typ byte, sid string, data []byte) error {
    bb := bytebufferpool.Get()
    defer bytebufferpool.Put(bb)

    b := []byte{typ, 0, 0}

    binary.BigEndian.PutUint16(b[1:], uint16(len(sid)+len(data)))

    bb.Write(b)
    bb.WriteString(sid)
    bb.Write(data)

    _, err := bb.WriteTo(dev.conn)

    return err
}

func (dev *Device) WriteFileMsg(typ byte, sid string, fileType byte, data []byte) error {
    bb := bytebufferpool.Get()
    defer bytebufferpool.Put(bb)

    bb.WriteByte(fileType)
    bb.Write(data)

    return dev.WriteMsg(typ, sid, bb.Bytes())
}

func (dev *Device) Close(srv *RttyServer) {
    dev.close.Do(func() {
        log.Error().Msgf("device '%s' disconnected", dev.id)
        srv.DelDevice(dev)
        dev.cancel()
        dev.conn.Close()
    })
}

func (dev *Device) ParseRegister(b []byte) bool {
    if len(b) < 1 {
        return false
    }

    dev.proto = b[0]

    if dev.proto > 4 {
        attrs := utils.ParseTLV(b[1:])
        if attrs == nil {
            return false
        }

        for typ, val := range attrs {
            switch typ {
            case msgRegAttrHeartbeat:
                dev.heartbeat = time.Duration(val[0]) * time.Second
            case msgRegAttrDevid:
                dev.id = string(val)
            case msgRegAttrDescription:
                dev.desc = string(val)
            case msgRegAttrToken:
                dev.token = string(val)
            case msgRegAttrGroup:
                dev.group = string(val)
            }
        }

        return true
    }

    b = b[1:]

    fields := bytes.Split(b, []byte{0})

    if len(fields) < 3 {
        return false
    }

    dev.id = string(fields[0])
    dev.desc = string(fields[1])
    dev.token = string(fields[2])

    return true
}

func (dev *Device) Register(srv *RttyServer) byte {
    cfg := &srv.cfg

    if dev.proto < RttyProtoRequired {
        log.Error().Msgf("minimum proto required %d, found %d for device '%s'", RttyProtoRequired, dev.proto, dev.id)
        return devRegErrHookFailed
    }

    log.Info().Msgf("cfg.Token:%s，dev.token:%s", cfg.Token, dev.token)
    if cfg.Token != "" && dev.token != cfg.Token {
        log.Error().Msgf("invalid token for device '%s'", dev.id)
        return devRegErrInvalidToken
    }

    devHookUrl := cfg.DevHookUrl
    if devHookUrl != "" {
        cli := &http.Client{
            Timeout: 3 * time.Second,
        }

        data := fmt.Sprintf(`{"group":"%s", "devid":"%s", "token":"%s"}`, dev.group, dev.id, dev.token)

        resp, err := cli.Post(devHookUrl, "application/json", strings.NewReader(data))
        if err != nil {
            log.Error().Msgf("call device hook url fail for device %s: %v", dev.id, err)
            return devRegErrHookFailed
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
            log.Error().Msgf("call device hook url for device '%s', StatusCode: %d", dev.id, resp.StatusCode)
            return devRegErrHookFailed
        }
    }

    if !srv.AddDevice(dev) {
        return devRegErrIdConflicting
    }

    return 0
}

func handleHeartbeatMsg(dev *Device, data []byte) error {
    if !parseHeartbeat(dev, data) {
        return fmt.Errorf("invalid heartbeat msg from device '%s'", dev.id)
    }
    return dev.WriteMsg(msgTypeHeartbeat, "", nil)
}

func parseHeartbeat(dev *Device, data []byte) bool {
    if dev.proto > 4 {
        attrs := utils.ParseTLV(data)
        if attrs == nil {
            return false
        }

        for typ, val := range attrs {
            switch typ {
            case msgHeartbeatAttrUptime:
                dev.uptime = binary.BigEndian.Uint32(val)
            }
        }
    } else {
        if len(data) < 4 {
            return false
        }
        dev.uptime = binary.BigEndian.Uint32(data[:4])
    }

    return true
}

func handleLogoutMsg(dev *Device, data []byte) error {
    if len(data) < 32 {
        return fmt.Errorf("invalid logout msg from device '%s'", dev.id)
    }

    sid := string(data[:32])

    if val, loaded := dev.users.LoadAndDelete(sid); loaded {
        user := val.(*User)
        user.Close()
    }

    return nil
}

func handleLoginMsg(dev *Device, data []byte) error {
    if len(data) < 33 {
        return fmt.Errorf("invalid login msg from device '%s'", dev.id)
    }

    sid := string(data[:32])
    code := data[32]

    if val, loaded := dev.pending.LoadAndDelete(sid); loaded {
        user := val.(*User)

        ok := code == 0
        errCode := 0

        if ok {
            log.Debug().Msgf("login session '%s' for device '%s' success", sid, dev.id)
            dev.users.Store(sid, user)
        } else {
            errCode = LoginErrorBusy
            log.Error().Msgf("login session '%s' for device '%s' fail, due to device busy", sid, dev.id)
        }

        if errCode == 0 {
            user.WriteMsg(websocket.TextMessage, []byte(fmt.Appendf(nil, `{"type":"login"}`)))
        } else {
            user.SendCloseMsg(LoginErrorBusy, "device busy")
        }

        user.pending <- ok
    }

    return nil
}

func handleTermDataMsg(dev *Device, data []byte) error {
    if len(data) < 32 {
        return fmt.Errorf("invalid term data msg from device '%s'", dev.id)
    }

    sid := string(data[:32])

    if val, ok := dev.users.Load(sid); ok {
        user := val.(*User)
        data[31] = 0
        user.WriteMsg(websocket.BinaryMessage, data[31:])
    }

    return nil
}

func handleFileMsg(dev *Device, data []byte) error {
    if len(data) < 33 {
        return fmt.Errorf("invalid file msg from device '%s'", dev.id)
    }

    sid := string(data[:32])
    typ := data[32]

    if val, ok := dev.users.Load(sid); ok {
        user := val.(*User)

        switch typ {
        case msgTypeFileSend:
            user.WriteMsg(websocket.TextMessage,
                fmt.Appendf(nil, `{"type":"sendfile", "name": "%s"}`, string(data[33:])))

        case msgTypeFileRecv:
            user.WriteMsg(websocket.TextMessage, []byte(`{"type":"recvfile"}`))

        case msgTypeFileData:
            data[32] = 1
            user.WriteMsg(websocket.BinaryMessage, data[32:])

        case msgTypeFileAck:
            user.WriteMsg(websocket.TextMessage, []byte(`{"type":"fileAck"}`))

        case msgTypeFileAbort:
            user.WriteMsg(websocket.BinaryMessage, []byte{1})
        }
    }

    return nil
}

func handleHttpMsg(dev *Device, data []byte) error {
    if len(data) < 18 {
        return fmt.Errorf("invalid http msg from device '%s'", dev.id)
    }

    addr := data[:18]
    data = data[18:]

    if c, ok := dev.https.Load(string(addr)); ok {
        c := c.(net.Conn)
        if len(data) == 0 {
            c.Close()
        } else {
            c.Write(data)
        }
    }

    return nil
}

func handleCmdMsg(dev *Device, data []byte) error {
    info := &CommandRespInfo{}

    err := jsoniter.Unmarshal(data, info)
    if err != nil {
        return fmt.Errorf("parse command resp info error: %v", err)
    }

    var attrs map[string]any
    err = jsoniter.Unmarshal(info.Attrs, &attrs)
    if err != nil {
        return fmt.Errorf("parse command resp attrs error: %v", err)
    }

    attrs["devid"] = dev.id

    if val, ok := dev.commands.Load(info.Token); ok {
        req := val.(*CommandReq)
        req.acked = true
        req.c.JSON(http.StatusOK, attrs)
        req.cancel()
    }

    return nil
}
