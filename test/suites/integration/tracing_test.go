package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	lib "github.com/trebent/kerberos/test/lib"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Verifies that basic tracing works as expected.
func TestTracingBasic(t *testing.T) {
	start := time.Now()
	response := lib.Get(fmt.Sprintf("http://%s:%d/gw/backend/echo/hi", lib.GetHost(), lib.GetPort()), t)

	decodedResponse := lib.VerifyGWResponse(response, http.StatusOK, t)

	traceParent, exists := decodedResponse.Headers["Traceparent"]
	if !exists || len(traceParent) == 0 {
		t.Fatal("Missing Traceparent header in response")
	} else {
		t.Logf("Traceparent header: %s", traceParent[0])
	}

	conn, err := grpc.NewClient(fmt.Sprintf("localhost:%d", lib.GetJaegerAPIPort()), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to jaeger: %v", err)
	}
	defer conn.Close()

	spans := lib.FindEchoSpans(conn, lib.DecodeTraceParent(traceParent[0], t), start, 2, t)
	for _, span := range spans {
		t.Logf("Found span %s belonging to trace %s", span.SpanID, span.TraceID)
	}
}
