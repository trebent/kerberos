package ft

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jaegertracing/jaeger-idl/model/v1"
	tracingv2 "github.com/jaegertracing/jaeger-idl/proto-gen/api_v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Verifies that basic tracing works as expected.
func TestTracing(t *testing.T) {
	start := time.Now()
	response := get(fmt.Sprintf("http://localhost:%d/gw/backend/echo/tracing-test", port), t)

	decodedResponse := verifyResponse(response, http.StatusOK, t)

	traceParent, exists := decodedResponse.Headers["Traceparent"]
	if !exists || len(traceParent) == 0 {
		t.Fatal("Missing Traceparent header in response")
	} else {
		t.Logf("Traceparent header: %s", traceParent[0])
	}

	conn, err := grpc.NewClient(fmt.Sprintf("localhost:%d", jaegerReadAPIPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to jaeger: %v", err)
	}
	defer conn.Close()

	spans := findSpans(conn, decodeTraceParent(traceParent[0], t), start, 2, t)
	for _, span := range spans {
		t.Logf("Found span %s belonging to trace %s", span.SpanID, span.TraceID)
	}
}

func findSpans(conn *grpc.ClientConn, traceID model.TraceID, start time.Time, spanCount int, t *testing.T) []*model.Span {
	t.Logf("Trace ID: %v", traceID)

	timeout := time.Now().Add(15 * time.Second)
	spans := make([]*model.Span, 0)

	for {
		if time.Now().After(timeout) {
			t.Fatalf("Timed out waiting for %d spans", spanCount)
		}

		t.Log("Listing traces...")
		client := tracingv2.NewQueryServiceClient(conn)
		findTracesClient, err := client.FindTraces(t.Context(), &tracingv2.FindTracesRequest{
			Query: &tracingv2.TraceQueryParameters{
				ServiceName: "echo",
			},
		})
		if err != nil {
			t.Fatalf("Failed to initialise get trace client: %v", err)
		}

		for {
			t.Log("Get span chunk...")
			chunk, err := findTracesClient.Recv()
			if err != nil && !errors.Is(err, io.EOF) {
				t.Fatalf("Error when receiving span chunk: %v", err)
			}

			for _, span := range chunk.GetSpans() {
				t.Logf("Inspecting span %v", span)

				if span.TraceID == traceID {
					t.Log("Found a matching trace ID")
					spans = append(spans, &span)
				}

				if len(spans) == spanCount {
					return spans
				}
			}

			if errors.Is(err, io.EOF) {
				t.Logf("Got EOF, breaking")
				break
			}
		}

		time.Sleep(3 * time.Second)
	}
}

func decodeTraceParent(traceParent string, t *testing.T) model.TraceID {
	split := strings.Split(traceParent, "-")
	traceID, err := model.TraceIDFromString(split[1])
	if err != nil {
		t.Fatalf("Error decoding trace parent: %v", err)
	}
	return traceID
}
