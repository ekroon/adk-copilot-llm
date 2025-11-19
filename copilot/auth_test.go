package copilot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNormalizeDomain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "domain with https",
			input:    "https://company.ghe.com",
			expected: "company.ghe.com",
		},
		{
			name:     "domain with http",
			input:    "http://company.ghe.com",
			expected: "company.ghe.com",
		},
		{
			name:     "domain with trailing slash",
			input:    "company.ghe.com/",
			expected: "company.ghe.com",
		},
		{
			name:     "plain domain",
			input:    "company.ghe.com",
			expected: "company.ghe.com",
		},
		{
			name:     "full URL with trailing slash",
			input:    "https://company.ghe.com/",
			expected: "company.ghe.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeDomain(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeDomain(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNewAuthenticator(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		auth := NewAuthenticator(AuthConfig{})
		if auth.deviceCodeURL != defaultDeviceCodeURL {
			t.Errorf("expected deviceCodeURL to be %q, got %q", defaultDeviceCodeURL, auth.deviceCodeURL)
		}
		if auth.accessURL != defaultAccessTokenURL {
			t.Errorf("expected accessURL to be %q, got %q", defaultAccessTokenURL, auth.accessURL)
		}
	})

	t.Run("enterprise configuration", func(t *testing.T) {
		enterpriseURL := "company.ghe.com"
		auth := NewAuthenticator(AuthConfig{
			EnterpriseURL: enterpriseURL,
		})
		expectedDeviceURL := "https://company.ghe.com/login/device/code"
		expectedAccessURL := "https://company.ghe.com/login/oauth/access_token"

		if auth.deviceCodeURL != expectedDeviceURL {
			t.Errorf("expected deviceCodeURL to be %q, got %q", expectedDeviceURL, auth.deviceCodeURL)
		}
		if auth.accessURL != expectedAccessURL {
			t.Errorf("expected accessURL to be %q, got %q", expectedAccessURL, auth.accessURL)
		}
	})
}

func TestPollForAccessToken_SlowDownHandling(t *testing.T) {
	// Track the number of requests and timing
	requestCount := 0
	var lastRequestTime time.Time
	var intervalsBetweenRequests []time.Duration

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		now := time.Now()
		
		if requestCount > 1 {
			interval := now.Sub(lastRequestTime)
			intervalsBetweenRequests = append(intervalsBetweenRequests, interval)
		}
		lastRequestTime = now

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		var resp AccessTokenResponse
		
		// First two requests: return slow_down
		if requestCount <= 2 {
			resp.Error = "slow_down"
		} else if requestCount == 3 {
			// Third request: return authorization_pending
			resp.Error = "authorization_pending"
		} else {
			// Fourth request: return success
			resp.AccessToken = "test_token_12345"
			resp.TokenType = "bearer"
		}

		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	auth := &Authenticator{
		accessURL:  server.URL,
		httpClient: server.Client(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use a very short initial interval for testing (1 second)
	token, err := auth.PollForAccessToken(ctx, "test_device_code", 1)

	if err != nil {
		t.Fatalf("Expected successful authentication, got error: %v", err)
	}

	if token != "test_token_12345" {
		t.Errorf("Expected token 'test_token_12345', got '%s'", token)
	}

	// Should have made 4 requests total
	if requestCount != 4 {
		t.Errorf("Expected 4 requests, got %d", requestCount)
	}

	// After slow_down errors, intervals should increase by approximately 5 seconds
	// We check that each interval is at least close to expected value accounting for timing jitter
	if len(intervalsBetweenRequests) >= 2 {
		// First interval after first slow_down should be around 6 seconds (1 + 5)
		if intervalsBetweenRequests[0] < 5*time.Second || intervalsBetweenRequests[0] > 7*time.Second {
			t.Errorf("First interval after slow_down should be ~6s, got %v", intervalsBetweenRequests[0])
		}
		// Second interval after second slow_down should be around 11 seconds (1 + 5 + 5)
		if len(intervalsBetweenRequests) >= 2 && (intervalsBetweenRequests[1] < 10*time.Second || intervalsBetweenRequests[1] > 12*time.Second) {
			t.Errorf("Second interval after slow_down should be ~11s, got %v", intervalsBetweenRequests[1])
		}
	}
}

func TestPollForAccessToken_AuthorizationPending(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		var resp AccessTokenResponse
		
		// First two requests: return authorization_pending
		if requestCount <= 2 {
			resp.Error = "authorization_pending"
		} else {
			// Third request: return success
			resp.AccessToken = "test_token_67890"
			resp.TokenType = "bearer"
		}

		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	auth := &Authenticator{
		accessURL:  server.URL,
		httpClient: server.Client(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use a short interval for testing
	token, err := auth.PollForAccessToken(ctx, "test_device_code", 1)

	if err != nil {
		t.Fatalf("Expected successful authentication, got error: %v", err)
	}

	if token != "test_token_67890" {
		t.Errorf("Expected token 'test_token_67890', got '%s'", token)
	}

	// Should have made 3 requests
	if requestCount != 3 {
		t.Errorf("Expected 3 requests, got %d", requestCount)
	}
}

func TestCheckAccessToken_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedError string
	}{
		{
			name:          "slow_down error",
			statusCode:    http.StatusOK,
			responseBody:  `{"error": "slow_down"}`,
			expectedError: "slow_down",
		},
		{
			name:          "authorization_pending error",
			statusCode:    http.StatusOK,
			responseBody:  `{"error": "authorization_pending"}`,
			expectedError: "authorization_pending",
		},
		{
			name:          "non-OK status",
			statusCode:    http.StatusBadRequest,
			responseBody:  `{"error": "invalid_request"}`,
			expectedError: "access token request failed with status 400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			auth := &Authenticator{
				accessURL:  server.URL,
				httpClient: server.Client(),
			}

			ctx := context.Background()
			token, err := auth.checkAccessToken(ctx, "test_device_code")

			if err == nil {
				t.Errorf("Expected error containing '%s', got nil", tt.expectedError)
			} else if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
			}

			if token != "" {
				t.Errorf("Expected empty token on error, got '%s'", token)
			}
		})
	}
}
