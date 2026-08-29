package lib

import (
	"fmt"
	"net/http"
	"testing"

	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

func FetchMetrics(host string, port int, t testing.TB) map[string]*io_prometheus_client.MetricFamily {
	// Verify metrics standings
	t.Logf("Metrics host and port %s:%d", host, port)
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s:%d/metrics", host, port), nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := Client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Unexpected status code: got %d, want %d", resp.StatusCode, http.StatusOK)
	}

	parser := expfmt.NewTextParser(model.LegacyValidation)
	metrics, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		t.Fatalf("Failed to parse metrics: %v", err)
	}

	return metrics
}

func GetCounterValue(metricFamily *io_prometheus_client.MetricFamily, t testing.TB) float64 {
	if metricFamily.GetType() != io_prometheus_client.MetricType_COUNTER {
		t.Fatal("getCounterValue called on non-counter metric")
	}

	var total float64
	for _, metric := range metricFamily.GetMetric() {
		total += metric.GetCounter().GetValue()
	}

	return total
}
