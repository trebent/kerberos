package security

import (
	"net/http"
	"net/http/httptest"
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
			name:  "quoted values are kept",
			lines: []string{`a="quoted"; session=abc; b=x=y`},
			want:  []string{`a="quoted"; b=x=y`},
		},
		{
			name:  "cookies Go cannot parse are dropped",
			lines: []string{"bad=caf\u00e9; b@d=1; esc=a\\b; theme=dark; session=abc"},
			want:  []string{"theme=dark"},
		},
		{
			name:  "values with spaces or commas are quoted",
			lines: []string{"a=has space; session=abc; b=x,y"},
			want:  []string{`a="has space"; b="x,y"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			for _, l := range tt.lines {
				r.Header.Add("Cookie", l)
			}

			StripKerberosCookies(r)

			if got := r.Header.Values("Cookie"); !slices.Equal(got, tt.want) {
				t.Fatalf("expected Cookie %q, got %q", tt.want, got)
			}
		})
	}
}
