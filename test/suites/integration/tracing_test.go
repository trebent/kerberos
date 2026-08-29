package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	authbasicapi "github.com/trebent/kerberos/test/client/auth/basic"
	lib "github.com/trebent/kerberos/test/lib"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Verifies that basic tracing works as expected. Verify both krb and echo services are found.
func TestGWTracing(t *testing.T) {
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

	echoSpans := lib.FindEchoSpans(conn, lib.DecodeTraceParent(traceParent[0], t), start, 2, t)
	for _, span := range echoSpans {
		t.Logf("Found 'echo' span %s belonging to trace %s", span.SpanID, span.TraceID)
	}

	krbSpans := lib.FindSpansByService(conn, "krb", start, 2, t)
	for _, span := range krbSpans {
		t.Logf("Found 'krb' span %s belonging to trace %s", span.SpanID, span.TraceID)
	}
}

func TestAdminAPITracing(t *testing.T) {
	start := time.Now()
	_ = lib.SuperLogin(t)

	conn, err := grpc.NewClient(fmt.Sprintf("localhost:%d", lib.GetJaegerAPIPort()), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to jaeger: %v", err)
	}
	defer conn.Close()

	krbSpans := lib.FindSpansByService(conn, "krb", start, 1, t)
	for _, span := range krbSpans {
		t.Logf("Found 'krb' span %s belonging to trace %s", span.SpanID, span.TraceID)
	}
}

func TestBasicAuthTracing(t *testing.T) {
	start := time.Now()

	resp, err := lib.BasicAuthClient.LoginWithResponse(t.Context(), authbasicapi.Orgid(alwaysOrgID), authbasicapi.LoginJSONRequestBody{
		Username: alwaysUser,
		Password: alwaysUserPassword,
	})
	lib.CheckErr(err, t)
	lib.VerifyStatusCode(resp.StatusCode(), http.StatusNoContent, t)

	conn, err := grpc.NewClient(fmt.Sprintf("localhost:%d", lib.GetJaegerAPIPort()), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to jaeger: %v", err)
	}
	defer conn.Close()

	krbSpans := lib.FindSpansByService(conn, "krb", start, 1, t)
	for _, span := range krbSpans {
		t.Logf("Found 'krb' span %s belonging to trace %s", span.SpanID, span.TraceID)
	}
}
