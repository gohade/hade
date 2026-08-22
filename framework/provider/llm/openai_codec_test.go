package llm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gohade/hade/framework"
	"github.com/gohade/hade/framework/contract"
	configprovider "github.com/gohade/hade/framework/provider/config"
)

func TestCodecBuildOpenAIRequest(t *testing.T) {
	req := contract.ChatRequest{
		Messages: []contract.Message{
			{Role: "user", Content: "hello"},
			{
				Role:    "assistant",
				Content: "calling",
				ToolCalls: []contract.ToolCall{{
					ID: "c1", Name: "echo", Arguments: `{"text":"hi"}`,
				}},
			},
			{Role: "tool", Content: "hi", ToolCallID: "c1"},
		},
		Tools: []contract.ToolSpec{{
			Name:        "echo",
			Description: "echo text",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text": map[string]interface{}{"type": "string"},
				},
				"required": []interface{}{"text"},
			},
		}},
		MaxTokens:   128,
		Temperature: 0.25,
	}

	body, err := BuildOpenAIRequest(req, "fixture-model")
	if err != nil {
		t.Fatalf("BuildOpenAIRequest() error = %v", err)
	}

	var got struct {
		Model    string `json:"model"`
		Messages []struct {
			Role       string `json:"role"`
			Content    string `json:"content"`
			ToolCallID string `json:"tool_call_id"`
			ToolCalls  []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"messages"`
		Tools []struct {
			Type     string `json:"type"`
			Function struct {
				Name        string                 `json:"name"`
				Description string                 `json:"description"`
				Parameters  map[string]interface{} `json:"parameters"`
			} `json:"function"`
		} `json:"tools"`
		MaxTokens   int     `json:"max_tokens"`
		Temperature float32 `json:"temperature"`
	}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.Model != "fixture-model" {
		t.Fatalf("model = %q", got.Model)
	}
	if got.Messages[0].Role != "user" || got.Messages[0].Content != "hello" {
		t.Fatalf("user message = %#v", got.Messages[0])
	}
	assistant := got.Messages[1]
	if len(assistant.ToolCalls) != 1 ||
		assistant.ToolCalls[0].Type != "function" ||
		assistant.ToolCalls[0].Function.Name != "echo" {
		t.Fatalf("assistant tool_calls = %#v", assistant.ToolCalls)
	}
	if got.Messages[2].Role != "tool" || got.Messages[2].ToolCallID != "c1" {
		t.Fatalf("tool message = %#v", got.Messages[2])
	}
	if len(got.Tools) != 1 ||
		got.Tools[0].Type != "function" ||
		got.Tools[0].Function.Name != "echo" ||
		got.Tools[0].Function.Parameters["type"] != "object" {
		t.Fatalf("tools = %#v", got.Tools)
	}
	if got.MaxTokens != 128 || got.Temperature != 0.25 {
		t.Fatalf("sampling fields = max_tokens:%d temperature:%v", got.MaxTokens, got.Temperature)
	}
}

func TestCodecParseOpenAIResponse(t *testing.T) {
	tests := []struct {
		name        string
		fixture     string
		wantFinish  string
		wantContent string
		wantCalls   int
	}{
		{
			name:       "tool calls with null content",
			fixture:    `{"choices":[{"finish_reason":"tool_calls","message":{"role":"assistant","content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"echo","arguments":"{\"text\":\"hi\"}"}}]}}]}`,
			wantFinish: contract.FinishToolCalls,
			wantCalls:  1,
		},
		{
			name:        "stop",
			fixture:     `{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"bye"}}]}`,
			wantFinish:  contract.FinishStop,
			wantContent: "bye",
		},
		{
			name:        "length",
			fixture:     `{"choices":[{"finish_reason":"length","message":{"role":"assistant","content":"partial"}}]}`,
			wantFinish:  contract.FinishLength,
			wantContent: "partial",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseOpenAIResponse([]byte(tt.fixture))
			if err != nil {
				t.Fatalf("ParseOpenAIResponse() error = %v", err)
			}
			if got.Finish != tt.wantFinish || got.Message.Content != tt.wantContent {
				t.Fatalf("response = %#v", got)
			}
			if len(got.ToolCalls) != tt.wantCalls || len(got.Message.ToolCalls) != tt.wantCalls {
				t.Fatalf("tool call counts = top:%d message:%d", len(got.ToolCalls), len(got.Message.ToolCalls))
			}
			if tt.wantCalls == 1 {
				if got.ToolCalls[0].ID != "c1" ||
					got.ToolCalls[0].Name != "echo" ||
					got.ToolCalls[0].Arguments != `{"text":"hi"}` {
					t.Fatalf("tool call = %#v", got.ToolCalls[0])
				}
			}
		})
	}
}

func TestCodecParseOpenAIResponseRejectsEmptyChoices(t *testing.T) {
	if _, err := ParseOpenAIResponse([]byte(`{"choices":[]}`)); err == nil {
		t.Fatal("ParseOpenAIResponse() expected error")
	}
}

func TestOpenAIChatUsesPathHeadersAndContext(t *testing.T) {
	requestStarted := make(chan struct{})
	releaseHandler := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret-key" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		close(requestStarted)
		<-releaseHandler
	}))
	defer func() {
		close(releaseHandler)
		server.Close()
	}()

	client := NewOpenAI(server.URL+"/v1/", "secret-key", "fixture-model")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := client.Chat(ctx, contract.ChatRequest{
			Messages: []contract.Message{{Role: "user", Content: "hello"}},
		})
		done <- err
	}()

	select {
	case <-requestStarted:
	case <-time.After(time.Second):
		t.Fatal("request did not reach test server")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, contract.ErrLLMFailed) {
			t.Fatalf("Chat() error = %v, want ErrLLMFailed", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Chat() did not stop after context cancellation")
	}
}

func TestOpenAIChatHandlesFailures(t *testing.T) {
	t.Run("empty api key", func(t *testing.T) {
		client := NewOpenAI("http://127.0.0.1", "", "fixture-model")
		_, err := client.Chat(context.Background(), contract.ChatRequest{})
		if !errors.Is(err, contract.ErrLLMFailed) {
			t.Fatalf("Chat() error = %v, want ErrLLMFailed", err)
		}
	})

	t.Run("non 2xx does not leak response or key", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "upstream-secret-body", http.StatusUnauthorized)
		}))
		defer server.Close()
		client := NewOpenAI(server.URL+"/", "secret-key", "fixture-model")
		_, err := client.Chat(context.Background(), contract.ChatRequest{})
		if !errors.Is(err, contract.ErrLLMFailed) {
			t.Fatalf("Chat() error = %v, want ErrLLMFailed", err)
		}
		if err != nil && (contains(err.Error(), "secret-key") || contains(err.Error(), "upstream-secret-body")) {
			t.Fatalf("Chat() leaked sensitive data: %v", err)
		}
	})

	t.Run("invalid response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.WriteString(w, `{`)
		}))
		defer server.Close()
		client := NewOpenAI(server.URL, "secret-key", "fixture-model")
		_, err := client.Chat(context.Background(), contract.ChatRequest{})
		if !errors.Is(err, contract.ErrLLMFailed) {
			t.Fatalf("Chat() error = %v, want ErrLLMFailed", err)
		}
	})

	t.Run("request encoding", func(t *testing.T) {
		client := NewOpenAI("http://127.0.0.1", "secret-key", "fixture-model")
		_, err := client.Chat(context.Background(), contract.ChatRequest{
			Tools: []contract.ToolSpec{{
				Name:       "bad",
				Parameters: map[string]interface{}{"invalid": func() {}},
			}},
		})
		if !errors.Is(err, contract.ErrLLMFailed) {
			t.Fatalf("Chat() error = %v, want ErrLLMFailed", err)
		}
	})
}

func TestHadeLLMProviderDefaultsAndConfig(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		provider := &HadeLLMProvider{}
		if !provider.IsDefer() || provider.Name() != contract.LLMKey {
			t.Fatalf("provider metadata is invalid")
		}
		got := provider.Params(framework.NewHadeContainer())
		want := []interface{}{"https://api.openai.com/v1", "", "gpt-4o-mini"}
		if len(got) != len(want) {
			t.Fatalf("Params() = %#v", got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("Params()[%d] = %#v, want %#v", i, got[i], want[i])
			}
		}
	})

	t.Run("config overrides", func(t *testing.T) {
		container := framework.NewHadeContainer()
		err := container.Bind(&configprovider.FakeConfigProvider{
			FileName: "llm",
			Content: []byte(`
base_url: http://localhost:9999/v1
api_key: configured-key
model: configured-model
`),
		})
		if err != nil {
			t.Fatalf("bind config: %v", err)
		}
		got := (&HadeLLMProvider{}).Params(container)
		want := []interface{}{"http://localhost:9999/v1", "configured-key", "configured-model"}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("Params()[%d] = %#v, want %#v", i, got[i], want[i])
			}
		}
	})
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
