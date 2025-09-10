package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaClient provides a client for the Ollama API
type OllamaClient struct {
	BaseURL      string
	HTTPClient   *http.Client
	DefaultModel string
}

// GenerateRequest represents a request to the Ollama API for text generation
type GenerateRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	System  string                 `json:"system,omitempty"`
	Context []int                  `json:"context,omitempty"`
	Stream  bool                   `json:"stream"`
	Raw     bool                   `json:"raw,omitempty"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// GenerateResponse represents a response from the Ollama API for text generation
type GenerateResponse struct {
	Model         string `json:"model"`
	Response      string `json:"response"`
	Context       []int  `json:"context,omitempty"`
	Done          bool   `json:"done"`
	TotalDuration int64  `json:"total_duration,omitempty"`
	Error         string `json:"error,omitempty"`
}

// ChatMessage represents a single message in a chat conversation
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents a request to the Ollama API for chat
type ChatRequest struct {
	Model    string                 `json:"model"`
	Messages []ChatMessage          `json:"messages"`
	Stream   bool                   `json:"stream"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

// ChatResponse represents a response from the Ollama API for chat
type ChatResponse struct {
	Model   string      `json:"model"`
	Message ChatMessage `json:"message"`
	Done    bool        `json:"done"`
	Error   string      `json:"error,omitempty"`
}

// StreamHandler is a function that handles streaming responses
type StreamHandler func(response interface{})

// NewClient creates a new OllamaClient with the given base URL
func NewClient(baseURL string, defaultModel string) *OllamaClient {
	return &OllamaClient{
		BaseURL:      baseURL,
		HTTPClient:   &http.Client{Timeout: time.Second * 60},
		DefaultModel: defaultModel,
	}
}

// Generate sends a prompt to the Ollama API and returns the generated text
func (c *OllamaClient) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	if req.Model == "" {
		req.Model = c.DefaultModel
	}
	req.Stream = false

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/generate", c.BaseURL), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("received non-OK response: %s, body: %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var genResp GenerateResponse
	if err := json.Unmarshal(body, &genResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	if genResp.Error != "" {
		return nil, fmt.Errorf("API error: %s", genResp.Error)
	}
	return &genResp, nil
}

// GenerateStream sends a prompt to the Ollama API and streams the responses
func (c *OllamaClient) GenerateStream(ctx context.Context, req *GenerateRequest, handler StreamHandler) error {
	if req.Model == "" {
		req.Model = c.DefaultModel
	}
	req.Stream = true

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/generate", c.BaseURL), bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("received non-OK response: %s, body: %s", resp.Status, string(body))
	}

	decoder := json.NewDecoder(resp.Body)
	for {
		var genResp GenerateResponse
		if err := decoder.Decode(&genResp); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			break
		}
		if genResp.Error != "" {
			return fmt.Errorf("API error: %s", genResp.Error)
		}
		handler(&genResp)
		if genResp.Done {
			break
		}
	}
	return nil
}

// Chat sends a chat request to the Ollama API
func (c *OllamaClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		req.Model = c.DefaultModel
	}
	req.Stream = false

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/chat", c.BaseURL), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("received non-OK response: %s, body: %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	if chatResp.Error != "" {
		return nil, fmt.Errorf("API error: %s", chatResp.Error)
	}
	return &chatResp, nil
}

// ChatStream sends a chat request to the Ollama API and streams the responses
func (c *OllamaClient) ChatStream(ctx context.Context, req *ChatRequest, handler StreamHandler) error {
	if req.Model == "" {
		req.Model = c.DefaultModel
	}
	req.Stream = true

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/chat", c.BaseURL), bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("received non-OK response: %s, body: %s", resp.Status, string(body))
	}

	decoder := json.NewDecoder(resp.Body)
	for {
		var chatResp ChatResponse
		if err := decoder.Decode(&chatResp); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			break
		}
		if chatResp.Error != "" {
			return fmt.Errorf("API error: %s", chatResp.Error)
		}
		handler(&chatResp)
		if chatResp.Done {
			break
		}
	}
	return nil
}

// ListModels lists all available models
func (c *OllamaClient) ListModels(ctx context.Context) ([]string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/tags", c.BaseURL), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("received non-OK response: %s, body: %s", resp.Status, string(body))
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var models []string
	for _, m := range result.Models {
		models = append(models, m.Name)
	}
	return models, nil
}
