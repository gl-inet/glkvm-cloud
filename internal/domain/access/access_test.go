package access

import (
	"strings"
	"testing"
)

func TestAccessCodeRoundTrip(t *testing.T) {
	in := AccessCode{Addr: "192.0.2.10:5915", CA: []byte{0x30, 0x82, 0x01, 0x02}, Code: "abc_DEF-123"}
	s := in.String()
	if !strings.HasPrefix(s, "glkvm1:") || strings.ContainsAny(s, " \n\t+/=") {
		t.Fatalf("access code %q is not a single URL-safe token with the version prefix", s)
	}

	// Pasted with stray whitespace and a line break.
	out, err := ParseAccessCode("  " + s[:20] + "\n" + s[20:] + " ")
	if err != nil {
		t.Fatal(err)
	}
	if out.Addr != in.Addr || out.Code != in.Code || string(out.CA) != string(in.CA) {
		t.Errorf("round trip = %+v, want %+v", out, in)
	}

	// No CA: the device falls back to system roots.
	out, err = ParseAccessCode(AccessCode{Addr: "kvm.example.com:5915", Code: "x"}.String())
	if err != nil || out.CA != nil {
		t.Errorf("CA-less code = (%+v, %v), want no CA", out, err)
	}
}

func TestParseAccessCodeRejects(t *testing.T) {
	for _, s := range []string{
		"",
		"glkvm2:abc",      // unknown version
		"glkvm1:%%%",      // not base64
		"glkvm1:" + "e30", // {} — no address or code
		AccessCode{Addr: "no-port", Code: "x"}.String(),
		AccessCode{Addr: "h:5915"}.String(), // no code
	} {
		if _, err := ParseAccessCode(s); err == nil {
			t.Errorf("ParseAccessCode(%q) accepted", s)
		}
	}
}

func TestCodeStatus(t *testing.T) {
	const now = 1000
	cases := []struct {
		c    RegistrationCode
		want CodeStatus
	}{
		{RegistrationCode{ExpiresAt: now + 1}, CodeUnused},
		{RegistrationCode{ExpiresAt: now}, CodeExpired},
		{RegistrationCode{ExpiresAt: now + 1, UsedAt: 5}, CodeUsed},
		{RegistrationCode{ExpiresAt: now - 1, UsedAt: 5}, CodeUsed},
		{RegistrationCode{ExpiresAt: now + 1, RevokedAt: 5}, CodeRevoked},
	}
	for _, tc := range cases {
		if got := tc.c.Status(now); got != tc.want {
			t.Errorf("%+v: status %s, want %s", tc.c, got, tc.want)
		}
	}

	a, _ := NewCode()
	b, _ := NewCode()
	if len(a) < 20 || a == b {
		t.Errorf("NewCode gave %q and %q", a, b)
	}
}
