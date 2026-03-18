package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/corvade/corvade/internal/api"
	"github.com/corvade/corvade/internal/capture"
	"github.com/corvade/corvade/internal/cost"
	"github.com/corvade/corvade/internal/proxy"
)

// mockOpenAIServer returns an httptest.Server that simulates the OpenAI API.
func mockOpenAIServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":    "chatcmpl-integration-test",
			"model": "gpt-4o",
			"usage": map[string]int{
				"prompt_tokens":     100,
				"completion_tokens": 50,
			},
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"role":    "assistant",
						"content": "Integration test response",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestFullPipeline(t *testing.T) {
	// 1. Create mock upstream OpenAI server
	upstream := mockOpenAIServer(t)
	defer upstream.Close()

	// 2. Create temp SQLite store
	store, err := capture.NewStore(t.TempDir() + "/integration.db")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// 3. Create a Hub and start it
	hub := api.NewHub()
	go hub.Run()

	// 4. Create a proxy server pointed at the mock upstream
	calc := cost.NewCalculator(nil)
	proxySrv := proxy.NewServer(store, calc, hub)
	proxySrv.SetUpstreamURL("openai", upstream.URL)

	proxyTestSrv := httptest.NewServer(proxySrv)
	defer proxyTestSrv.Close()

	// 5. Send a POST /v1/chat/completions through the proxy with X-Corvade-Agent header
	reqBody := `{"model":"gpt-4o","messages":[{"role":"user","content":"integration test"}]}`
	req, err := http.NewRequest(http.MethodPost, proxyTestSrv.URL+"/v1/chat/completions", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-test-integration-key")
	req.Header.Set("X-Corvade-Agent", "integration-agent")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("proxy request failed: %v", err)
	}
	defer resp.Body.Close()

	// 6. Verify 200 response
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected proxy status 200, got %d", resp.StatusCode)
	}

	// 7. Wait 200ms for async capture
	time.Sleep(200 * time.Millisecond)

	// 8. Query the API (RegisterRoutes on a new mux) to GET /api/traces
	apiMux := http.NewServeMux()
	api.RegisterRoutes(apiMux, store, hub)
	apiSrv := httptest.NewServer(apiMux)
	defer apiSrv.Close()

	apiResp, err := http.Get(apiSrv.URL + "/api/traces")
	if err != nil {
		t.Fatalf("GET /api/traces failed: %v", err)
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode != http.StatusOK {
		t.Fatalf("expected GET /api/traces status 200, got %d", apiResp.StatusCode)
	}

	var traces []capture.Trace
	if err := json.NewDecoder(apiResp.Body).Decode(&traces); err != nil {
		t.Fatalf("failed to decode traces response: %v", err)
	}

	// 9. Verify 1 trace returned with correct model, agent, and non-zero cost
	if len(traces) != 1 {
		t.Fatalf("expected 1 trace, got %d", len(traces))
	}

	tr := traces[0]

	if tr.Model != "gpt-4o" {
		t.Errorf("expected model 'gpt-4o', got %q", tr.Model)
	}

	if tr.Agent == nil || *tr.Agent != "integration-agent" {
		t.Errorf("expected agent 'integration-agent', got %v", tr.Agent)
	}

	if tr.Cost == nil || *tr.Cost <= 0 {
		t.Errorf("expected non-zero cost, got %v", tr.Cost)
	}
}
