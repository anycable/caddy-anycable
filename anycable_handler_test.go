package caddy_anycable

import (
	"fmt"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUnmarshalCaddyfile(t *testing.T) {
	tests := []struct {
		input     string
		expectErr bool
		expected  []string
	}{
		{
			input: `anycable {
                log_level debug
                redis_url redis://localhost:6379/5
            }`,
			expectErr: false,
			expected:  []string{"--log_level=debug", "--redis_url=redis://localhost:6379/5"},
		},
		{
			input: `anycable {
                log_level
            }`,
			expectErr: true,
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("case: %d", i), func(t *testing.T) {
			h := AnyCableHandler{}
			disp := caddyfile.NewTestDispenser(tc.input)
			err := h.UnmarshalCaddyfile(disp)

			if (err != nil) != tc.expectErr {
				t.Errorf("Test case %d: Expected error: %v, but got: %v", i, tc.expectErr, err != nil)
			}

			if !tc.expectErr && len(h.Options) != len(tc.expected) {
				t.Fatalf("Expected %d options, got %d", len(tc.expected), len(h.Options))
			}

			for j, opt := range h.Options {
				if opt != tc.expected[j] {
					t.Errorf("Expected option %d to be '%s', but got '%s'", j, tc.expected[j], opt)
				}
			}
		})
	}
}

func TestServeHTTP(t *testing.T) {
	websocketHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("WebSocket handler response"))
	})

	handler := &AnyCableHandler{
		handler: websocketHandler,
	}

	t.Run("WebSocket path", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, _ := http.NewRequest("GET", "/cable", nil)

		nextCalled := false
		nextHandler := caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			nextCalled = true
			return nil
		})

		if err := handler.ServeHTTP(recorder, request, nextHandler); err != nil {
			t.Errorf("ServeHTTP failed: %v", err)
		}

		if nextCalled {
			t.Errorf("Next handler was called for WebSocket path")
		}
	})

	t.Run("Non-WebSocket path", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		request, _ := http.NewRequest("GET", "/not-cable", nil)

		nextCalled := false
		nextHandler := caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			nextCalled = true
			return nil
		})

		if err := handler.ServeHTTP(recorder, request, nextHandler); err != nil {
			t.Errorf("ServeHTTP failed: %v", err)
		}

		if !nextCalled {
			t.Errorf("Next handler was not called for non-WebSocket path")
		}
	})
}
