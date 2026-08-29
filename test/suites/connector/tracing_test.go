package connector

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	lib "github.com/trebent/kerberos/test/lib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestTracing(t *testing.T) {
	start := time.Now()

	// 1 request without a cookie
	resp, err := lib.Client.Get(fmt.Sprintf("http://%s:%d/hi", lib.GetHost(), lib.GetConnectorPort()))
	lib.CheckErr(err, t)
	_ = resp.Body.Close()
	lib.VerifyStatusCode(resp.StatusCode, 401, t)

	// 1 request with a valid session cookie
	cookie := loginAndGetSessionCookie(t)
	if cookie == nil {
		t.Fatal("Expected session cookie to be set, but it was nil")
	}
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

	conn, err := grpc.NewClient(fmt.Sprintf("localhost:%d", lib.GetJaegerAPIPort()), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to jaeger: %v", err)
	}
	defer conn.Close()

	spans := lib.FindSpansByService(conn, "admin-connector", start, 2, t)
	for _, span := range spans {
		t.Logf("Found span %s belonging to trace %s", span.SpanID, span.TraceID)
	}
}
