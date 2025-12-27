package composer

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/trebent/kerberos/internal/composer/types"
)

type testFlow struct {
	name   string
	t      *testing.T
	next   types.FlowComponent
	called sync.WaitGroup
}

func newTestFlow(name string, t *testing.T) *testFlow {
	f := &testFlow{}
	f.name = name
	f.t = t
	f.called = sync.WaitGroup{}
	f.called.Add(1)
	return f
}

func (t *testFlow) Next(next types.FlowComponent) {
	t.next = next
}

// ServeHTTP implements [types.FlowComponent].
func (t *testFlow) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.t.Logf("In test flow: %s", t.name)
	t.called.Done()
	if t.next == nil {
		w.WriteHeader(http.StatusOK)
	} else {
		t.next.ServeHTTP(w, r)
	}
}

var _ types.FlowComponent = (*testFlow)(nil)

func TestComposerFlow(t *testing.T) {
	one := newTestFlow("obs", t)
	two := newTestFlow("router", t)
	three := newTestFlow("composable", t)
	four := newTestFlow("forwarder", t)

	composer := New(&Opts{
		Observability: one,
		Router:        two,
		Custom:        three,
		Forwarder:     four,
	})

	req, err := http.NewRequest(http.MethodGet, "http://example.com/test", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	t.Log("Serving the request...")
	recorder := httptest.NewRecorder()
	composer.ServeHTTP(recorder, req)

	t.Log("Awaiting one...")
	one.called.Wait()
	t.Log("Awaiting two...")
	two.called.Wait()
	t.Log("Awaiting three...")
	three.called.Wait()
	t.Log("Awaiting four...")
	four.called.Wait()

	if recorder.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, recorder.Result().StatusCode)
	}
}
