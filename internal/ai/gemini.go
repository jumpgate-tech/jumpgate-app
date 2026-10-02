package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// geminiDefaultBase is Gemini's real API host, used when New's baseURL is
// "".
const geminiDefaultBase = "https://generativelanguage.googleapis.com"

type geminiProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func newGeminiProvider(apiKey, baseURL string) *geminiProvider {
	if baseURL == "" {
		baseURL = geminiDefaultBase
	}
	return &geminiProvider{apiKey: apiKey, baseURL: baseURL, client: newHTTPClient()}
}

func (p *geminiProvider) Name() string { return "gemini" }

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
}

func (p *geminiProvider) Explain(ctx context.Context, req ExplainRequest) (string, error) {
	prompt := buildPrompt(req)

	reqBody := geminiRequest{Contents: []geminiContent{{Parts: []geminiPart{{Text: prompt}}}}}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("gemini: encode request: %w", err)
	}

	// The key goes in a header, never the URL: a URL is quoted verbatim in every
	// *url.Error the transport returns, which would carry the key into logs and
	// API responses.
	endpoint := p.baseURL + "/v1beta/models/gemini-2.0-flash:generateContent"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return "", fmt.Errorf("gemini: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", p.apiKey)

	res, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("gemini: request failed: %w", err)
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("gemini: read response: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini: unexpected status %d: %s", res.StatusCode, string(respBody))
	}

	var parsed geminiResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("gemini: decode response: %w", err)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: response had no candidates")
	}
	return parsed.Candidates[0].Content.Parts[0].Text, nil
}
