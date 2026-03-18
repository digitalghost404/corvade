package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/corvade/corvade/internal/capture"
	"github.com/spf13/cobra"
)

// NewSessionsCmd creates the "corvade sessions" command.
func NewSessionsCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "sessions",
		Short: "List recent sessions",
		Long:  "Lists recent agent sessions with trace counts, costs, and status.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSessions(limit)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of sessions to display")

	return cmd
}

func runSessions(limit int) error {
	home, _ := os.UserHomeDir()
	dbPath := filepath.Join(home, ".corvade", "corvade.db")

	store, err := capture.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer store.Close()

	sessions, err := store.ListSessionsFiltered(capture.SessionFilter{
		Limit: &limit,
	})
	if err != nil {
		return fmt.Errorf("listing sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("  No sessions found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tAGENT\tTRACES\tCOST\tSTATUS\tSTARTED")
	fmt.Fprintln(w, "──\t─────\t──────\t────\t──────\t───────")

	for _, s := range sessions {
		id := truncateID(s.ID)
		agent := ptrStringOrDash(s.Agent)
		costStr := fmt.Sprintf("$%.2f", s.TotalCost)
		started := s.StartTime.Local().Format(time.RFC3339)

		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\n",
			id, agent, s.TraceCount, costStr, s.Status, started)
	}

	w.Flush()
	return nil
}

// truncateID returns the first 8 characters of an ID for display.
func truncateID(id string) string {
	if len(id) > 8 {
		return id[:8] + "..."
	}
	return id
}

// ptrStringOrDash returns the string value or "-" if nil.
func ptrStringOrDash(s *string) string {
	if s != nil && *s != "" {
		return *s
	}
	return "-"
}
