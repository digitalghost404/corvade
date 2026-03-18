package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/corvade/corvade/internal/capture"
	"github.com/spf13/cobra"
)

// NewInspectCmd creates the "corvade inspect" command.
func NewInspectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect <id>",
		Short: "Inspect a trace or session by ID",
		Long:  "Looks up the given ID as a trace first, then as a session. Prints the record as formatted JSON.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInspect(args[0])
		},
	}

	return cmd
}

func runInspect(id string) error {
	home, _ := os.UserHomeDir()
	dbPath := filepath.Join(home, ".corvade", "corvade.db")

	store, err := capture.NewStore(dbPath)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer store.Close()

	// Try trace first
	trace, err := store.GetTrace(id)
	if err == nil {
		return printJSON(trace)
	}

	// Try session
	session, err := store.GetSession(id)
	if err == nil {
		return printJSON(session)
	}

	return fmt.Errorf("no trace or session found with ID: %s", id)
}

func printJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
