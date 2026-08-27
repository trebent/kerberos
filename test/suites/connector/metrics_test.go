package connector

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/trebent/kerberos/test/lib"
)

func TestMetrics(t *testing.T) {
	startMetrics := lib.FetchMetrics(lib.GetHost(), lib.GetConnectorMetricsPort(), t)

	cookie := loginAndGetSessionCookie(t)
	if cookie == nil {
		t.Fatal("Expected session cookie to be set, but it was nil")
	}

	// 1 request without a cookie set.
	resp, err := lib.Client.Get(fmt.Sprintf("http://%s:%d/hi", lib.GetHost(), lib.GetConnectorPort()))
	lib.CheckErr(err, t)
	_ = resp.Body.Close()
	lib.VerifyStatusCode(resp.StatusCode, http.StatusUnauthorized, t)

	// 1 request with a valid cookie set.
	req := &http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("%s:%d", lib.GetHost(), lib.GetConnectorPort()),
		},
		Header: make(http.Header),
	}
	req.AddCookie(cookie)

	respOK, err := lib.Client.Do(req)
	lib.CheckErr(err, t)
	_ = resp.Body.Close()
	lib.VerifyStatusCode(respOK.StatusCode, http.StatusOK, t)

	// 1 request with an invalid cookie set.
	reqInvalid := &http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("%s:%d", lib.GetHost(), lib.GetConnectorPort()),
		},
		Header: make(http.Header),
	}
	reqInvalid.AddCookie(&http.Cookie{
		Name:  "session",
		Value: "invalid-session-value",
	})

	respInvalid, err := lib.Client.Do(reqInvalid)
	lib.CheckErr(err, t)
	_ = respInvalid.Body.Close()
	lib.VerifyStatusCode(respInvalid.StatusCode, http.StatusUnauthorized, t)

	endMetrics := lib.FetchMetrics(lib.GetHost(), lib.GetConnectorMetricsPort(), t)

	for metricName, endMetric := range endMetrics {
		switch metricName {
		case "admin_connector_call_total":
			t.Log("Verifying admin_connector_call_total metric")
			if startMetric, exists := startMetrics[metricName]; exists {
				startCount := lib.GetCounterValue(startMetric, t)
				endCount := lib.GetCounterValue(endMetric, t)

				if endCount-startCount != 3.0 {
					t.Errorf("metric %s did not increment as expected: got %f, want %f", metricName, endCount-startCount, 3.0)
				}
			} else {
				t.Errorf("metric %s not found in start metrics", metricName)
			}
		case "admin_connector_call_denied_total":
			t.Log("Verifying admin_connector_call_denied_total metric")
			if startMetric, exists := startMetrics[metricName]; exists {
				startCount := lib.GetCounterValue(startMetric, t)
				endCount := lib.GetCounterValue(endMetric, t)

				if endCount-startCount != 2.0 {
					t.Errorf("metric %s did not increment as expected: got %f, want %f", metricName, endCount-startCount, 2.0)
				}
			} else {
				t.Errorf("metric %s not found in start metrics", metricName)
			}
		}
	}
}
