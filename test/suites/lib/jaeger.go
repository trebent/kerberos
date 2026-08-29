package lib

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/jaegertracing/jaeger-idl/model/v1"
	tracingv2 "github.com/jaegertracing/jaeger-idl/proto-gen/api_v2"
	"google.golang.org/grpc"
)

// FindSpansByService finds spans for the given service name and returns them.
func FindSpansByService(conn *grpc.ClientConn, serviceName string, start time.Time, spanCount int, t testing.TB) []*model.Span {
	t.Logf("Service Name: %v", serviceName)

	begin := time.Now()
	defer func(b time.Time) {
		t.Logf("Took %s to find the spans", time.Since(b).String())
	}(begin)
	timeout := begin.Add(15 * time.Second)
	spans := make(map[string]*model.Span, 0)

	for {
		if time.Now().After(timeout) {
			t.Fatalf("Timed out waiting for %d spans", spanCount)
		}

		t.Log("Listing traces...")
		client := tracingv2.NewQueryServiceClient(conn)
		findTracesClient, err := client.FindTraces(t.Context(), &tracingv2.FindTracesRequest{
			Query: &tracingv2.TraceQueryParameters{
				ServiceName:  serviceName,
				StartTimeMin: start,
				StartTimeMax: time.Now(),
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

				spans[span.SpanID.String()] = &span

				if len(spans) == spanCount {
					spanSlice := make([]*model.Span, 0, 2)
					for _, span := range spans {
						spanSlice = append(spanSlice, span)
					}
					return spanSlice
				}
			}

			if errors.Is(err, io.EOF) {
				t.Logf("Got EOF, breaking")
				break
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
}

// FindEchoSpans finds spans for the echo service with the given trace ID and returns them.
func FindEchoSpans(conn *grpc.ClientConn, traceID model.TraceID, start time.Time, spanCount int, t testing.TB) []*model.Span {
	t.Logf("Trace ID: %v", traceID)

	begin := time.Now()
	defer func(b time.Time) {
		t.Logf("Took %s to find the spans", time.Since(b).String())
	}(begin)
	timeout := begin.Add(15 * time.Second)
	spans := make(map[string]*model.Span, 0)

	for {
		if time.Now().After(timeout) {
			t.Fatalf("Timed out waiting for %d spans", spanCount)
		}

		t.Log("Listing traces...")
		client := tracingv2.NewQueryServiceClient(conn)
		findTracesClient, err := client.FindTraces(t.Context(), &tracingv2.FindTracesRequest{
			Query: &tracingv2.TraceQueryParameters{
				ServiceName:  "echo",
				StartTimeMin: start,
				StartTimeMax: time.Now(),
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

					spans[span.SpanID.String()] = &span
				}

				if len(spans) == spanCount {
					spanSlice := make([]*model.Span, 0, 2)
					for _, span := range spans {
						spanSlice = append(spanSlice, span)
					}
					return spanSlice
				}
			}

			if errors.Is(err, io.EOF) {
				t.Logf("Got EOF, breaking")
				break
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func DecodeTraceParent(traceParent string, t testing.TB) model.TraceID {
	split := strings.Split(traceParent, "-")
	traceID, err := model.TraceIDFromString(split[1])
	if err != nil {
		t.Fatalf("Error decoding trace parent: %v", err)
	}
	return traceID
}
