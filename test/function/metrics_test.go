package ft

import (
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/api"
	v1api "github.com/prometheus/client_golang/api/prometheus/v1"
)

func TestQuery(t *testing.T) {
	client, err := api.NewClient(api.Config{
		Address: fmt.Sprintf("http://%s:%d", host, metricsPort),
	})

	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	_ = v1api.NewAPI(client)
}
