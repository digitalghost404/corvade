package cli

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/corvade/corvade/internal/config"
	"github.com/spf13/cobra"

	_ "github.com/mattn/go-sqlite3"
)

// NewDoctorCmd creates the "corvade doctor" command.
func NewDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check system health and prerequisites",
		Long:  "Runs diagnostic checks to verify that Corvade can start correctly.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor()
		},
	}
	return cmd
}

func runDoctor() error {
	fmt.Println()

	cfg, _, _ := config.Load(config.DefaultPath())
	allOK := true

	// Check proxy port
	if checkPort(cfg.Port) {
		fmt.Printf("  ✓ Port %d available\n", cfg.Port)
	} else {
		fmt.Printf("  ✗ Port %d in use\n", cfg.Port)
		allOK = false
	}

	// Check dashboard port
	if checkPort(cfg.DashboardPort) {
		fmt.Printf("  ✓ Port %d available\n", cfg.DashboardPort)
	} else {
		fmt.Printf("  ✗ Port %d in use\n", cfg.DashboardPort)
		allOK = false
	}

	// Check SQLite writable
	home, _ := os.UserHomeDir()
	dbPath := filepath.Join(home, ".corvade", "corvade.db")
	if checkSQLiteWritable(dbPath) {
		fmt.Printf("  ✓ SQLite writable at %s\n", dbPath)
	} else {
		fmt.Printf("  ✗ SQLite not writable at %s\n", dbPath)
		allOK = false
	}

	// Check disk space
	diskSpace, err := getAvailableDiskSpace(home)
	if err == nil {
		fmt.Printf("  ✓ %.1f GB disk space available\n", diskSpace)
	} else {
		fmt.Printf("  ✗ Could not check disk space: %v\n", err)
		allOK = false
	}

	fmt.Println()

	if !allOK {
		return fmt.Errorf("some checks failed")
	}
	return nil
}

// checkPort attempts to listen on the port to see if it's available.
func checkPort(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// checkSQLiteWritable verifies the directory is writable and a db can be created.
func checkSQLiteWritable(dbPath string) bool {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false
	}

	// Try to create/open a temp file in the directory
	tmpFile := filepath.Join(dir, ".corvade-doctor-test")
	f, err := os.Create(tmpFile)
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(tmpFile)
	return true
}
