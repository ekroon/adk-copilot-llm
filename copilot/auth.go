package copilot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// DeviceCodeResponse represents the response from the device code endpoint.
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// AccessTokenResponse represents the response from the access token endpoint.
type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error,omitempty"`
}

// AuthConfig holds configuration for authentication.
type AuthConfig struct {
	// EnterpriseURL is the optional GitHub Enterprise URL.
	EnterpriseURL string
	// HTTPClient is an optional custom HTTP client.
	HTTPClient *http.Client
}

// Authenticator handles GitHub Copilot authentication.
type Authenticator struct {
	deviceCodeURL string
	accessURL     string
	httpClient    *http.Client
}

// NewAuthenticator creates a new Authenticator with the given configuration.
func NewAuthenticator(cfg AuthConfig) *Authenticator {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	auth := &Authenticator{
		httpClient: cfg.HTTPClient,
	}

	if cfg.EnterpriseURL != "" {
		domain := normalizeDomain(cfg.EnterpriseURL)
		auth.deviceCodeURL = fmt.Sprintf("https://%s/login/device/code", domain)
		auth.accessURL = fmt.Sprintf("https://%s/login/oauth/access_token", domain)
	} else {
		auth.deviceCodeURL = defaultDeviceCodeURL
		auth.accessURL = defaultAccessTokenURL
	}

	return auth
}

// StartDeviceFlow initiates the device authorization flow.
func (a *Authenticator) StartDeviceFlow(ctx context.Context) (*DeviceCodeResponse, error) {
	slog.Debug("Starting device flow authentication", "url", a.deviceCodeURL)
	
	reqBody := map[string]string{
		"client_id": copilotClientID,
		"scope":     "read:user",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		slog.Error("Failed to marshal device flow request", "error", err)
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.deviceCodeURL, strings.NewReader(string(body)))
	if err != nil {
		slog.Error("Failed to create device flow request", "error", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GitHubCopilotChat/0.35.0")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		slog.Error("Failed to send device flow request", "error", err)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("Device code request failed", "status", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("device code request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var deviceResp DeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&deviceResp); err != nil {
		slog.Error("Failed to decode device flow response", "error", err)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	slog.Info("Device flow started successfully",
		"verification_uri", deviceResp.VerificationURI,
		"user_code", deviceResp.UserCode,
		"expires_in", deviceResp.ExpiresIn,
		"interval", deviceResp.Interval)

	return &deviceResp, nil
}

// PollForAccessToken polls the access token endpoint until authorization is complete.
func (a *Authenticator) PollForAccessToken(ctx context.Context, deviceCode string, interval int) (string, error) {
	currentInterval := time.Duration(interval) * time.Second
	ticker := time.NewTicker(currentInterval)
	defer ticker.Stop()

	slog.Info("Starting to poll for access token", "initial_interval_seconds", interval)

	for {
		select {
		case <-ctx.Done():
			slog.Warn("Context cancelled while polling for access token", "error", ctx.Err())
			return "", ctx.Err()
		case <-ticker.C:
			slog.Debug("Checking access token status")
			token, err := a.checkAccessToken(ctx, deviceCode)
			if err != nil {
				// Check if it's a pending error
				if strings.Contains(err.Error(), "authorization_pending") {
					slog.Debug("Authorization still pending, continuing to poll")
					continue
				}
				// Check if we're polling too fast
				if strings.Contains(err.Error(), "slow_down") {
					// Increase the interval by 5 seconds as per OAuth spec
					currentInterval += 5 * time.Second
					ticker.Reset(currentInterval)
					slog.Warn("Received slow_down error, increasing polling interval",
						"new_interval_seconds", currentInterval.Seconds())
					continue
				}
				slog.Error("Failed to check access token", "error", err)
				return "", err
			}
			if token != "" {
				slog.Info("Successfully obtained access token")
				return token, nil
			}
		}
	}
}

// checkAccessToken checks if the access token is available.
func (a *Authenticator) checkAccessToken(ctx context.Context, deviceCode string) (string, error) {
	reqBody := map[string]string{
		"client_id":   copilotClientID,
		"device_code": deviceCode,
		"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		slog.Error("Failed to marshal access token request", "error", err)
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.accessURL, strings.NewReader(string(body)))
	if err != nil {
		slog.Error("Failed to create access token request", "error", err)
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GitHubCopilotChat/0.35.0")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		slog.Error("Failed to send access token request", "error", err)
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Debug("Access token request returned non-OK status", "status", resp.StatusCode, "body", string(body))
		return "", fmt.Errorf("access token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp AccessTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		slog.Error("Failed to decode access token response", "error", err)
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if tokenResp.Error != "" {
		slog.Debug("Access token response contains error", "error", tokenResp.Error)
		return "", fmt.Errorf("%s", tokenResp.Error)
	}

	return tokenResp.AccessToken, nil
}

// Authenticate performs the complete device flow authentication.
// It returns the access token and prints instructions for the user.
func (a *Authenticator) Authenticate(ctx context.Context) (string, error) {
	slog.Info("Starting GitHub Copilot authentication")
	
	// Start device flow
	deviceResp, err := a.StartDeviceFlow(ctx)
	if err != nil {
		slog.Error("Failed to start device flow", "error", err)
		return "", fmt.Errorf("failed to start device flow: %w", err)
	}

	fmt.Printf("\nTo authenticate with GitHub Copilot:\n")
	fmt.Printf("1. Visit: %s\n", deviceResp.VerificationURI)
	fmt.Printf("2. Enter code: %s\n\n", deviceResp.UserCode)
	fmt.Printf("Waiting for authorization...\n")

	// Poll for access token
	token, err := a.PollForAccessToken(ctx, deviceResp.DeviceCode, deviceResp.Interval)
	if err != nil {
		slog.Error("Failed to get access token", "error", err)
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	fmt.Printf("Successfully authenticated!\n")
	slog.Info("Authentication completed successfully")
	return token, nil
}
