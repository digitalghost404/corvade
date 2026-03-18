package cli

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

// NewTailCmd creates the "corvade tail" command.
func NewTailCmd() *cobra.Command {
	var agent string
	var model string
	var port int

	cmd := &cobra.Command{
		Use:   "tail",
		Short: "Live-stream trace events from a running Corvade instance",
		Long:  "Connects to the Corvade WebSocket and prints a live trace feed to stdout.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTail(agent, model, port)
		},
	}

	cmd.Flags().StringVar(&agent, "agent", "", "Filter traces by agent name")
	cmd.Flags().StringVar(&model, "model", "", "Filter traces by model name")
	cmd.Flags().IntVar(&port, "port", 4401, "Dashboard API port to connect to")

	return cmd
}

// tailEvent represents a WebSocket event from the hub.
type tailEvent struct {
	Type string                 `json:"event"`
	Data map[string]interface{} `json:"data"`
}

func runTail(agentFilter, modelFilter string, port int) error {
	u := url.URL{Scheme: "ws", Host: fmt.Sprintf("localhost:%d", port), Path: "/ws"}
	fmt.Printf("  Connecting to %s ...\n", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	fmt.Println("  Connected. Watching for traces...")
	fmt.Println()

	// Handle SIGINT gracefully
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var ev tailEvent
			if err := json.Unmarshal(message, &ev); err != nil {
				continue
			}

			if ev.Type != "trace:new" {
				continue
			}

			// Apply filters
			if agentFilter != "" {
				if a, ok := ev.Data["agent"].(string); !ok || a != agentFilter {
					continue
				}
			}
			if modelFilter != "" {
				if m, ok := ev.Data["model"].(string); !ok || m != modelFilter {
					continue
				}
			}

			printTraceEvent(ev.Data)
		}
	}()

	select {
	case <-done:
		return nil
	case <-interrupt:
		fmt.Println("\n  Disconnected.")
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		return nil
	}
}

func printTraceEvent(data map[string]interface{}) {
	ts := time.Now().Format("15:04:05")

	model := stringOrDefault(data, "model", "unknown")
	costVal := floatOrDefault(data, "cost", 0)
	latency := floatOrDefault(data, "latency_ms", 0)
	status := stringOrDefault(data, "status", "")

	costStr := fmt.Sprintf("$%.2f", costVal)
	latencyStr := fmt.Sprintf("%.1fs", latency/1000)

	statusIcon := "✓"
	if status == "error" {
		statusIcon = "✗"
	}

	// Build summary
	parts := []string{statusIcon}
	if provider := stringOrDefault(data, "provider", ""); provider != "" {
		parts = append(parts, provider)
	}

	summary := strings.Join(parts, " ")

	// Pad model to 14 chars for alignment
	if len(model) > 14 {
		model = model[:14]
	}

	fmt.Printf("%s %-14s │ %6s │ %5s │ %s\n", ts, model, costStr, latencyStr, summary)
}

func stringOrDefault(data map[string]interface{}, key, def string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return def
}

func floatOrDefault(data map[string]interface{}, key string, def float64) float64 {
	switch v := data[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	}
	return def
}
