package proxy

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/corvade/corvade/internal/capture"
	"github.com/corvade/corvade/internal/cost"
	"github.com/corvade/corvade/internal/proxy/providers"
)

// EventEmitter allows the proxy to emit real-time events (e.g., over WebSocket).
type EventEmitter interface {
	Emit(event string, data interface{})
}

// Server is the core HTTP reverse proxy that intercepts agent-to-LLM traffic.
type Server struct {
	store        *capture.Store
	calc         *cost.Calculator
	emitter      EventEmitter
	upstreamURLs map[string]string
	mux          *http.ServeMux
}

// NewServer creates a proxy server with default upstream URLs for known providers.
func NewServer(store *capture.Store, calc *cost.Calculator, emitter EventEmitter) *Server {
	s := &Server{
		store:   store,
		calc:    calc,
		emitter: emitter,
		upstreamURLs: map[string]string{
			"openai":    "https://api.openai.com",
			"anthropic": "https://api.anthropic.com",
		},
		mux: http.NewServeMux(),
	}
	s.mux.HandleFunc("/", s.handleProxy)
	return s
}

// SetUpstreamURL overrides the upstream URL for a provider (useful for testing).
func (s *Server) SetUpstreamURL(provider, url string) {
	s.upstreamURLs[provider] = url
}

// ServeHTTP delegates to the internal mux.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// detectProvider determines the provider and upstream path from the request path.
// /v1/* routes to openai, /anthropic/* routes to anthropic.
func detectProvider(path string) (provider, upstreamPath string) {
	if strings.HasPrefix(path, "/anthropic/") {
		return "anthropic", strings.TrimPrefix(path, "/anthropic")
	}
	if strings.HasPrefix(path, "/v1/") || path == "/v1" {
		return "openai", path
	}
	return "", path
}

// extractAPIKeyHash extracts the API key from the request and returns its SHA-256 hash.
// Never stores the raw key.
func extractAPIKeyHash(r *http.Request) *string {
	key := ""
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		key = strings.TrimPrefix(auth, "Bearer ")
	} else if xKey := r.Header.Get("x-api-key"); xKey != "" {
		key = xKey
	}
	if key == "" {
		return nil
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
	return &hash
}

// handleProxy is the main handler that forwards requests to the appropriate upstream.
func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	provider, upstreamPath := detectProvider(r.URL.Path)
	if provider == "" {
		http.Error(w, "unknown route: cannot detect provider", http.StatusBadGateway)
		return
	}

	baseURL, ok := s.upstreamURLs[provider]
	if !ok {
		http.Error(w, "no upstream configured for provider: "+provider, http.StatusBadGateway)
		return
	}

	// Extract Corvade headers before stripping
	agent := r.Header.Get("X-Corvade-Agent")
	session := r.Header.Get("X-Corvade-Session")
	step := r.Header.Get("X-Corvade-Step")

	// Extract API key hash
	apiKeyHash := extractAPIKeyHash(r)

	// Read request body
	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	// Parse request info (model, stream)
	var reqInfo *providers.RequestInfo
	switch provider {
	case "openai":
		reqInfo, _ = providers.OpenAIParseRequest(reqBody)
	case "anthropic":
		reqInfo, _ = providers.AnthropicParseRequest(reqBody)
	}
	if reqInfo == nil {
		reqInfo = &providers.RequestInfo{}
	}

	// Build upstream request
	upstreamURL := baseURL + upstreamPath
	if r.URL.RawQuery != "" {
		upstreamURL += "?" + r.URL.RawQuery
	}

	upReq, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL, strings.NewReader(string(reqBody)))
	if err != nil {
		http.Error(w, "failed to create upstream request", http.StatusInternalServerError)
		return
	}

	// Copy headers, stripping Corvade-specific ones
	for key, vals := range r.Header {
		if strings.HasPrefix(strings.ToLower(key), "x-corvade-") {
			continue
		}
		for _, v := range vals {
			upReq.Header.Add(key, v)
		}
	}

	// Forward to upstream
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(upReq)
	if err != nil {
		http.Error(w, "upstream request failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Read upstream response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "failed to read upstream response", http.StatusBadGateway)
		return
	}

	latency := int(time.Since(start).Milliseconds())

	// Parse response info (tokens)
	var respInfo *providers.ResponseInfo
	switch provider {
	case "openai":
		respInfo, _ = providers.OpenAIParseResponse(respBody)
	case "anthropic":
		respInfo, _ = providers.AnthropicParseResponse(respBody)
	}
	if respInfo == nil {
		respInfo = &providers.ResponseInfo{}
	}

	// Write response to client
	for key, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(key, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)

	// Capture trace asynchronously
	go s.captureTrace(provider, reqInfo.Model, string(reqBody), string(respBody),
		resp.StatusCode, respInfo, latency, apiKeyHash, agent, session, step)
}

// captureTrace writes a trace to SQLite in the background. Best-effort — failures
// are logged but never interrupt the agent's response.
func (s *Server) captureTrace(
	provider, model, request, response string,
	statusCode int,
	respInfo *providers.ResponseInfo,
	latencyMS int,
	apiKeyHash *string,
	agent, session, step string,
) {
	if s.store == nil {
		return
	}

	tr := capture.Trace{
		Provider:   provider,
		Model:      model,
		Request:    request,
		Response:   &response,
		StatusCode: statusCode,
		LatencyMS:  &latencyMS,
		APIKeyHash: apiKeyHash,
	}

	if agent != "" {
		tr.Agent = &agent
	}
	if session != "" {
		tr.SessionID = &session
	}
	if step != "" {
		tr.Step = &step
	}

	if respInfo != nil {
		tr.TokensPrompt = &respInfo.PromptTokens
		tr.TokensCompletion = &respInfo.CompletionTokens
		if respInfo.CachedTokens > 0 {
			tr.TokensCached = &respInfo.CachedTokens
		}

		if s.calc != nil && model != "" {
			c := s.calc.Calculate(model, respInfo.PromptTokens, respInfo.CompletionTokens)
			if c > 0 {
				tr.Cost = &c
			}
		}
	}

	traceID, err := s.store.InsertTrace(tr)
	if err != nil {
		log.Printf("capture trace failed: %v", err)
		return
	}

	// Emit event if emitter is configured
	if s.emitter != nil {
		s.emitter.Emit("trace:new", map[string]interface{}{
			"id":       traceID,
			"provider": provider,
			"model":    model,
		})
	}
}
