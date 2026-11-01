package totp

import (
	"testing"
	"time"
)

// RFC 6238 Appendix B (SHA1, 8 digits) truncated to 6 digits: the secret
// "12345678901234567890" in base32, at T=59s, yields 94287082 -> "287082".
func TestCodeMatchesRFC6238(t *testing.T) {
	const secret = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	got, err := Code(secret, time.Unix(59, 0))
	if err != nil {
		t.Fatal(err)
	}
	if got != "287082" {
		t.Fatalf("Code = %q, want 287082", got)
	}
}

func TestCodeRoundTripsWithVerify(t *testing.T) {
	secret, _, err := GenerateSecret("glkvm-cloud", "test")
	if err != nil {
		t.Fatal(err)
	}
	code, err := Code(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !Verify(secret, code) {
		t.Fatalf("Verify rejected Code output %q", code)
	}
}
