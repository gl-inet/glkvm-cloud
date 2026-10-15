package access

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"time"
)

const (
	// CodeTTL is how long a registration code can be redeemed.
	CodeTTL = 24 * time.Hour
	// MaxCodesPerRequest caps how many codes one request may generate.
	MaxCodesPerRequest = 20
)

// NewCode returns a fresh random registration code (128 bits).
func NewCode() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// AccessCode is what an admin hands to a device owner, pasted as one string:
// where the platform's device port is, the CA its certificate chains to, and
// a one-time registration code. Nobody has to handle addresses or
// certificates by hand.
type AccessCode struct {
	Addr string // host:port of the mTLS device port
	CA   []byte // DER of the CA to verify the device port against; nil = system roots
	Code string // one-time registration code
}

// accessCodePrefix names the format version; a new format gets a new prefix
// so codes already handed out keep working.
const accessCodePrefix = "glkvm1:"

type accessCodeJSON struct {
	Addr string `json:"a"`
	CA   string `json:"ca,omitempty"` // base64 DER
	Code string `json:"c"`
}

// String encodes the access code.
func (a AccessCode) String() string {
	j := accessCodeJSON{Addr: a.Addr, Code: a.Code}
	if len(a.CA) > 0 {
		j.CA = base64.StdEncoding.EncodeToString(a.CA)
	}
	b, _ := json.Marshal(j)
	return accessCodePrefix + base64.RawURLEncoding.EncodeToString(b)
}

var errBadAccessCode = errors.New("not a valid access code")

// ParseAccessCode decodes an access code. Whitespace (from copy and paste)
// is ignored.
func ParseAccessCode(s string) (AccessCode, error) {
	s = strings.Join(strings.Fields(s), "")
	rest, ok := strings.CutPrefix(s, accessCodePrefix)
	if !ok {
		return AccessCode{}, errBadAccessCode
	}
	b, err := base64.RawURLEncoding.DecodeString(rest)
	if err != nil {
		return AccessCode{}, errBadAccessCode
	}
	var j accessCodeJSON
	if err := json.Unmarshal(b, &j); err != nil {
		return AccessCode{}, errBadAccessCode
	}
	if _, _, err := net.SplitHostPort(j.Addr); err != nil || j.Code == "" {
		return AccessCode{}, errBadAccessCode
	}
	a := AccessCode{Addr: j.Addr, Code: j.Code}
	if j.CA != "" {
		if a.CA, err = base64.StdEncoding.DecodeString(j.CA); err != nil {
			return AccessCode{}, errBadAccessCode
		}
	}
	return a, nil
}
