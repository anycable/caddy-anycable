package caddy_anycable

import (
	"fmt"
	"github.com/anycable/anycable-go/config"
	"github.com/anycable/anycable-go/sse"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUnmarshalCaddyfile(t *testing.T) {
	tests := []struct {
		input        string
		expectErr    bool
		expectedOpts []string
		expectedWS   bool
		expectedSSE  bool
	}{
		{
			input: `anycable {
                log_level debug
                redis_url redis://localhost:6379/5
                ws_skip_url_checker true
                sse_skip_url_checker false
            }`,
			expectErr:    false,
			expectedOpts: []string{"--log_level=debug", "--redis_url=redis://localhost:6379/5"},
			expectedWS:   true,
			expectedSSE:  false,
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

			if !tc.expectErr {
				if len(h.Options) != len(tc.expectedOpts) {
					t.Fatalf("Expected %d options, got %d", len(tc.expectedOpts), len(h.Options))
				}

				for j, opt := range h.Options {
					if opt != tc.expectedOpts[j] {
						t.Errorf("Expected option %d to be '%s', but got '%s'", j, tc.expectedOpts[j], opt)
					}
				}

				if wsSkipUrlChecker != tc.expectedWS {
					t.Errorf("Expected wsSkipUrlChecker to be '%v', but got '%v'", tc.expectedWS, wsSkipUrlChecker)
				}

				if sseSkipUrlChecker != tc.expectedSSE {
					t.Errorf("Expected sseSkipUrlChecker to be '%v', but got '%v'", tc.expectedSSE, sseSkipUrlChecker)
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

	sseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("SSE handler response"))
	})

	handler := &AnyCableHandler{
		wsHandler:  websocketHandler,
		sseHandler: sseHandler,
		config:     &config.Config{Path: []string{"/cable"}, SSE: sse.Config{Enabled: true, Path: "/events"}},
	}

	tests := []struct {
		name         string
		path         string
		expectedBody string
		expectedCode int
		expectedNext bool
		skipWSCheck  bool
		skipSSECheck bool
	}{
		{
			name:         "WebSocket path",
			path:         "/cable",
			expectedBody: "WebSocket handler response",
			expectedCode: http.StatusOK,
			expectedNext: false,
		},
		{
			name:         "SSE path",
			path:         "/events",
			expectedBody: "SSE handler response",
			expectedCode: http.StatusOK,
			expectedNext: false,
		},
		{
			name:         "Non-WebSocket path",
			path:         "/not-cable",
			expectedBody: "",
			expectedCode: http.StatusOK,
			expectedNext: true,
		},
		{
			name:         "WebSocket path",
			path:         "/different_url",
			expectedBody: "WebSocket handler response",
			expectedCode: http.StatusOK,
			expectedNext: false,
			skipWSCheck:  true,
		},
		{
			name:         "SSE path",
			path:         "/different_sse_url",
			expectedBody: "SSE handler response",
			expectedCode: http.StatusOK,
			expectedNext: false,
			skipSSECheck: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wsSkipUrlChecker = tc.skipWSCheck
			sseSkipUrlChecker = tc.skipSSECheck

			recorder := httptest.NewRecorder()
			request, _ := http.NewRequest("GET", tc.path, nil)

			nextCalled := false
			nextHandler := caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
				nextCalled = true
				return nil
			})

			err := handler.ServeHTTP(recorder, request, nextHandler)
			if err != nil {
				t.Errorf("ServeHTTP failed: %v", err)
			}

			if tc.expectedNext != nextCalled {
				t.Errorf("Expected next handler to be called: %v, but got: %v", tc.expectedNext, nextCalled)
			}

			if recorder.Code != tc.expectedCode {
				t.Errorf("Expected status code to be %d, but got %d", tc.expectedCode, recorder.Code)
			}

			if recorder.Body.String() != tc.expectedBody {
				t.Errorf("Expected body to be '%s', but got '%s'", tc.expectedBody, recorder.Body.String())
			}
		})
	}
}
