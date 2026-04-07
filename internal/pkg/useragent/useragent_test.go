package useragent

import "testing"

func TestFriendly(t *testing.T) {
	cases := []struct {
		name string
		ua   string
		want string
	}{
		{
			name: "empty",
			ua:   "",
			want: "Unknown",
		},
		{
			name: "chrome on windows 10",
			ua:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36",
			want: "Chrome 146 · Windows",
		},
		{
			name: "edge on windows",
			ua:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36 Edg/130.0.0.0",
			want: "Edge 130 · Windows",
		},
		{
			name: "firefox on linux",
			ua:   "Mozilla/5.0 (X11; Linux x86_64; rv:120.0) Gecko/20100101 Firefox/120.0",
			want: "Firefox 120 · Linux",
		},
		{
			name: "safari on macOS",
			ua:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
			want: "Safari 17 · macOS",
		},
		{
			name: "chrome on android 13",
			ua:   "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
			want: "Chrome 120 · Android 13",
		},
		{
			name: "safari on iphone 17",
			ua:   "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
			want: "Safari 17 · iOS 17",
		},
		{
			name: "opera",
			ua:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 OPR/106.0.0.0",
			want: "Opera 106 · Windows",
		},
		{
			name: "unknown UA",
			ua:   "curl/8.0.1",
			want: "curl/8.0.1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Friendly(tc.ua)
			if got != tc.want {
				t.Errorf("Friendly(%q) = %q, want %q", tc.ua, got, tc.want)
			}
		})
	}
}
