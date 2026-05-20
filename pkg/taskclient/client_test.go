package taskclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

import (
	"github.com/eagerod/rmq-http-bridge/pkg/types"
)

func TestClientSend(t *testing.T) {
	tests := []struct {
		name           string
		payload        types.TaskPayload
		statusCode     int
		responseBody   string
		expectError    bool
		validateServer func(t *testing.T, r *http.Request, p types.TaskPayload)
	}{
		{
			name: "successful request",
			payload: types.TaskPayload{
				Endpoint:     "https://example.com/webhook",
				Content:      "hello world",
				Base64Decode: false,
				Retries:      3,
				Headers: map[string]string{
					"Authorization": "Bearer token",
				},
				Backoff: 5,
				Timeout: 30,
			},
			statusCode:  http.StatusNoContent,
			expectError: false,
			validateServer: func(t *testing.T, r *http.Request, p types.TaskPayload) {
				if r.Method != http.MethodPost {
					t.Fatalf("expected POST request, got %s", r.Method)
				}

				if ct := r.Header.Get("Content-Type"); ct != "application/json" {
					t.Fatalf("expected application/json content type, got %s", ct)
				}

				var got types.TaskPayload
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Fatalf("failed to decode request body: %v", err)
				}

				if got.Endpoint != p.Endpoint {
					t.Fatalf("expected endpoint %s, got %s", p.Endpoint, got.Endpoint)
				}

				if got.Content != p.Content {
					t.Fatalf("expected content %s, got %s", p.Content, got.Content)
				}

				if got.Retries != p.Retries {
					t.Fatalf("expected retries %d, got %d", p.Retries, got.Retries)
				}
			},
		},
		{
			name: "server error",
			payload: types.TaskPayload{
				Endpoint: "https://example.com/webhook",
				Content:  "test",
			},
			statusCode:   http.StatusInternalServerError,
			responseBody: `internal server error`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.validateServer != nil {
					tt.validateServer(t, r, tt.payload)
				}

				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := New(server.URL)

			resp, err := client.Send(context.Background(), tt.payload)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if string(resp) != tt.responseBody {
				t.Fatalf(
					"expected response body %q, got %q",
					tt.responseBody,
					string(resp),
				)
			}
		})
	}
}
