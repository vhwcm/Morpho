package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrMissingAPIKey = errors.New("gemini api key ausente")

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("erro Gemini (%d): %s", e.StatusCode, e.Body)
}

type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

type ModelInfo struct {
	Name                       string   `json:"name"`
	DisplayName                string   `json:"displayName"`
	Description                string   `json:"description"`
	SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
}

func (m ModelInfo) SupportsGenerateContent() bool {
	for _, method := range m.SupportedGenerationMethods {
		if method == "generateContent" {
			return true
		}
	}
	return false
}

func NewClient(apiKey, model string) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, ErrMissingAPIKey
	}
	if strings.TrimSpace(model) == "" {
		model = "gemini-2.0-flash"
	}

	return &Client{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}, nil
}

func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	type requestBody struct {
		Contents []struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"contents"`
	}

	body := requestBody{}
	body.Contents = append(body.Contents, struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	}{
		Parts: []struct {
			Text string `json:"text"`
		}{{Text: prompt}},
	})

	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	u := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		url.PathEscape(c.model),
		url.QueryEscape(c.apiKey),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 300 {
		return "", &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	type responseBody struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	var parsed responseBody
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", err
	}

	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("resposta Gemini sem conteúdo")
	}

	return strings.TrimSpace(parsed.Candidates[0].Content.Parts[0].Text), nil
}

func (c *Client) ListModels(ctx context.Context) ([]ModelInfo, error) {
	u := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models?key=%s",
		url.QueryEscape(c.apiKey),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 300 {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	type listModelsResponse struct {
		Models []ModelInfo `json:"models"`
	}

	var parsed listModelsResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}

	return parsed.Models, nil
}
