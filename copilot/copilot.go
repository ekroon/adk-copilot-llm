// Package copilot provides an implementation of the adk-go LLM interface
// for GitHub Copilot, including authentication and content generation.
package copilot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"strings"
	"sync"
	"time"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

const (
	// Default GitHub URLs
	defaultGitHubDomain     = "github.com"
	defaultBaseURL          = "https://api.githubcopilot.com"
	defaultCopilotAPIKeyURL = "https://api.github.com/copilot_internal/v2/token"
	defaultDeviceCodeURL    = "https://github.com/login/device/code"
	defaultAccessTokenURL   = "https://github.com/login/oauth/access_token"
	// copilotClientID is the OAuth client ID for GitHub Copilot.
	// This is the public client ID used by GitHub Copilot integrations,
	// as documented in the opencode-copilot-auth reference implementation:
	// https://github.com/sst/opencode-copilot-auth/blob/main/index.mjs
	copilotClientID            = "Iv1.b507a08c87ecfe98"
	copilotChatCompletionsPath = "/chat/completions"
)

// Config holds the configuration for the Copilot LLM.
type Config struct {
	// GitHubToken is the GitHub OAuth access token (refresh token in OAuth flow).
	GitHubToken string
	// EnterpriseURL is the optional GitHub Enterprise URL.
	EnterpriseURL string
	// Model is the model identifier to use (e.g., "gpt-4", "gpt-3.5-turbo").
	Model string
	// HTTPClient is an optional custom HTTP client.
	HTTPClient *http.Client
}

// CopilotLLM implements the model.LLM interface for GitHub Copilot.
type CopilotLLM struct {
	config        Config
	baseURL       string
	apiKeyURL     string
	deviceCodeURL string
	accessURL     string
	httpClient    *http.Client

	// Token management
	mu              sync.RWMutex
	copilotAPIKey   string
	apiKeyExpiresAt time.Time
}

// New creates a new CopilotLLM instance with the given configuration.
func New(cfg Config) (*CopilotLLM, error) {
	if cfg.GitHubToken == "" {
		return nil, fmt.Errorf("GitHubToken is required")
	}

	if cfg.Model == "" {
		cfg.Model = "gpt-4"
	}

	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{
			Timeout: 60 * time.Second,
		}
	}

	llm := &CopilotLLM{
		config:     cfg,
		httpClient: cfg.HTTPClient,
	}

	// Set up URLs based on whether enterprise URL is provided
	if cfg.EnterpriseURL != "" {
		domain := normalizeDomain(cfg.EnterpriseURL)
		llm.baseURL = fmt.Sprintf("https://copilot-api.%s", domain)
		llm.apiKeyURL = fmt.Sprintf("https://api.%s/copilot_internal/v2/token", domain)
		llm.deviceCodeURL = fmt.Sprintf("https://%s/login/device/code", domain)
		llm.accessURL = fmt.Sprintf("https://%s/login/oauth/access_token", domain)
	} else {
		llm.baseURL = defaultBaseURL
		llm.apiKeyURL = defaultCopilotAPIKeyURL
		llm.deviceCodeURL = defaultDeviceCodeURL
		llm.accessURL = defaultAccessTokenURL
	}

	return llm, nil
}

// Name returns the name of this LLM implementation.
func (c *CopilotLLM) Name() string {
	return "github-copilot"
}

// GenerateContent implements the model.LLM interface's GenerateContent method.
func (c *CopilotLLM) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		// Ensure we have a valid Copilot API key
		if err := c.ensureAPIKey(ctx); err != nil {
			yield(nil, fmt.Errorf("failed to get API key: %w", err))
			return
		}

		// Convert the request to OpenAI chat format
		chatReq, err := c.convertRequest(req)
		if err != nil {
			yield(nil, fmt.Errorf("failed to convert request: %w", err))
			return
		}

		chatReq.Stream = stream
		chatReq.Model = c.config.Model
		if req.Model != "" {
			chatReq.Model = req.Model
		}

		// Make the request
		if stream {
			c.generateStreamingContent(ctx, chatReq, yield)
		} else {
			c.generateNonStreamingContent(ctx, chatReq, yield)
		}
	}
}

// ensureAPIKey ensures we have a valid Copilot API key, refreshing if necessary.
func (c *CopilotLLM) ensureAPIKey(ctx context.Context) error {
	c.mu.RLock()
	if c.copilotAPIKey != "" && time.Now().Before(c.apiKeyExpiresAt) {
		c.mu.RUnlock()
		return nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if c.copilotAPIKey != "" && time.Now().Before(c.apiKeyExpiresAt) {
		return nil
	}

	// Fetch new API key
	req, err := http.NewRequestWithContext(ctx, "GET", c.apiKeyURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create API key request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.GitHubToken))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "GitHubCopilotChat/0.32.4")
	req.Header.Set("Editor-Version", "vscode/1.105.1")
	req.Header.Set("Editor-Plugin-Version", "copilot-chat/0.32.4")
	req.Header.Set("Copilot-Integration-Id", "vscode-chat")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch API key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to fetch API key: status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		Token     string `json:"token"`
		ExpiresAt int64  `json:"expires_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode API key response: %w", err)
	}

	c.copilotAPIKey = tokenResp.Token
	c.apiKeyExpiresAt = time.Unix(tokenResp.ExpiresAt, 0)

	return nil
}

// convertRequest converts an LLM request to OpenAI chat completion format.
func (c *CopilotLLM) convertRequest(req *model.LLMRequest) (*chatCompletionRequest, error) {
	chatReq := &chatCompletionRequest{
		Messages: make([]chatMessage, 0, len(req.Contents)),
	}

	// Convert genai.Content to chat messages
	for _, content := range req.Contents {
		msg := chatMessage{
			Role: strings.ToLower(content.Role),
		}

		// Convert parts to content
		if len(content.Parts) == 1 {
			// Single part - use string content
			part := content.Parts[0]
			if part.Text != "" {
				msg.Content = part.Text
			} else {
				// For other types, try to serialize
				data, err := json.Marshal(part)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal content part: %w", err)
				}
				msg.Content = string(data)
			}
		} else if len(content.Parts) > 1 {
			// Multiple parts - use array format
			parts := make([]map[string]interface{}, 0, len(content.Parts))
			for _, part := range content.Parts {
				if part.Text != "" {
					parts = append(parts, map[string]interface{}{
						"type": "text",
						"text": part.Text,
					})
				} else {
					// Handle other types like images, etc.
					data, err := json.Marshal(part)
					if err != nil {
						return nil, fmt.Errorf("failed to marshal content part: %w", err)
					}
					var partMap map[string]interface{}
					if err := json.Unmarshal(data, &partMap); err != nil {
						return nil, fmt.Errorf("failed to unmarshal content part: %w", err)
					}
					parts = append(parts, partMap)
				}
			}
			msg.ContentParts = parts
		}

		chatReq.Messages = append(chatReq.Messages, msg)
	}

	// Add configuration if present
	if req.Config != nil {
		if req.Config.Temperature != nil {
			temp := float64(*req.Config.Temperature)
			chatReq.Temperature = &temp
		}
		if req.Config.MaxOutputTokens != 0 {
			maxTokens := req.Config.MaxOutputTokens
			chatReq.MaxTokens = &maxTokens
		}
		if req.Config.TopP != nil {
			topP := float64(*req.Config.TopP)
			chatReq.TopP = &topP
		}
	}

	return chatReq, nil
}

// generateNonStreamingContent generates content without streaming.
func (c *CopilotLLM) generateNonStreamingContent(ctx context.Context, chatReq *chatCompletionRequest, yield func(*model.LLMResponse, error) bool) {
	reqBody, err := json.Marshal(chatReq)
	if err != nil {
		yield(nil, fmt.Errorf("failed to marshal request: %w", err))
		return
	}

	url := c.baseURL + copilotChatCompletionsPath
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqBody)))
	if err != nil {
		yield(nil, fmt.Errorf("failed to create request: %w", err))
		return
	}

	c.setRequestHeaders(req, false)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		yield(nil, fmt.Errorf("failed to send request: %w", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		yield(nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body)))
		return
	}

	var chatResp chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		yield(nil, fmt.Errorf("failed to decode response: %w", err))
		return
	}

	// Convert to LLMResponse
	llmResp := c.convertResponse(&chatResp, false)
	yield(llmResp, nil)
}

// generateStreamingContent generates content with streaming.
func (c *CopilotLLM) generateStreamingContent(ctx context.Context, chatReq *chatCompletionRequest, yield func(*model.LLMResponse, error) bool) {
	reqBody, err := json.Marshal(chatReq)
	if err != nil {
		yield(nil, fmt.Errorf("failed to marshal request: %w", err))
		return
	}

	url := c.baseURL + copilotChatCompletionsPath
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqBody)))
	if err != nil {
		yield(nil, fmt.Errorf("failed to create request: %w", err))
		return
	}

	c.setRequestHeaders(req, true)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		yield(nil, fmt.Errorf("failed to send request: %w", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		yield(nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body)))
		return
	}

	// Read SSE stream
	reader := newSSEReader(resp.Body)
	for {
		select {
		case <-ctx.Done():
			yield(nil, ctx.Err())
			return
		default:
		}

		line, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			yield(nil, fmt.Errorf("failed to read stream: %w", err))
			return
		}

		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk chatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// Skip malformed chunks
			continue
		}

		llmResp := c.convertChunk(&chunk)
		if !yield(llmResp, nil) {
			return
		}
	}

	// Send final completion marker
	yield(&model.LLMResponse{TurnComplete: true}, nil)
}

// setRequestHeaders sets the required headers for Copilot API requests.
func (c *CopilotLLM) setRequestHeaders(req *http.Request, stream bool) {
	c.mu.RLock()
	apiKey := c.copilotAPIKey
	c.mu.RUnlock()

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "GitHubCopilotChat/0.32.4")
	req.Header.Set("Editor-Version", "vscode/1.105.1")
	req.Header.Set("Editor-Plugin-Version", "copilot-chat/0.32.4")
	req.Header.Set("Copilot-Integration-Id", "vscode-chat")
	req.Header.Set("Openai-Intent", "conversation-panel")

	// Check if this is an agent call by looking at message roles
	// For now, default to "user" as X-Initiator
	req.Header.Set("X-Initiator", "user")

	if stream {
		req.Header.Set("Accept", "text/event-stream")
	}
}

// convertResponse converts an OpenAI chat completion response to LLMResponse.
func (c *CopilotLLM) convertResponse(resp *chatCompletionResponse, partial bool) *model.LLMResponse {
	llmResp := &model.LLMResponse{
		Partial:      partial,
		TurnComplete: !partial,
	}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		llmResp.Content = &genai.Content{
			Role:  choice.Message.Role,
			Parts: []*genai.Part{genai.NewPartFromText(choice.Message.Content)},
		}

		if choice.FinishReason != "" {
			llmResp.FinishReason = mapFinishReason(choice.FinishReason)
		}
	}

	if resp.Usage != nil {
		llmResp.UsageMetadata = &genai.GenerateContentResponseUsageMetadata{
			PromptTokenCount:     int32(resp.Usage.PromptTokens),
			CandidatesTokenCount: int32(resp.Usage.CompletionTokens),
			TotalTokenCount:      int32(resp.Usage.TotalTokens),
		}
	}

	return llmResp
}

// convertChunk converts a streaming chunk to LLMResponse.
func (c *CopilotLLM) convertChunk(chunk *chatCompletionChunk) *model.LLMResponse {
	llmResp := &model.LLMResponse{
		Partial:      true,
		TurnComplete: false,
	}

	if len(chunk.Choices) > 0 {
		choice := chunk.Choices[0]
		if choice.Delta.Content != "" {
			llmResp.Content = &genai.Content{
				Role:  "model",
				Parts: []*genai.Part{genai.NewPartFromText(choice.Delta.Content)},
			}
		}

		if choice.FinishReason != "" {
			llmResp.FinishReason = mapFinishReason(choice.FinishReason)
			llmResp.TurnComplete = true
			llmResp.Partial = false
		}
	}

	return llmResp
}

// mapFinishReason maps OpenAI finish reasons to genai.FinishReason.
func mapFinishReason(reason string) genai.FinishReason {
	switch reason {
	case "stop":
		return genai.FinishReasonStop
	case "length":
		return genai.FinishReasonMaxTokens
	case "content_filter":
		return genai.FinishReasonSafety
	default:
		return genai.FinishReasonOther
	}
}

// normalizeDomain normalizes a domain or URL to just the domain.
func normalizeDomain(url string) string {
	domain := strings.TrimPrefix(url, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimSuffix(domain, "/")
	return domain
}

// chatCompletionRequest represents an OpenAI chat completion request.
type chatCompletionRequest struct {
	Model        string        `json:"model"`
	Messages     []chatMessage `json:"messages"`
	Temperature  *float64      `json:"temperature,omitempty"`
	TopP         *float64      `json:"top_p,omitempty"`
	MaxTokens    *int32        `json:"max_tokens,omitempty"`
	Stream       bool          `json:"stream,omitempty"`
	N            *int          `json:"n,omitempty"`
	Stop         []string      `json:"stop,omitempty"`
	PresencePen  *float64      `json:"presence_penalty,omitempty"`
	FrequencyPen *float64      `json:"frequency_penalty,omitempty"`
}

// chatMessage represents a chat message.
type chatMessage struct {
	Role         string
	Content      string
	ContentParts []map[string]interface{}
}

// MarshalJSON implements custom JSON marshaling for chatMessage.
func (m chatMessage) MarshalJSON() ([]byte, error) {
	type Alias struct {
		Role    string      `json:"role"`
		Content interface{} `json:"content,omitempty"`
	}

	var content interface{}
	if len(m.ContentParts) > 0 {
		content = m.ContentParts
	} else if m.Content != "" {
		content = m.Content
	}

	return json.Marshal(&Alias{
		Role:    m.Role,
		Content: content,
	})
}

// chatCompletionResponse represents an OpenAI chat completion response.
type chatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []chatCompletionChoice `json:"choices"`
	Usage   *chatCompletionUsage   `json:"usage,omitempty"`
}

// chatCompletionChoice represents a choice in the response.
type chatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// chatCompletionUsage represents token usage information.
type chatCompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// chatCompletionChunk represents a streaming chunk.
type chatCompletionChunk struct {
	ID      string                      `json:"id"`
	Object  string                      `json:"object"`
	Created int64                       `json:"created"`
	Model   string                      `json:"model"`
	Choices []chatCompletionChunkChoice `json:"choices"`
}

// chatCompletionChunkChoice represents a choice in a streaming chunk.
type chatCompletionChunkChoice struct {
	Index        int       `json:"index"`
	Delta        chatDelta `json:"delta"`
	FinishReason string    `json:"finish_reason"`
}

// chatDelta represents the delta content in a streaming chunk.
type chatDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// sseReader reads Server-Sent Events from a stream.
type sseReader struct {
	reader io.Reader
	buffer []byte
}

// newSSEReader creates a new SSE reader.
func newSSEReader(r io.Reader) *sseReader {
	return &sseReader{
		reader: r,
		buffer: make([]byte, 0, 4096),
	}
}

// ReadLine reads the next line from the SSE stream.
func (r *sseReader) ReadLine() (string, error) {
	for {
		// Check if we have a complete line in the buffer
		if idx := bytes.IndexByte(r.buffer, '\n'); idx >= 0 {
			line := string(r.buffer[:idx])
			r.buffer = r.buffer[idx+1:]
			return strings.TrimSpace(line), nil
		}

		// Read more data
		buf := make([]byte, 1024)
		n, err := r.reader.Read(buf)
		if n > 0 {
			r.buffer = append(r.buffer, buf[:n]...)
		}
		if err != nil {
			if err == io.EOF && len(r.buffer) > 0 {
				// Return remaining buffer
				line := string(r.buffer)
				r.buffer = nil
				return strings.TrimSpace(line), nil
			}
			return "", err
		}
	}
}
