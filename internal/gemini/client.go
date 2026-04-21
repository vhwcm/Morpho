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

	"github.com/vhwcm/Morpho/internal/logger"
)

var ErrMissingAPIKey = errors.New("gemini api key ausente")

type APIError struct {
	StatusCode int
	Body       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("erro Gemini (%d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("erro Gemini (%d): %s", e.StatusCode, e.Body)
}

func parseAPIError(statusCode int, body []byte) *APIError {
	apiErr := &APIError{
		StatusCode: statusCode,
		Body:       string(body),
	}

	type geminiErrorResponse struct {
		Error struct {
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}

	var parsed geminiErrorResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return apiErr
	}
	if parsed.Error.Message != "" {
		apiErr.Message = parsed.Error.Message
		if apiErr.StatusCode == 429 {
			apiErr.Message = "Cota de requisições excedida (Rate Limit). Tente novamente em alguns segundos."
		}
	}

	return apiErr
}

type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

type EmbeddingClient interface {
	Embed(ctx context.Context, text string) ([]float64, error)
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
		model = "gemini-2.5-flash"
	}

	return &Client{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}, nil
}

type ChatMessage struct {
	Role         string // "user", "model", "function"
	Content      string
	FunctionName string
}

type Tool struct {
	FunctionDeclarations []FunctionDeclaration `json:"function_declarations"`
}

type FunctionDeclaration struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type FunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type FunctionResponse struct {
	Name     string      `json:"name"`
	Response interface{} `json:"response"`
}

type ChatResult struct {
	Message       string
	FunctionCalls []FunctionCall
}

func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	res, err := c.Chat(ctx, "", []ChatMessage{{Role: "user", Content: prompt}})
	if err != nil {
		return "", err
	}
	return res.Message, nil
}

func (c *Client) Chat(ctx context.Context, systemPrompt string, history []ChatMessage, tools ...Tool) (ChatResult, error) {
	type part struct {
		Text             string            `json:"text,omitempty"`
		FunctionCall     *FunctionCall     `json:"functionCall,omitempty"`
		FunctionResponse *FunctionResponse `json:"functionResponse,omitempty"`
	}
	type content struct {
		Role  string `json:"role,omitempty"`
		Parts []part `json:"parts"`
	}
	type requestBody struct {
		Contents          []content `json:"contents"`
		SystemInstruction *content  `json:"system_instruction,omitempty"`
		Tools             []Tool    `json:"tools,omitempty"`
	}

	body := requestBody{
		Tools: tools,
	}

	if systemPrompt != "" {
		body.SystemInstruction = &content{
			Parts: []part{{Text: systemPrompt}},
		}
	}

	for _, msg := range history {
		role := msg.Role
		p := part{}

		if role == "function" {
			p.FunctionResponse = &FunctionResponse{
				Name: msg.FunctionName,
				Response: map[string]interface{}{
					"content": msg.Content,
				},
			}
		} else {
			if role == "" {
				role = "user"
			}
			p.Text = msg.Content
		}

		body.Contents = append(body.Contents, content{
			Role:  role,
			Parts: []part{p},
		})
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return ChatResult{}, err
	}

	u := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		url.PathEscape(c.model),
		url.QueryEscape(c.apiKey),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return ChatResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	logger.Debug("Iniciando chamada para Gemini API", map[string]interface{}{
		"model":       c.model,
		"tools_count": len(tools),
	})

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logger.Error("Erro na requisição Gemini", err)
		return ChatResult{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ChatResult{}, err
	}

	if resp.StatusCode >= 300 {
		apiErr := parseAPIError(resp.StatusCode, respBody)
		logger.Error("Gemini API retornou erro", apiErr)
		return ChatResult{}, apiErr
	}

	type responseBody struct {
		Candidates []struct {
			Content struct {
				Role  string `json:"role"`
				Parts []part `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	var parsed responseBody
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return ChatResult{}, err
	}

	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		err := errors.New("resposta Gemini sem conteúdo")
		logger.Error("Resposta inválida do Gemini", err)
		return ChatResult{}, err
	}

	candidate := parsed.Candidates[0]
	result := ChatResult{}

	for _, p := range candidate.Content.Parts {
		if p.FunctionCall != nil {
			result.FunctionCalls = append(result.FunctionCalls, *p.FunctionCall)
		} else if p.Text != "" {
			if result.Message != "" {
				result.Message += "\n"
			}
			result.Message += strings.TrimSpace(p.Text)
		}
	}

	logger.Debug("Resposta recebida do Gemini", map[string]interface{}{
		"has_text":        result.Message != "",
		"functions_count": len(result.FunctionCalls),
	})

	return result, nil
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
		return nil, parseAPIError(resp.StatusCode, respBody)
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

func (c *Client) Embed(ctx context.Context, text string) ([]float64, error) {
	type part struct {
		Text string `json:"text"`
	}
	type content struct {
		Parts []part `json:"parts"`
	}
	type requestBody struct {
		Content content `json:"content"`
	}

	payload, err := json.Marshal(requestBody{Content: content{Parts: []part{{Text: text}}}})
	if err != nil {
		return nil, err
	}

	embedModel := c.model
	if embedModel == "" {
		embedModel = "text-embedding-004"
	}
	if !strings.Contains(embedModel, "embedding") {
		embedModel = "text-embedding-004"
	}

	u := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:embedContent?key=%s",
		url.PathEscape(embedModel),
		url.QueryEscape(c.apiKey),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 300 {
		return nil, parseAPIError(resp.StatusCode, body)
	}

	type embedResp struct {
		Embedding struct {
			Values []float64 `json:"values"`
		} `json:"embedding"`
	}

	var parsed embedResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Embedding.Values) == 0 {
		return nil, errors.New("embedding vazio")
	}
	return parsed.Embedding.Values, nil
}
