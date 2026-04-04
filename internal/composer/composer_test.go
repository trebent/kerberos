package composer

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
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

func (t *testFlow) Next(next FlowComponent) {
	t.next = next
}

// GetMeta implements [FlowComponent].
func (t *testFlow) GetMeta() []adminapi.FlowMeta {
	meta := &adminapi.FlowMeta{
		Name: t.name,
		Data: adminapi.FlowMeta_Data{},
	}

	if t.next != nil {
		return append([]adminapi.FlowMeta{*meta}, t.next.GetMeta()...)
	}

	return []adminapi.FlowMeta{*meta}
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

func TestComposerGetFlow(t *testing.T) {
	one := newTestFlow("obs", t)
	two := newTestFlow("router", t)
	three := newTestFlow("composable", t)
	four := newTestFlow("forwarder", t)

	c := New(&Opts{
		Observability: one,
		Router:        two,
		Custom:        three,
		Forwarder:     four,
	})

	meta := c.GetFlow()

	if meta[0].Name != "obs" {
		t.Fatalf("expected first meta name 'obs', got %q", meta[0].Name)
	}
	if meta[1].Name != "router" {
		t.Fatalf("expected second meta name 'router', got %q", meta[1].Name)
	}
	if meta[2].Name != "composable" {
		t.Fatalf("expected third meta name 'composable', got %q", meta[2].Name)
	}
	if meta[3].Name != "forwarder" {
		t.Fatalf("expected fourth meta name 'forwarder', got %q", meta[3].Name)
	}
	if len(meta) > 4 {
		t.Fatal("expected meta length to be 4 (end of chain)")
	}
}

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
