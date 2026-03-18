package main

import (
	"fmt"
	"os"

	"github.com/corvade/corvade/internal/cli"
	"github.com/spf13/cobra"
)

var version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:   "corvade",
	Short: "The AI agent control plane",
	Long:  "Corvade intercepts agent-to-LLM traffic for debugging, governance, and memory.",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("corvade v%s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(cli.NewStartCmd(version))
	rootCmd.AddCommand(cli.NewTailCmd())
	rootCmd.AddCommand(cli.NewDoctorCmd())
	rootCmd.AddCommand(cli.NewSessionsCmd())
	rootCmd.AddCommand(cli.NewInspectCmd())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
