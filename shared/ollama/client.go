package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/statherm/local-llm-examples/shared/types"
)

const defaultBaseURL = "http://localhost:11434"

// Client communicates with a local Ollama instance via its REST API.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient returns a Client pointing at the default Ollama address.
func NewClient() *Client {
	return &Client{
		BaseURL: defaultBaseURL,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// chatRequest is the JSON body sent to /api/chat.
type chatRequest struct {
	Model    string            `json:"model"`
	Messages []chatMessage     `json:"messages"`
	Stream   bool              `json:"stream"`
	Format   json.RawMessage   `json:"format,omitempty"`
	Options  map[string]any    `json:"options,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse is the JSON body returned by /api/chat (non-streaming).
type chatResponse struct {
	Model           string      `json:"model"`
	Message         chatMessage `json:"message"`
	TotalDuration   int64       `json:"total_duration"`   // nanoseconds
	LoadDuration    int64       `json:"load_duration"`    // nanoseconds
	PromptEvalCount int         `json:"prompt_eval_count"`
	EvalCount       int         `json:"eval_count"`
	EvalDuration    int64       `json:"eval_duration"` // nanoseconds
}

// ChatCompletion sends a chat request to Ollama and returns the response text
// along with performance metadata. If jsonMode is true, the model is asked to
// return valid JSON.
func (c *Client) ChatCompletion(model, system, prompt string, jsonMode bool) (string, types.ModelMetadata, error) {
	msgs := []chatMessage{}
	if system != "" {
		msgs = append(msgs, chatMessage{Role: "system", Content: system})
	}
	msgs = append(msgs, chatMessage{Role: "user", Content: prompt})

	req := chatRequest{
		Model:    model,
		Messages: msgs,
		Stream:   false,
	}
	if jsonMode {
		req.Format = json.RawMessage(`"json"`)
		// Cap output tokens to prevent repetition loops. Many small models
		// (qwen2.5:3b, phi3:mini, mistral:7b) generate thousands of tokens
		// of repeated JSON or chain-of-thought in JSON mode without this.
		// 256 tokens is plenty for any classification/structured-output response.
		req.Options = map[string]any{"num_predict": 256}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", types.ModelMetadata{}, fmt.Errorf("marshal request: %w", err)
	}

	start := time.Now()

	httpReq, err := http.NewRequest("POST", c.BaseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", types.ModelMetadata{}, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return "", types.ModelMetadata{}, fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", types.ModelMetadata{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", types.ModelMetadata{}, fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", types.ModelMetadata{}, fmt.Errorf("unmarshal response: %w", err)
	}

	totalTime := time.Since(start)
	ttft := time.Duration(chatResp.TotalDuration-chatResp.EvalDuration) * time.Nanosecond

	var tokPerSec float64
	if chatResp.EvalDuration > 0 {
		tokPerSec = float64(chatResp.EvalCount) / (float64(chatResp.EvalDuration) / 1e9)
	}

	meta := types.ModelMetadata{
		Model:        chatResp.Model,
		TokensIn:     chatResp.PromptEvalCount,
		TokensOut:    chatResp.EvalCount,
		TTFT:         ttft,
		TotalTime:    totalTime,
		TokensPerSec: tokPerSec,
	}

	return chatResp.Message.Content, meta, nil
}
