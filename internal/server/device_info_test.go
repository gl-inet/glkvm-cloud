package server

import (
	"encoding/binary"
	"testing"
	"time"
)

// A KVM reports its GL hardware model in its client info. The platform keeps
// it: the device list offers remote control for KVMs, a web proxy for other
// hosts running rtty-go.
func TestClientInfoStoresModel(t *testing.T) {
	factory := newTestCA(t)
	f := newMTLSFixture(t, factory)
	f.addCode(t, "code")
	code, conn, err := f.register(factory.device(t, "9483c4b71fb1", clientAuthEKU), "zh71fb1", "9483c4b71fb1", "code")
	if err != nil || code != 0 {
		t.Fatalf("join: (%d, %v)", code, err)
	}
	f.waitStatus(t, "zh71fb1", "online")

	info := `{"type":"client_info","client":"rtty-go","os":"linux","model":"rm1pe"}`
	head := []byte{msgTypeDeviceInfo, 0, 0}
	binary.BigEndian.PutUint16(head[1:], uint16(len(info)))
	if _, err := conn.Write(append(head, info...)); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(3 * time.Second); ; time.Sleep(10 * time.Millisecond) {
		var row struct{ Client, Model string }
		f.db.Gorm().Raw(`SELECT client, model FROM devices WHERE ddns = 'zh71fb1'`).Scan(&row)
		if row.Client == "rtty-go" && row.Model == "rm1pe" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("device row has client %q model %q, want rtty-go rm1pe", row.Client, row.Model)
		}
	}
	f.closeAndWait(t, conn, "zh71fb1")
}

func TestIsKVM(t *testing.T) {
	for info, want := range map[string]bool{
		``:                                     true, // C rtty: no client info
		`{"client":"rtty-go","model":"rm1pe"}`: true,
		`{"client":"rtty-go"}`:                 false, // another host running rtty-go
		`{"client":"rtty-go","model":""}`:      false,
		`not json`:                             true,
	} {
		dev := &Device{}
		dev.setClientInfo([]byte(info))
		if got := dev.IsKVM(); got != want {
			t.Errorf("client info %q: IsKVM() = %v, want %v", info, got, want)
		}
	}
}
