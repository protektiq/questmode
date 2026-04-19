package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"questmode/backend/internal/story"
)

const (
	anthropicMessagesURL   = "https://api.anthropic.com/v1/messages"
	anthropicVersion       = "2023-06-01"
	chapterModel           = "claude-sonnet-4-20250514"
	chapterMaxTokens       = 1000
	chapterTimeout         = 5 * time.Second
	minAPIKeyLength        = 20
	maxAPIKeyLength        = 256
	maxAnthropicTextLength = 20000
)

var FALLBACK_CHAPTER = ChapterResponse{
	Chapter: "The path grew quiet as the friends paused, took a breath, and planned their next brave step together.",
	Questions: []Question{
		{
			Q:      "What did the friends do before moving on?",
			Answer: "They paused and made a plan.",
			Hint:   "Look for what they did during the quiet moment.",
		},
		{
			Q:      "How did the friends feel at the end?",
			Answer: "Brave and ready.",
			Hint:   "Think about the final words describing their mood.",
		},
	},
}

type ClaudeClient struct {
	httpClient *http.Client
	apiKey     string
}

type anthropicMessageRequest struct {
	Model     string                   `json:"model"`
	MaxTokens int                      `json:"max_tokens"`
	System    string                   `json:"system"`
	Messages  []anthropicRequestPrompt `json:"messages"`
}

type anthropicRequestPrompt struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicMessageResponse struct {
	Content []anthropicContentBlock `json:"content"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func NewClaudeClient(apiKey string) (*ClaudeClient, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, errors.New("api key is required")
	}
	if len(apiKey) < minAPIKeyLength || len(apiKey) > maxAPIKeyLength {
		return nil, fmt.Errorf("api key length must be between %d and %d", minAPIKeyLength, maxAPIKeyLength)
	}
	if !strings.HasPrefix(apiKey, "sk-") {
		return nil, errors.New("api key has invalid format")
	}

	return &ClaudeClient{
		httpClient: &http.Client{Timeout: chapterTimeout},
		apiKey:     apiKey,
	}, nil
}

func (c *ClaudeClient) GenerateChapter(ctx context.Context, state story.StoryState) (ChapterResponse, error) {
	if c == nil || c.httpClient == nil {
		return FALLBACK_CHAPTER, errors.New("claude client is not initialized")
	}

	payload := anthropicMessageRequest{
		Model:     chapterModel,
		MaxTokens: chapterMaxTokens,
		System:    buildChapterPrompt(state),
		Messages: []anthropicRequestPrompt{
			{Role: "user", Content: "generate"},
		},
	}

	rawText, err := c.requestAnthropicText(ctx, payload)
	if err != nil {
		return FALLBACK_CHAPTER, err
	}

	chapter, err := parseChapterResponse(rawText)
	if err != nil {
		return FALLBACK_CHAPTER, fmt.Errorf("parse chapter response: %w", err)
	}
	return chapter, nil
}

func (c *ClaudeClient) requestAnthropicText(ctx context.Context, payload anthropicMessageRequest) (string, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal anthropic request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicMessagesURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create anthropic request: %w", err)
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("content-type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if isTimeoutError(err) {
			return "", fmt.Errorf("anthropic request timed out: %w", err)
		}
		return "", fmt.Errorf("anthropic request failed: %w", err)
	}
	defer resp.Body.Close()

	rawResp, err := io.ReadAll(io.LimitReader(resp.Body, maxAnthropicTextLength))
	if err != nil {
		return "", fmt.Errorf("read anthropic response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("anthropic request returned status %d", resp.StatusCode)
	}

	var parsed anthropicMessageResponse
	if err := json.Unmarshal(rawResp, &parsed); err != nil {
		return "", fmt.Errorf("parse anthropic envelope: %w", err)
	}

	text := strings.TrimSpace(extractFirstTextBlock(parsed.Content))
	if text == "" {
		return "", errors.New("anthropic response missing text content")
	}
	return text, nil
}

func extractFirstTextBlock(blocks []anthropicContentBlock) string {
	for _, block := range blocks {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			return block.Text
		}
	}
	return ""
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
