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

package server

import (
    "context"
    "encoding/binary"
    "net/http"
    "sync"
    "sync/atomic"
    "time"

    "rttys/internal/store/sqlite"
    "rttys/utils"

    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
    jsoniter "github.com/json-iterator/go"
    "github.com/rs/zerolog/log"
)

type User struct {
    conn    *websocket.Conn
    sid     string
    dev     *Device
    pending chan bool
    close   sync.Once
    closed  atomic.Bool
}

type UserMsg struct {
    Type string `json:"type"`
    Cols uint16 `json:"cols"`
    Rows uint16 `json:"rows"`
    Ack  uint16 `json:"ack"`
    Size uint32 `json:"size"`
    Name string `json:"name"`
}

const (
    LoginErrorOffline = 4000
    LoginErrorBusy    = 4001
    LoginErrorTimeout = 4002
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true },
}

func handleUserConnection(srv *RttyServer, c *gin.Context) {
    defer LogPanic()

    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        log.Error().Err(err).Msg("upgrade to websocket failed")
        return
    }

    devid := c.Param("devid")
    if devid == "" {
        log.Error().Msg("device ID is required")
        conn.Close()
        return
    }

    user := &User{conn: conn}

    dev := srv.GetDevice(c.Query("group"), devid)
    if dev == nil {
        user.SendCloseMsg(LoginErrorOffline, "device not found")
        conn.Close()
        return
    }

    sid := utils.GenUniqueID()

    user.sid = sid
    user.dev = dev
    user.pending = make(chan bool, 1)

    dev.pending.Store(sid, user)

    defer user.Close()

    if err := dev.WriteMsg(msgTypeLogin, sid, nil); err != nil {
        log.Error().Msgf("send login msg to device %s fail: %v", dev.id, err)
        return
    }

    ctx, cancel := context.WithCancel(dev.ctx)

    go func() {
        <-ctx.Done()
        user.Close()
    }()

    defer cancel()

    if !waitForLogin(user, dev, ctx, sid) {
        return
    }

    var sshLogID int64
    if cont := sqlite.TryContainer(); cont != nil && cont.DeviceLogSvc != nil {
        actorID, actorName := principalFromCtx(c)
        sshLogID = cont.DeviceLogSvc.StartRemoteSSHSession(c.Request.Context(), dev.id, dev.desc, actorID, actorName, c.ClientIP())
        if cont.NotificationSvc != nil {
            cont.NotificationSvc.NotifyRemoteAccess("SSH", dev.id, dev.desc, actorName, c.ClientIP())
        }
    }
    defer func() {
        if sshLogID > 0 {
            if cont := sqlite.TryContainer(); cont != nil && cont.DeviceLogSvc != nil {
                cont.DeviceLogSvc.EndSession(context.Background(), sshLogID)
            }
        }
    }()

    for {
        msgType, data, err := conn.ReadMessage()
        if err != nil {
            if !user.closed.Load() {
                closeError, ok := err.(*websocket.CloseError)
                if !ok || (closeError.Code != websocket.CloseGoingAway &&
                    closeError.Code != websocket.CloseAbnormalClosure &&
                    closeError.Code != websocket.CloseNormalClosure) {
                    log.Error().Msgf("user read fail: %v", err)
                }
            }
            return
        }

        if msgType == websocket.BinaryMessage {
            if len(data) < 1 {
                log.Error().Msgf("invalid msg from user")
                return
            }

            typ := msgTypeTermData
            if data[0] == 1 {
                typ = msgTypeFile
            }

            err = dev.WriteMsg(typ, sid, data[1:])
        } else {
            msg := &UserMsg{}

            err = jsoniter.Unmarshal(data, msg)
            if err != nil {
                log.Error().Msgf("invalid msg from user")
                return
            }

            switch msg.Type {
            case "winsize":
                b := make([]byte, 4)

                binary.BigEndian.PutUint16(b, msg.Cols)
                binary.BigEndian.PutUint16(b[2:], msg.Rows)

                err = dev.WriteMsg(msgTypeWinsize, sid, b)

            case "ack":
                b := make([]byte, 2)
                binary.BigEndian.PutUint16(b, msg.Ack)
                err = dev.WriteMsg(msgTypeAck, sid, b)

            case "fileInfo":
                b := make([]byte, 4+len(msg.Name))
                binary.BigEndian.PutUint32(b, msg.Size)
                copy(b[4:], []byte(msg.Name))

                err = dev.WriteFileMsg(msgTypeFile, sid, msgTypeFileInfo, b)

            case "fileCanceled":
                err = dev.WriteFileMsg(msgTypeFile, sid, msgTypeFileAbort, nil)

            case "fileAck":
                err = dev.WriteFileMsg(msgTypeFile, sid, msgTypeFileAck, nil)
            }
        }

        if err != nil {
            log.Error().Msgf("write msg to device '%s' fail: %v", dev.id, err)
            return
        }
    }
}

func (user *User) SendCloseMsg(code int, text string) {
    user.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, text), time.Now().Add(time.Second))
}

func (user *User) Close() {
    user.close.Do(func() {
        dev := user.dev
        sid := user.sid

        user.closed.Store(true)

        if _, loaded := dev.users.LoadAndDelete(sid); loaded {
            dev.WriteMsg(msgTypeLogout, sid, nil)
        }

        dev.pending.Delete(sid)
        user.conn.Close()

        log.Debug().Msgf("user with session '%s' closed", sid)
    })
}

func (user *User) WriteMsg(typ int, data []byte) error {
    return user.conn.WriteMessage(typ, data)
}

func waitForLogin(user *User, dev *Device, ctx context.Context, sid string) bool {
    for {
        select {
        case <-ctx.Done():
            return false

        case ok := <-user.pending:
            return ok

        case <-time.After(TermLoginTimeout):
            if _, loaded := dev.pending.LoadAndDelete(sid); loaded {
                log.Error().Msgf("login timeout for session %s of device %s", sid, dev.id)
                user.SendCloseMsg(LoginErrorTimeout, "login timeout")
                return false
            }
        }
    }
}
