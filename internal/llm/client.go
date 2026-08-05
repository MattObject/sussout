package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Messages         []Message `json:"messages"`
	Model            string    `json:"model"`
	MaxTokens        int       `json:"max_tokens,omitempty"`
	IncludeReasoning bool      `json:"include_reasoning"`
}

func newChatRequest(messages []Message, model string) chatRequest {
	return chatRequest{
		Messages:         messages,
		Model:            model,
		MaxTokens:        4096,
		IncludeReasoning: false,
	}
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Role             string `json:"role"`
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"message"`
	} `json:"choices"`
}

type ClientConfig struct {
	BaseURL string
	Model   string
	APIKey  string
}

type LMStudioClient struct {
	config     ClientConfig
	httpClient *http.Client
}

func NewLMStudioClient(config ClientConfig) *LMStudioClient {
	return &LMStudioClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 300 * time.Second,
		},
	}
}

func (c *LMStudioClient) SendMessage(ctx context.Context, messages []Message) (string, error) {
	if c.config.Model == "" {
		return "", fmt.Errorf("model not set")
	}

	reqBody := newChatRequest(messages, c.config.Model)

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.apiURL("/v1/chat/completions"), bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	msg := result.Choices[0].Message
	content := msg.Content
	if content == "" {
		return "", fmt.Errorf("empty response from model")
	}

	return content, nil
}

type modelListResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

func (c *LMStudioClient) ListModels(ctx context.Context) ([]string, error) {
	return c.listModelsAt(ctx, []string{"/v1/models", "/models"})
}

func (c *LMStudioClient) apiURL(path string) string {
	base := strings.TrimRight(c.config.BaseURL, "/")
	if strings.HasSuffix(base, "/v1") {
		return base + strings.TrimPrefix(path, "/v1")
	}
	return base + path
}

func (c *LMStudioClient) listModelsAt(ctx context.Context, paths []string) ([]string, error) {
	var lastErr error
	for _, path := range paths {
		url := c.apiURL(path)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			lastErr = fmt.Errorf("create request: %w", err)
			continue
		}

		if c.config.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("list models: %w", err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("models API error %d", resp.StatusCode)
			continue
		}

		var result modelListResponse
		if err := json.Unmarshal(body, &result); err != nil {
			lastErr = fmt.Errorf("decode models: %w", err)
			continue
		}

		models := make([]string, len(result.Data))
		for i, m := range result.Data {
			models[i] = m.ID
		}
		return models, nil
	}
	return nil, lastErr
}

func (c *LMStudioClient) Detect() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	models, err := c.ListModels(ctx)
	if err != nil {
		return "", fmt.Errorf("model detection failed: %w", err)
	}

	if len(models) == 0 {
		return "", fmt.Errorf("no models found on server")
	}

	c.config.Model = models[0]
	return models[0], nil
}

func (c *LMStudioClient) Config() ClientConfig {
	return c.config
}

func (c *LMStudioClient) SetBaseURL(url string) {
	c.config.BaseURL = url
}

func (c *LMStudioClient) SetAPIKey(key string) {
	c.config.APIKey = key
}

func (c *LMStudioClient) SetModel(model string) {
	c.config.Model = model
}
