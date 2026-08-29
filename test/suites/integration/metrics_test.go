package integration

import (
	"context"
	"fmt"
	"testing"

	adminapi "github.com/trebent/kerberos/test/client/admin"
	authbasicapi "github.com/trebent/kerberos/test/client/auth/basic"
	lib "github.com/trebent/kerberos/test/lib"
)

// Verifies that basic metrics are present and incremented as expected for GW calls.
func TestGWMetrics(t *testing.T) {
	startMetrics := lib.FetchMetrics(lib.GetHost(), lib.GetMetricsPort(), t)

	url := fmt.Sprintf("http://%s:%d/gw/backend/echo/hi", lib.GetHost(), lib.GetPort())
	_ = lib.Get(url, t)
	_ = lib.Put(url, []byte("metrics test"), t)
	_ = lib.Post(url, []byte("metrics test"), t)
	_ = lib.Delete(url, t)
	_ = lib.Patch(url, []byte("metrics test"), t)

	endMetrics := lib.FetchMetrics(lib.GetHost(), lib.GetMetricsPort(), t)
	for metricName, endMetric := range endMetrics {
		switch metricName {
		case "request_count_total":
			t.Log("Verifying request_count_total metric")

			startCount := float64(0)
			if startMetric, exists := startMetrics[metricName]; exists {
				startCount = lib.GetCounterValue(startMetric, t)
			}
			endCount := lib.GetCounterValue(endMetric, t)

			if endCount-startCount != 5.0 {
				t.Errorf("metric %s did not increment as expected: got %f, want %f", metricName, endCount-startCount, 5.0)
			}
		case "response_total":
			t.Log("Verifying response_total metric")

			startCount := float64(0)
			if startMetric, exists := startMetrics[metricName]; exists {
				startCount = lib.GetCounterValue(startMetric, t)
			}
			endCount := lib.GetCounterValue(endMetric, t)

			if endCount-startCount != 5.0 {
				t.Errorf("metric %s did not increment as expected: got %f, want %f", metricName, endCount-startCount, 5.0)
			}
		}
	}
}

// Verifies that basic metrics are present and incremented as expected for admin API calls.
func TestAdminMetrics(t *testing.T) {
	startMetrics := lib.FetchMetrics(lib.GetHost(), lib.GetMetricsPort(), t)

	_, _ = lib.AdminClient.LoginSuperuserWithResponse(
		context.Background(), adminapi.LoginSuperuserJSONRequestBody{
			ClientId:     lib.SuperUserClientID,
			ClientSecret: lib.SuperUserClientSecret,
		},
	)

	endMetrics := lib.FetchMetrics(lib.GetHost(), lib.GetMetricsPort(), t)
	for metricName, endMetric := range endMetrics {
		switch metricName {
		case "admin_request_count_total":
			t.Log("Verifying admin_request_count_total metric")

			startCount := float64(0)
			if startMetric, exists := startMetrics[metricName]; exists {
				startCount = lib.GetCounterValue(startMetric, t)
			}
			endCount := lib.GetCounterValue(endMetric, t)

			if endCount-startCount != 1.0 {
				t.Errorf("metric %s did not increment as expected: got %f, want %f", metricName, endCount-startCount, 5.0)
			}
		case "admin_response_total":
			t.Log("Verifying admin_response_total metric")

			startCount := float64(0)
			if startMetric, exists := startMetrics[metricName]; exists {
				startCount = lib.GetCounterValue(startMetric, t)
			}
			endCount := lib.GetCounterValue(endMetric, t)

			if endCount-startCount != 1.0 {
				t.Errorf("metric %s did not increment as expected: got %f, want %f", metricName, endCount-startCount, 5.0)
			}
		}
	}
}

// Verifies that basic metrics are present and incremented as expected for basic auth calls.
func TestBasicAuthMetrics(t *testing.T) {
	startMetrics := lib.FetchMetrics(lib.GetHost(), lib.GetMetricsPort(), t)

	_, _ = lib.BasicAuthClient.LoginWithResponse(t.Context(), authbasicapi.Orgid(alwaysOrgID), authbasicapi.LoginJSONRequestBody{
		Username: alwaysUser,
		Password: alwaysUserPassword,
	})

	endMetrics := lib.FetchMetrics(lib.GetHost(), lib.GetMetricsPort(), t)
	for metricName, endMetric := range endMetrics {
		switch metricName {
		case "admin_request_count_total":
			t.Log("Verifying admin_request_count_total metric")

			startCount := float64(0)
			if startMetric, exists := startMetrics[metricName]; exists {
				startCount = lib.GetCounterValue(startMetric, t)
			}
			endCount := lib.GetCounterValue(endMetric, t)

			if endCount-startCount != 1.0 {
				t.Errorf("metric %s did not increment as expected: got %f, want %f", metricName, endCount-startCount, 5.0)
			}
		case "admin_response_total":
			t.Log("Verifying admin_response_total metric")

			startCount := float64(0)
			if startMetric, exists := startMetrics[metricName]; exists {
				startCount = lib.GetCounterValue(startMetric, t)
			}
			endCount := lib.GetCounterValue(endMetric, t)

			if endCount-startCount != 1.0 {
				t.Errorf("metric %s did not increment as expected: got %f, want %f", metricName, endCount-startCount, 5.0)
			}
		}
	}
}
