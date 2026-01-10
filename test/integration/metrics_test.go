package integration

import (
	"fmt"
	"net/http"
	"testing"

	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

// Verifies that basic metrics are present and incremented as expected.
func TestMetricsBasic(t *testing.T) {
	startMetrics := fetchMetrics(t)

	url := fmt.Sprintf("http://%s:%d/gw/backend/echo/metrics-test", getHost(), getPort())
	_ = get(url, t)
	_ = put(url, []byte("metrics test"), t)
	_ = post(url, []byte("metrics test"), t)
	_ = delete(url, t)
	_ = patch(url, []byte("metrics test"), t)

	endMetrics := fetchMetrics(t)
	for metricName, endMetric := range endMetrics {
		switch metricName {
		case "request_count_total":
			t.Log("Verifying request_count_total metric")

			startCount := float64(0)
			if startMetric, exists := startMetrics[metricName]; exists {
				startCount = getCounterValue(startMetric)
			}
			endCount := getCounterValue(endMetric)

			if endCount-startCount != 5.0 {
				t.Errorf("metric %s did not increment as expected: got %f, want %f", metricName, endCount-startCount, 5.0)
			}
		case "response_total":
			t.Log("Verifying response_total metric")

			startCount := float64(0)
			if startMetric, exists := startMetrics[metricName]; exists {
				startCount = getCounterValue(startMetric)
			}
			endCount := getCounterValue(endMetric)

			if endCount-startCount != 5.0 {
				t.Errorf("metric %s did not increment as expected: got %f, want %f", metricName, endCount-startCount, 5.0)
			}
		}
	}
}

func fetchMetrics(t *testing.T) map[string]*io_prometheus_client.MetricFamily {
	// Verify metrics standings
	t.Logf("Metrics host and port %s:%d", getHost(), getMetricsPort())
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s:%d/metrics", getHost(), getMetricsPort()), nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
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

func getCounterValue(metricFamily *io_prometheus_client.MetricFamily) float64 {
	if metricFamily.GetType() != io_prometheus_client.MetricType_COUNTER {
		panic("getCounterValue called on non-counter metric")
	}

	var total float64
	for _, metric := range metricFamily.GetMetric() {
		total += metric.GetCounter().GetValue()
	}
	return total
}
