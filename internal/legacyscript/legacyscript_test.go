package legacyscript

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testCA(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test deployment CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

const turnJSON = `{
    "username": "turnuser",
    "ttl": 864000,
    "password": "p@ss'word",
    "uris": [
        "turn:192.0.2.10:3478?transport=udp",
        "turn:192.0.2.10:3478?transport=tcp"
    ]
}`

func params(t *testing.T) Params {
	return Params{
		Server:   "192.0.2.10:5912",
		Token:    `to'k$en "with" spaces` + "`x`",
		CA:       testCA(t),
		TURN:     turnJSON,
		WebUIURL: "https://kvm.example.com",
		Version:  "v3.1.0",
	}
}

// heredoc returns the body of the heredoc that ends with delimiter.
func heredoc(t *testing.T, script, delimiter string) string {
	t.Helper()
	start := strings.Index(script, "<<'"+delimiter+"'\n")
	if start < 0 {
		t.Fatalf("no heredoc %s", delimiter)
	}
	body := script[start+len("<<'"+delimiter+"'\n"):]
	end := strings.Index(body, "\n"+delimiter+"\n")
	if end < 0 {
		t.Fatalf("heredoc %s is not closed", delimiter)
	}
	return body[:end+1]
}

// checkSyntax runs sh -n over script with every shell at hand; busybox is
// what the KVM runs.
func checkSyntax(t *testing.T, name, script string) {
	t.Helper()
	f := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(f, []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	ran := false
	for _, sh := range [][]string{{"busybox", "sh"}, {"dash"}, {"sh"}} {
		if _, err := exec.LookPath(sh[0]); err != nil {
			continue
		}
		ran = true
		if out, err := exec.Command(sh[0], append(sh[1:], "-n", f)...).CombinedOutput(); err != nil {
			t.Errorf("%s: %s -n: %v\n%s", name, strings.Join(sh, " "), err, out)
		}
	}
	if !ran {
		t.Skip("no shell to check the syntax with")
	}
}

func TestRenderSyntax(t *testing.T) {
	s, err := Render(params(t))
	if err != nil {
		t.Fatal(err)
	}
	checkSyntax(t, "kvm.sh", s.Text)
	checkSyntax(t, "S01selfCloud", heredoc(t, s.Text, "GLKVM_S01"))
	checkSyntax(t, "rtty-loop.sh", heredoc(t, s.Text, "GLKVM_LOOP"))
}

func TestRenderContent(t *testing.T) {
	p := params(t)
	s, err := Render(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := heredoc(t, s.Text, "GLKVM_CA"); got != p.CA {
		t.Errorf("CA written = %q, want the deployment CA", got)
	}
	s01 := heredoc(t, s.Text, "GLKVM_S01")
	if got := heredoc(t, s01, "GLKVM_TURN"); got != turnJSON+"\n" {
		t.Errorf("TURN config written = %q", got)
	}
	if !strings.Contains(s01, `pkill -f "rtty.*-d $device_mac"`) {
		t.Error("stop does not end the rtty the firmware's cleanup expects")
	}

	loop := heredoc(t, s.Text, "GLKVM_LOOP")
	for _, want := range []string{
		// Old firmware checks the platform against the CA given with -C.
		`rtty -s -C "$CA_FILE" -T 2 -I "$device_id" -h "$SERVER" -t "$TOKEN" -d "$device_mac" &`,
		// Newer firmware, without -s/-x/-C, gets the CA in its fixed file.
		`cp "$CA_FILE" "$FW_CA.new" && mv -f "$FW_CA.new" "$FW_CA"`,
		`rtty -T 2 -I "$device_id" -h "$SERVER" -t "$TOKEN" -d "$device_mac" &`,
		`FW_CA=/var/run/gl-rtty/ca.crt`,
	} {
		if !strings.Contains(loop, want) {
			t.Errorf("watchdog lacks %q", want)
		}
	}
	for _, bad := range []string{"-sx", "rtty -a", " -x "} {
		if strings.Contains(loop, bad) {
			t.Errorf("watchdog runs rtty with %q", bad)
		}
	}
	if !strings.Contains(s.Text, "GL.iNet Cloud and the self-hosted platform cannot be used together") {
		t.Error("the script does not warn about GL.iNet Cloud")
	}
	if s.CAFingerprint == "" || strings.Count(s.CAFingerprint, ":") != 31 {
		t.Errorf("CA fingerprint = %q", s.CAFingerprint)
	}
}

// Values reach the device exactly as given, whatever characters they hold.
func TestRenderQuoting(t *testing.T) {
	p := params(t)
	s, err := Render(p)
	if err != nil {
		t.Fatal(err)
	}
	loop := heredoc(t, s.Text, "GLKVM_LOOP")
	var assigns []string
	for _, line := range strings.Split(loop, "\n") {
		if strings.HasPrefix(line, "SERVER=") || strings.HasPrefix(line, "TOKEN=") {
			assigns = append(assigns, line)
		}
	}
	if len(assigns) != 2 {
		t.Fatalf("assignments = %q", assigns)
	}
	out, err := exec.Command("sh", "-c", strings.Join(assigns, "\n")+"\nprintf '%s\\n%s' \"$SERVER\" \"$TOKEN\"").Output()
	if err != nil {
		t.Fatal(err)
	}
	if want := p.Server + "\n" + p.Token; string(out) != want {
		t.Errorf("shell sees %q, want %q", out, want)
	}

	// The web UI address goes into JSON, quoted for printf.
	if !strings.Contains(s.Text, `'"https://kvm.example.com"'`) {
		t.Error("web UI URL is not passed as a quoted JSON string")
	}
}

func TestRenderWithoutTURN(t *testing.T) {
	p := params(t)
	p.TURN = ""
	s, err := Render(p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(s.Text, "turnserver.json") {
		t.Error("TURN config written although the platform has no TURN server")
	}
	checkSyntax(t, "S01selfCloud", heredoc(t, s.Text, "GLKVM_S01"))
}

func TestRenderRejects(t *testing.T) {
	for name, mod := range map[string]func(*Params){
		"no server":         func(p *Params) { p.Server = "" },
		"no token":          func(p *Params) { p.Token = "" },
		"no CA":             func(p *Params) { p.CA = "" },
		"not a certificate": func(p *Params) { p.CA = "-----BEGIN PRIVATE KEY-----\nAA==\n-----END PRIVATE KEY-----\n" },
		"delimiter in TURN": func(p *Params) { p.TURN = "{\n}\nGLKVM_S01\nrm -rf /\n" },
	} {
		p := params(t)
		mod(&p)
		if _, err := Render(p); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

// Anywhere but on a GL KVM the script stops before it writes anything.
func TestScriptStopsOffKVM(t *testing.T) {
	if _, err := os.Stat("/proc/gl-hw-info"); err == nil {
		t.Skip("running on a GL device: the script would install itself")
	}
	s, err := Render(params(t))
	if err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("sh", "-c", s.Text).CombinedOutput()
	if err == nil || !strings.Contains(string(out), "for GL KVM devices only") {
		t.Errorf("script off a KVM: err=%v out=%q, want it to stop", err, out)
	}
	if strings.Contains(string(out), "starting self-hosted cloud") {
		t.Error("the script went on to install itself")
	}
}
