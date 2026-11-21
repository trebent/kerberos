package security

import (
	"net/http"
	"slices"
	"testing"
)

func TestStripKerberosCookies(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		want  []string
	}{
		{name: "no cookie header", lines: nil, want: nil},
		{
			name:  "only Kerberos cookies",
			lines: []string{"session=abc; refresh=def; csrf=ghi"},
			want:  nil,
		},
		{
			name:  "Kerberos and other cookies",
			lines: []string{"grafana_session=x; session=abc; theme=dark; csrf=ghi"},
			want:  []string{"grafana_session=x; theme=dark"},
		},
		{
			name:  "several cookie lines",
			lines: []string{"session=abc", "lang=en; refresh=def"},
			want:  []string{"lang=en"},
		},
		{
			name:  "similar names are kept",
			lines: []string{"session_id=1; my-session=2; sessionx=3"},
			want:  []string{"session_id=1; my-session=2; sessionx=3"},
		},
		{
			name:  "values are kept as sent",
			lines: []string{`a="quoted value"; session=abc; b=x=y`},
			want:  []string{`a="quoted value"; b=x=y`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := http.Header{}
			for _, l := range tt.lines {
				h.Add("Cookie", l)
			}

			StripKerberosCookies(h)

			if got := h.Values("Cookie"); !slices.Equal(got, tt.want) {
				t.Fatalf("expected Cookie %q, got %q", tt.want, got)
			}
		})
	}
}
