// echo is a simple HTTP server that echoes back the request
// body and headers. It is used for testing purposes.
package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

type response struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Headers map[string][]string `json:"headers"`
	Body    []byte              `json:"body"`
}

var _ io.Writer = &response{}

// Write implements io.Writer.
func (r *response) Write(p []byte) (n int, err error) {
	r.Body = append(r.Body, p...)
	return len(p), nil
}

func main() {
	// Create a new HTTP server
	srv := &http.Server{
		Addr: ":8080",
	}

	// Register the echo handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		println(r.Method + " " + r.URL.String())
		w.Header().Set("Content-Type", "application/json")

		w.WriteHeader(http.StatusOK)

		resp := &response{
			Method:  r.Method,
			URL:     r.URL.String(),
			Headers: r.Header,
		}

		if r.Body != nil && r.Body != http.NoBody {
			defer r.Body.Close()
			// Read the request body
			_, err := io.Copy(resp, r.Body)
			if err != nil {
				http.Error(w, "{\"error\": \"failed to read request body\"}", http.StatusInternalServerError)
				return
			}
		}

		responseBytes, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			http.Error(w, "{\"error\": \"failed to marshal response\"}", http.StatusInternalServerError)
			return
		}

		w.Write(responseBytes)
	})

	// Start the server
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
