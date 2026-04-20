package gemini

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestNewClientValidationAndDefaults(t *testing.T) {
	if _, err := NewClient("", "gemini-2.0-flash"); !errors.Is(err, ErrMissingAPIKey) {
		t.Fatalf("esperava ErrMissingAPIKey, got=%v", err)
	}

	c, err := NewClient("api-key", "")
	if err != nil {
		t.Fatalf("erro ao criar client: %v", err)
	}
	if c.model != "gemini-2.0-flash" {
		t.Fatalf("modelo padrão inesperado: %s", c.model)
	}
}

func TestModelInfoSupportsGenerateContent(t *testing.T) {
	m := ModelInfo{SupportedGenerationMethods: []string{"countTokens", "generateContent"}}
	if !m.SupportsGenerateContent() {
		t.Fatalf("deveria detectar suporte a generateContent")
	}

	m = ModelInfo{SupportedGenerationMethods: []string{"embedContent"}}
	if m.SupportsGenerateContent() {
		t.Fatalf("não deveria detectar suporte indevido")
	}
}

func TestMockClientGenerate(t *testing.T) {
	m := NewMockClient()

	out, err := m.Generate(context.Background(), "Preciso de ajuda com backend em Go")
	if err != nil {
		t.Fatalf("mock backend não deveria falhar: %v", err)
	}
	if out == "" {
		t.Fatalf("mock backend deveria retornar texto")
	}

	out, err = m.Generate(context.Background(), "Faça review deste PR")
	if err != nil {
		t.Fatalf("mock review não deveria falhar: %v", err)
	}
	if out == "" {
		t.Fatalf("mock review deveria retornar texto")
	}

	out, err = m.Generate(context.Background(), "tarefa genérica")
	if err != nil {
		t.Fatalf("mock genérico não deveria falhar: %v", err)
	}
	if out == "" {
		t.Fatalf("mock genérico deveria retornar texto")
	}
}

func TestClientGenerateSuccess(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("método inesperado: %s", req.Method)
		}

		if ct := req.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("content-type inesperado: %s", ct)
		}

		q := req.URL.Query().Get("key")
		if q != "test-key" {
			t.Fatalf("query key inesperada: %s", q)
		}

		if !strings.Contains(req.URL.String(), "models/gemini-2.5-flash%20pro:generateContent") {
			t.Fatalf("URL deveria conter modelo escapado: %s", req.URL.String())
		}

		payload, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("erro ao ler payload: %v", err)
		}

		if !strings.Contains(string(payload), "planeje a implementação") {
			t.Fatalf("payload não contém prompt esperado: %s", string(payload))
		}

		return testHTTPResponse(http.StatusOK, `{"candidates":[{"content":{"parts":[{"text":"  resposta da IA  "}]}}]}`), nil
	})

	c := &Client{
		apiKey: "test-key",
		model:  "gemini-2.5-flash pro",
		httpClient: &http.Client{
			Transport: transport,
		},
	}

	out, err := c.Generate(context.Background(), "planeje a implementação")
	if err != nil {
		t.Fatalf("generate não deveria falhar: %v", err)
	}
	if out != "resposta da IA" {
		t.Fatalf("texto retornado deveria vir trimado, got=%q", out)
	}
}

func TestClientGenerateErrors(t *testing.T) {
	t.Run("api error propagates status and body", func(t *testing.T) {
		transport := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return testHTTPResponse(http.StatusTooManyRequests, `{"error":"rate limited"}`), nil
		})

		c := &Client{apiKey: "k", model: "gemini-2.0-flash", httpClient: &http.Client{Transport: transport}}

		_, err := c.Generate(context.Background(), "teste")
		if err == nil {
			t.Fatalf("esperava erro da API")
		}

		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("erro deveria ser APIError, got=%T", err)
		}
		if apiErr.StatusCode != http.StatusTooManyRequests {
			t.Fatalf("status inesperado: %d", apiErr.StatusCode)
		}
		if !strings.Contains(apiErr.Body, "rate limited") {
			t.Fatalf("body inesperado: %s", apiErr.Body)
		}
	})

	t.Run("response without candidates returns validation error", func(t *testing.T) {
		transport := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return testHTTPResponse(http.StatusOK, `{"candidates":[]}`), nil
		})

		c := &Client{apiKey: "k", model: "gemini-2.0-flash", httpClient: &http.Client{Transport: transport}}

		_, err := c.Generate(context.Background(), "teste")
		if err == nil {
			t.Fatalf("esperava erro para resposta sem conteúdo")
		}
		if !strings.Contains(strings.ToLower(err.Error()), "sem conteúdo") {
			t.Fatalf("mensagem de erro inesperada: %v", err)
		}
	})

	t.Run("transport error is returned", func(t *testing.T) {
		transport := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New("network down")
		})

		c := &Client{apiKey: "k", model: "gemini-2.0-flash", httpClient: &http.Client{Transport: transport}}

		_, err := c.Generate(context.Background(), "teste")
		if err == nil || !strings.Contains(err.Error(), "network down") {
			t.Fatalf("erro de transporte deveria propagar, got=%v", err)
		}
	})
}

func TestClientListModelsSuccessAndErrors(t *testing.T) {
	t.Run("list models success", func(t *testing.T) {
		transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodGet {
				t.Fatalf("método inesperado: %s", req.Method)
			}
			if req.URL.Query().Get("key") != "test-key" {
				t.Fatalf("query key inesperada: %s", req.URL.String())
			}

			escaped := url.QueryEscape("test-key")
			if !strings.Contains(req.URL.String(), "models?key="+escaped) {
				t.Fatalf("URL de listagem inesperada: %s", req.URL.String())
			}

			return testHTTPResponse(http.StatusOK, `{"models":[{"name":"models/gemini-2.0-flash","displayName":"Gemini Flash","supportedGenerationMethods":["generateContent"]}]}`), nil
		})

		c := &Client{apiKey: "test-key", model: "gemini-2.0-flash", httpClient: &http.Client{Transport: transport}}

		models, err := c.ListModels(context.Background())
		if err != nil {
			t.Fatalf("list models não deveria falhar: %v", err)
		}
		if len(models) != 1 {
			t.Fatalf("quantidade inesperada de modelos: %d", len(models))
		}
		if models[0].Name != "models/gemini-2.0-flash" || !models[0].SupportsGenerateContent() {
			t.Fatalf("modelo retornado inesperado: %+v", models[0])
		}
	})

	t.Run("list models api error", func(t *testing.T) {
		transport := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return testHTTPResponse(http.StatusUnauthorized, `{"error":"unauthorized"}`), nil
		})

		c := &Client{apiKey: "k", model: "gemini-2.0-flash", httpClient: &http.Client{Transport: transport}}

		_, err := c.ListModels(context.Background())
		if err == nil {
			t.Fatalf("esperava erro da API")
		}
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
			t.Fatalf("erro inesperado: %v", err)
		}
	})
}
