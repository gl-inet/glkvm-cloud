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
	"flag"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"rttys/xconfig"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

func TestRttysStress(t *testing.T) {
    duration := 10 * time.Minute

    timeoutFlag := flag.Lookup("test.timeout")
    if timeoutFlag != nil {
        duration = timeoutFlag.Value.(flag.Getter).Get().(time.Duration)
    }

    cfg := xconfig.Config{
        AddrDev:  ":5912",
        AddrUser: ":5913",
    }

    srv := &RttyServer{cfg: cfg}

    go func() {
        err := srv.Run()
        if err != nil {
            log.Fatal().Msg(err.Error())
        }
    }()

    ctx, cancel := context.WithTimeout(context.Background(), duration-time.Second*2)
    defer cancel()

    time.Sleep(time.Millisecond * 100)

    log.Info().Msg("Waiting for devices to connect for testing...")

    devices := &sync.Map{}

    for {
        select {
        case <-ctx.Done():
            log.Info().Msg("Test timeout, exiting...")
            return
        default:
            time.Sleep(time.Second * 1)

            srv.groups.Range(func(key, value any) bool {
                group := key.(string)
                g := value.(*DeviceGroup)
                g.devices.Range(func(key, value any) bool {
                    dev := value.(*Device)
                    if _, loaded := devices.LoadOrStore(dev.id, group+dev.id); !loaded {
                        go runDeviceTest(ctx, devices, group, dev.id)
                    }
                    return true
                })
                return true
            })
        }
    }
}

func runDeviceTest(ctx context.Context, devices *sync.Map, group, devID string) {
    ctx, cancel := context.WithCancel(ctx)

    defer func() {
        time.Sleep(time.Second)
        cancel()
        devices.Delete(group + devID)
    }()

    go runHttpTest(ctx, group, devID)

    wg := &sync.WaitGroup{}

    for range 7 {
        wg.Add(1)
        go runWebSocketTest(ctx, group, devID, wg)
    }

    wg.Wait()
}

func runWebSocketTest(ctx context.Context, group, devID string, wg *sync.WaitGroup) {
    conn, _, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:5913/connect/"+devID+"?group="+group, nil)
    if err != nil {
        log.Fatal().Msg(err.Error())
    }
    defer conn.Close()
    defer wg.Done()

    go func() {
        <-ctx.Done()
        conn.Close()
    }()

    go func() {
        msg := []byte{0}
        msg = append(msg, []byte("ttttttttttttttttttttttttttttt\n")...)
        msg = append(msg, []byte("ttttttttttttttttttttttttttttt\n")...)
        for {
            err = conn.WriteMessage(websocket.BinaryMessage, msg)
            if err != nil {
                return
            }
            time.Sleep(time.Millisecond * 20)
        }
    }()

    for {
        _, _, err := conn.ReadMessage()
        if err != nil {
            return
        }
    }
}

func runHttpTest(ctx context.Context, group, devID string) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            runHttpTestOnce(ctx, group, devID)
        }
    }
}

func runHttpTestOnce(ctx context.Context, group, devID string) {
    addr := ""

    if group == "" {
        addr = "http://127.0.0.1:5913/web/"
    } else {
        addr = "http://127.0.0.1:5913/web2/" + group + "/"
    }

    addr += devID + "/http/" + encodeURIComponent("127.0.0.1:80/")

    jar, _ := cookiejar.New(nil)
    client := &http.Client{
        Jar: jar,
    }

    request, _ := http.NewRequestWithContext(ctx, "GET", addr, nil)

    for range 10 {
        res, err := client.Do(request)
        if err != nil {
            log.Info().Msg(err.Error())
            return
        }
        defer res.Body.Close()

        io.ReadAll(res.Body)

        time.Sleep(10 * time.Millisecond)
    }
}

func encodeURIComponent(str string) string {
    r := url.QueryEscape(str)
    r = strings.ReplaceAll(r, "+", "%20")
    return r
}
