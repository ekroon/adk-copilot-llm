package copilot

import (
	"testing"
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
