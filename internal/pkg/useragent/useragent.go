// Package useragent provides a tiny, dependency-free helper that turns a raw
// HTTP User-Agent header into a short human-readable label such as
// "Chrome 146 · Windows" or "Safari 17 · iOS 17". It only recognises the
// browsers / OSes that we actually display in the trusted-device list — for
// anything unknown it falls back to a truncated copy of the original UA.
package useragent

import (
	"regexp"
	"strings"
)

// Friendly turns a raw User-Agent header into a short human-readable label.
// Returns "Unknown" for an empty input.
func Friendly(ua string) string {
	ua = strings.TrimSpace(ua)
	if ua == "" {
		return "Unknown"
	}

	browser := parseBrowser(ua)
	os := parseOS(ua)

	switch {
	case browser != "" && os != "":
		return browser + " · " + os
	case browser != "":
		return browser
	case os != "":
		return os
	}

	// Fallback: truncated raw UA so we never lose information entirely.
	if len(ua) > 80 {
		return ua[:80] + "…"
	}
	return ua
}

// --- browser detection ----------------------------------------------------

// Order matters: Edge / Opera / Brave embed "Chrome" in their UA, so they must
// be checked first. Likewise Chrome embeds "Safari", so Safari is last.
var browserPatterns = []struct {
	name string
	re   *regexp.Regexp
}{
	{"Edge", regexp.MustCompile(`Edg(?:e|A|iOS)?/(\d+)`)},
	{"Opera", regexp.MustCompile(`(?:OPR|Opera)/(\d+)`)},
	{"Vivaldi", regexp.MustCompile(`Vivaldi/(\d+)`)},
	{"Firefox", regexp.MustCompile(`Firefox/(\d+)`)},
	{"Chrome", regexp.MustCompile(`(?:Chrome|CriOS)/(\d+)`)},
	{"Safari", regexp.MustCompile(`Version/(\d+)[\d.]*\s+.*Safari/`)},
}

func parseBrowser(ua string) string {
	for _, p := range browserPatterns {
		m := p.re.FindStringSubmatch(ua)
		if len(m) >= 2 {
			return p.name + " " + m[1]
		}
	}
	if strings.Contains(ua, "Safari/") {
		return "Safari"
	}
	return ""
}

// --- OS detection ---------------------------------------------------------

var (
	reAndroid = regexp.MustCompile(`Android (\d+)`)
	reIOS     = regexp.MustCompile(`(?:iPhone OS|CPU OS) (\d+)`)
	reMac     = regexp.MustCompile(`Mac OS X (\d+)[._](\d+)`)
)

func parseOS(ua string) string {
	switch {
	case strings.Contains(ua, "Windows NT"):
		// Windows NT 10.0 covers Windows 10 and 11; Microsoft never bumped the
		// NT version, so we can't tell them apart from the UA alone.
		return "Windows"
	case strings.Contains(ua, "Android"):
		if m := reAndroid.FindStringSubmatch(ua); len(m) >= 2 {
			return "Android " + m[1]
		}
		return "Android"
	case strings.Contains(ua, "iPhone") || strings.Contains(ua, "iPad"):
		if m := reIOS.FindStringSubmatch(ua); len(m) >= 2 {
			return "iOS " + m[1]
		}
		return "iOS"
	case strings.Contains(ua, "Mac OS X"):
		if m := reMac.FindStringSubmatch(ua); len(m) >= 3 {
			major := m[1]
			minor := m[2]
			// Mac OS X 10.x is "macOS", 11+ is also "macOS" but with the major
			// number directly.
			if major == "10" {
				return "macOS"
			}
			return "macOS " + major + "." + minor
		}
		return "macOS"
	case strings.Contains(ua, "CrOS"):
		return "ChromeOS"
	case strings.Contains(ua, "Linux"):
		return "Linux"
	}
	return ""
}
