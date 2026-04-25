package custom_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/trebent/kerberos/internal/composer"
	"github.com/trebent/kerberos/internal/composer/custom"
)

func BenchmarkCustom(b *testing.B) {
	custom := custom.NewComponent(
		&custom.Dummy{O: 3, CustomHandler: func(_ composer.FlowComponent, _ http.ResponseWriter, _ *http.Request) {
			// Do nothing.
		}},
		&custom.Dummy{O: 1, CustomHandler: func(fc composer.FlowComponent, w http.ResponseWriter, r *http.Request) {
			fc.ServeHTTP(w, r)
		}},
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8080/some/path", nil)

	b.ResetTimer()
	for b.Loop() {
		custom.ServeHTTP(rec, req)
	}
}
