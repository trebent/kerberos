package auth

import (
	"errors"
	"net/http"
	"net/url"
	"testing"
)

func TestFindMethod(t *testing.T) {
	a := authorizer{
		cfg: &authConfig{
			Scheme: &scheme{
				Mappings: []*mapping{
					{
						Backend: "backend1",
						Method:  "basic",
						Exempt:  []string{},
					},
					{
						Backend: "backend2",
						Method:  "basic",
						Exempt:  []string{},
					},
					{
						Backend: "backend3",
						Method:  "basic",
						Exempt: []string{
							"/url/1",
							"/",
							"/bla/bla/*/bla/ha",
						},
					},
				},
			},
		},
	}

	_, err := a.findMethod("backend1", &http.Request{URL: &url.URL{Path: "/"}})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	_, err = a.findMethod("backend2", &http.Request{URL: &url.URL{Path: "/"}})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	_, err = a.findMethod("backend3", &http.Request{URL: &url.URL{Path: "/some/url"}})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	_, err = a.findMethod("backend3", &http.Request{URL: &url.URL{Path: "/"}})
	if !errors.Is(err, errExempted) {
		t.Fatalf("Expected exemption error, got %v", err)
	}

	_, err = a.findMethod("backend3", &http.Request{URL: &url.URL{Path: "/url/1"}})
	if !errors.Is(err, errExempted) {
		t.Fatalf("Expected exemption error, got %v", err)
	}

	_, err = a.findMethod("backend3", &http.Request{URL: &url.URL{Path: "/bla/bla/hello/bla/ha"}})
	if !errors.Is(err, errExempted) {
		t.Fatalf("Expected exemption error, got %v", err)
	}
}
