package embedder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type embedRequest struct {
	Text string `json:"text"`
}
type embedResponse struct {
	Embedding []float32 `json:"embedding"`
}

func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
	embedderStartTime := time.Now()

	body, err := json.Marshal(embedRequest{Text: text})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedRequest: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/embed", bytes.NewReader(body)) // TODO: unsupported protocol scheme => fix the baseUrl

	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var er embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(er.Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding")
	}
	embedderReqTime := time.Since(embedderStartTime)
	log.Printf("embedding time: %v", embedderReqTime)

	return er.Embedding, nil
}
