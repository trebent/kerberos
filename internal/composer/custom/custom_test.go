package custom

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/trebent/kerberos/internal/composer/types"
)

func TestCustom(t *testing.T) {
	wg1 := sync.WaitGroup{}
	wg1.Add(1)
	wg2 := sync.WaitGroup{}
	wg2.Add(1)
	wg3 := sync.WaitGroup{}
	wg3.Add(1)
	wgFinal := sync.WaitGroup{}
	wgFinal.Add(1)

	custom := NewComponent(
		&types.Dummy{O: 3, CustomHandler: func(fc types.FlowComponent, w http.ResponseWriter, r *http.Request) {
			t.Log("Running component 3")
			wg3.Done()
			fc.ServeHTTP(w, r)
		}},
		&types.Dummy{O: 2, CustomHandler: func(fc types.FlowComponent, w http.ResponseWriter, r *http.Request) {
			t.Log("Running component 2")
			wg2.Done()
			fc.ServeHTTP(w, r)
		}},
		&types.Dummy{O: 1, CustomHandler: func(fc types.FlowComponent, w http.ResponseWriter, r *http.Request) {
			t.Log("Running component 1")
			wg1.Done()
			fc.ServeHTTP(w, r)
		}},
	)
	finalComp := &types.Dummy{
		CustomHandler: func(fc types.FlowComponent, w http.ResponseWriter, r *http.Request) {
			t.Log("Running final componnet")
			wgFinal.Done()
		},
	}
	custom.Next(finalComp)

	custom.ServeHTTP(httptest.NewRecorder(), &http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Path: "some/path"},
	})
	wg1.Wait()
	wg2.Wait()
	wg3.Wait()
	wgFinal.Wait()
}
