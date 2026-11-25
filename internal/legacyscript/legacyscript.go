// Package legacyscript renders the script that adds a GL KVM on firmware
// below V1.11.0, which cannot use access codes, to the platform. It is pasted
// into the KVM's terminal and keeps the KVM's C rtty connected to the token
// port, checking the platform against the deployment CA.
package legacyscript

import (
	_ "embed"
	"encoding/json"
	"errors"
	"strings"
	"text/template"

	"rttys/internal/pkg/pki"
)

//go:embed kvm.sh.tmpl
var kvmScript string

var tmpl = template.Must(template.New("kvm.sh").Funcs(template.FuncMap{"sh": shQuote}).Parse(kvmScript))

// Params is what goes into the script.
type Params struct {
	Server   string // host:port of the token port
	Token    string // the shared device token
	CA       string // the deployment CA, PEM
	TURN     string // contents of /tmp/turnserver.json; "" = no TURN server
	WebUIURL string // the platform's web UI, shown in the KVM's own web UI
	Version  string // platform version, named in the script's header
}

// Script is a rendered script and the fingerprint of the CA in it.
type Script struct {
	Text          string
	CAFingerprint string
}

// heredoc delimiters in the script: no value written through one may hold a
// line equal to it.
var delimiters = []string{"GLKVM_CA", "GLKVM_S01", "GLKVM_TURN", "GLKVM_LOOP"}

// Render builds the script for p.
func Render(p Params) (Script, error) {
	switch {
	case p.Server == "":
		return Script{}, errors.New("legacyscript: no device access address")
	case p.Token == "":
		return Script{}, errors.New("legacyscript: no device token")
	}
	fp, err := pki.DisplayFingerprint([]byte(p.CA))
	if err != nil {
		return Script{}, err
	}
	ca := p.CA
	if !strings.HasSuffix(ca, "\n") {
		ca += "\n"
	}
	for _, v := range []string{ca, p.TURN} {
		for _, line := range strings.Split(v, "\n") {
			for _, d := range delimiters {
				if strings.TrimSpace(line) == d {
					return Script{}, errors.New("legacyscript: value holds a heredoc delimiter")
				}
			}
		}
	}
	web, _ := json.Marshal(p.WebUIURL)

	var b strings.Builder
	err = tmpl.Execute(&b, struct {
		Params
		CA        string
		WebUIJSON string
	}{p, ca, string(web)})
	if err != nil {
		return Script{}, err
	}
	return Script{Text: b.String(), CAFingerprint: fp}, nil
}

// shQuote quotes s for the shell as a single word, taken literally.
func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
