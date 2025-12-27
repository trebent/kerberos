// nolint
package main

import (
	"context"
	"net/http"

	obs "github.com/trebent/kerberos/internal/composer/observability"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func main() {
	shutdown, err := obs.Instrument(context.TODO(), "tracing-testing-client", "0.1.0")
	if err != nil {
		println("Error initializing OpenTelemetry:", err.Error())
		return
	}
	defer shutdown(context.Background())

	println("Running client")

	ctx, span := otel.Tracer("client").Start(context.Background(), "client-request")
	defer span.End()

	req, _ := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"http://localhost:15000/echo?message=Hello%20World%21&delay=1000",
		nil,
	)
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	c := http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		println("Error on GETing towards test server", err.Error())
	}
	resp.Body.Close()
	println("GET request completed with status code:", resp.StatusCode)
}
