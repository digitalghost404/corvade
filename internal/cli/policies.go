package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/corvade/corvade/internal/policy"
	"github.com/corvade/corvade/internal/policy/rules"
	"gopkg.in/yaml.v3"
	"github.com/spf13/cobra"
)

// NewPoliciesCmd creates the "corvade policies" command with a "validate" subcommand.
func NewPoliciesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policies",
		Short: "Display loaded policy rules",
		Long:  "Reads ~/.corvade/policies.yaml and displays all configured policy rules.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPolicies()
		},
	}

	cmd.AddCommand(newPoliciesValidateCmd())
	return cmd
}

func newPoliciesValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate the policies file and exit",
		Long:  "Parses ~/.corvade/policies.yaml and exits 0 on success, 1 on errors.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPoliciesValidate()
		},
	}
}

func defaultPolicyPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".corvade", "policies.yaml")
}

func runPolicies() error {
	policyPath := defaultPolicyPath()

	// Check if file exists
	if _, err := os.Stat(policyPath); os.IsNotExist(err) {
		fmt.Printf("  No policies file found at %s\n", policyPath)
		return nil
	}

	builder := rules.DefaultBuilder()
	summaries, err := policy.Validate(policyPath, builder)
	if err != nil {
		fmt.Printf("  Error loading policies: %v\n", err)
		return err
	}

	// Print table
	w := tabwriter.NewWriter(os.Stdout, 2, 0, 3, ' ', 0)
	fmt.Fprintln(w, "  Rule\tType\tMode\tStatus")
	for _, s := range summaries {
		status := "✓ valid"
		if !s.Valid {
			status = "✗ " + s.Error
		}
		fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", s.Name, s.Type, s.Mode, status)
	}
	w.Flush()
	fmt.Println()

	// Print summary line
	fmt.Printf("  %d rule(s) loaded from %s\n", len(summaries), policyPath)

	// Print webhook status if configured
	webhookURL := readWebhookURL(policyPath)
	if webhookURL != "" {
		truncated := webhookURL
		if len(truncated) > 40 {
			truncated = truncated[:40] + "..."
		}
		fmt.Printf("  Webhook: configured (%s)\n", truncated)
	}

	return nil
}

func runPoliciesValidate() error {
	policyPath := defaultPolicyPath()

	// Check if file exists
	if _, err := os.Stat(policyPath); os.IsNotExist(err) {
		fmt.Printf("  No policies file found at %s\n", policyPath)
		os.Exit(1)
	}

	builder := rules.DefaultBuilder()
	summaries, err := policy.Validate(policyPath, builder)
	if err != nil {
		fmt.Printf("  ✗ %v\n", err)
		fmt.Println("  1 error found")
		os.Exit(1)
	}

	fmt.Printf("  ✓ policies.yaml is valid (%d rule(s))\n", len(summaries))
	return nil
}

// validatePolicyFile validates the policy file at the given path using the default rule builder.
// Returns RuleSummary slice on success or an error.
func validatePolicyFile(policyPath string) ([]policy.RuleSummary, error) {
	return policy.Validate(policyPath, rules.DefaultBuilder())
}

// readWebhookURL reads the webhook URL from the policy file without full validation.
func readWebhookURL(policyPath string) string {
	data, err := os.ReadFile(policyPath)
	if err != nil {
		return ""
	}

	var pf policy.PolicyFile
	if err := yaml.Unmarshal(data, &pf); err != nil {
		return ""
	}

	if pf.Webhook != nil {
		return pf.Webhook.URL
	}
	return ""
}
