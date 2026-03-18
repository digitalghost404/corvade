package cli

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/corvade/corvade/internal/api"
	"github.com/corvade/corvade/internal/capture"
	"github.com/corvade/corvade/internal/config"
	"github.com/corvade/corvade/internal/cost"
	"github.com/corvade/corvade/internal/proxy"
	"github.com/spf13/cobra"
)

// NewStartCmd creates the "corvade start" command.
func NewStartCmd(version string) *cobra.Command {
	var headless bool
	var demo bool
	var port int
	var dashboardPort int

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the Corvade proxy and dashboard",
		Long:  "Starts the proxy server, API server, and WebSocket hub for intercepting agent-to-LLM traffic.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart(version, headless, demo, port, dashboardPort)
		},
	}

	cmd.Flags().BoolVar(&headless, "headless", false, "Disable the dashboard API server")
	cmd.Flags().BoolVar(&demo, "demo", false, "Start with demo data loaded")
	cmd.Flags().IntVar(&port, "port", 0, "Proxy port (default: from config or 4400)")
	cmd.Flags().IntVar(&dashboardPort, "dashboard-port", 0, "Dashboard API port (default: from config or 4401)")

	return cmd
}

func runStart(version string, headless, demo bool, portOverride, dashboardPortOverride int) error {
	// 1. Load config
	cfgPath := config.DefaultPath()
	cfg, created, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if created {
		fmt.Printf("  Created default config at %s\n\n", cfgPath)
	}

	// Apply flag overrides
	if portOverride > 0 {
		cfg.Port = portOverride
	}
	if dashboardPortOverride > 0 {
		cfg.DashboardPort = dashboardPortOverride
	}

	// 2. Open SQLite store
	home, _ := os.UserHomeDir()
	dbPath := filepath.Join(home, ".corvade", "corvade.db")
	store, err := capture.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer store.Close()

	// 2b. Load demo data if requested
	if demo {
		count, err := loadDemoData(store)
		if err != nil {
			return fmt.Errorf("loading demo data: %w", err)
		}
		fmt.Printf("  Loaded %d demo traces\n", count)
	}

	// 3. Create cost calculator with overrides from config
	overrides := make(map[string]cost.ModelPricing)
	for model, co := range cfg.CostOverrides {
		overrides[model] = cost.ModelPricing{
			InputPer1K:  co.InputPer1K,
			OutputPer1K: co.OutputPer1K,
		}
	}
	calc := cost.NewCalculator(overrides)

	// 4. Start WebSocket hub
	hub := api.NewHub()
	go hub.Run()

	// 5. Start proxy server
	proxySrv := proxy.NewServer(store, calc, hub)
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Port)
		if err := http.ListenAndServe(addr, proxySrv); err != nil {
			log.Fatalf("proxy server error: %v", err)
		}
	}()

	// 6. Start API server (unless headless)
	if !headless {
		apiSrv := api.NewAPIServer(store, hub, cfg.DashboardPort)
		go func() {
			if err := apiSrv.Start(); err != nil {
				log.Fatalf("api server error: %v", err)
			}
		}()
	}

	// 7. Print startup banner
	dbSize := getFileSize(dbPath)
	fmt.Println()
	fmt.Printf("  ▗▖  Corvade v%s\n", version)
	fmt.Println("  ▝▘  The AI agent control plane")
	fmt.Println()
	fmt.Printf("  Proxy:      http://localhost:%d\n", cfg.Port)
	if !headless {
		fmt.Printf("  Dashboard:  http://localhost:%d\n", cfg.DashboardPort)
	}
	fmt.Printf("  Storage:    %s (%s)\n", dbPath, dbSize)
	fmt.Println()
	fmt.Println("  Point your agents here:")
	fmt.Printf("    OPENAI_BASE_URL=http://localhost:%d/v1\n", cfg.Port)
	fmt.Printf("    ANTHROPIC_BASE_URL=http://localhost:%d/anthropic\n", cfg.Port)
	fmt.Println()
	fmt.Println("  Watching for traffic...")

	// 8. Wait for SIGINT/SIGTERM
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	fmt.Println("\n  Shutting down...")

	return nil
}

// getFileSize returns a human-readable file size string.
func getFileSize(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "0 B"
	}
	size := info.Size()
	switch {
	case size >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(size)/float64(1<<30))
	case size >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(size)/float64(1<<20))
	case size >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(size)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
