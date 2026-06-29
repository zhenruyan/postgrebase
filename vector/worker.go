package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// GetEmbedding generates a vector embedding for the specified text using an OpenAI-compatible API.
func GetEmbedding(ctx context.Context, apiKey, baseUrl, modelId, text string) ([]float32, error) {
	if baseUrl == "" {
		baseUrl = "https://api.openai.com/v1"
	}
	if !strings.HasSuffix(baseUrl, "/") {
		baseUrl += "/"
	}
	url := baseUrl + "embeddings"

	reqBody, err := json.Marshal(map[string]any{
		"input": text,
		"model": modelId,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errData map[string]any
		json.NewDecoder(resp.Body).Decode(&errData)
		return nil, fmt.Errorf("embedding provider returned status %d: %v", resp.StatusCode, errData)
	}

	var respData struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return nil, err
	}

	if len(respData.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned in response")
	}

	return respData.Data[0].Embedding, nil
}
