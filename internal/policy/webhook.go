package policy

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// sendWebhook dispatches policy violations to the configured webhook URL.
// It filters events by the webhook's Events config before sending.
// This method is called from a goroutine in Evaluate; it must not panic.
func (e *Engine) sendWebhook(violations []Violation, ctx EvalContext) {
	e.mu.RLock()
	wh := e.webhook
	e.mu.RUnlock()

	if wh == nil || wh.URL == "" {
		return
	}

	// Filter: only send if at least one violation matches a configured event.
	shouldSend := false
	for _, v := range violations {
		for _, event := range wh.Events {
			if event == "all" || event == v.Mode {
				shouldSend = true
				break
			}
		}
		if shouldSend {
			break
		}
	}
	if !shouldSend {
		return
	}

	// Build payload using the first violation for the top-level rule fields.
	payload := map[string]interface{}{
		"event":     "policy_violation",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"rule": map[string]string{
			"name": violations[0].Rule,
			"type": violations[0].Type,
			"mode": violations[0].Mode,
		},
		"violation": map[string]interface{}{
			"message": violations[0].Message,
			"blocked": violations[0].Mode == "enforce",
		},
		"request": map[string]string{
			"model":    ctx.Model,
			"agent":    ctx.Agent,
			"session":  ctx.Session,
			"provider": ctx.Provider,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("webhook marshal error: %v", err)
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(wh.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("webhook delivery failed: %v", err)
		return
	}
	resp.Body.Close()
}
