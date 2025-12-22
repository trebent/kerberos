package composer

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type testFlow struct {
	name   string
	t      *testing.T
	next   FlowComponent
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

func (t *testFlow) Compose(next FlowComponent) FlowComponent {
	t.next = next
	return t
}

// ServeHTTP implements [FlowComponent].
func (t *testFlow) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.t.Logf("In test flow: %s", t.name)
	t.called.Done()
	if t.next == nil {
		w.WriteHeader(http.StatusOK)
	} else {
		t.next.ServeHTTP(w, r)
	}
}

var _ FlowComponent = (*testFlow)(nil)

func TestComposerFlow(t *testing.T) {
	one := newTestFlow("obs", t)
	two := newTestFlow("router", t)
	three := newTestFlow("composable", t)
	four := newTestFlow("forwarder", t)

	composer := &Composer{
		Observability: one,
		Router:        two,
		Composable:    three,
		Forwarder:     four,
	}

	req, err := http.NewRequest(http.MethodGet, "http://example.com/test", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	t.Log("Serving the request...")
	composer.ServeHTTP(httptest.NewRecorder(), req)

	t.Log("Awaiting one...")
	one.called.Wait()
	t.Log("Awaiting two...")
	two.called.Wait()
	t.Log("Awaiting three...")
	three.called.Wait()
	t.Log("Awaiting four...")
	four.called.Wait()
}
