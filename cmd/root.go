package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "helpme",
	Short: "Audit — a Socratic tool to audit ideas and activate critical thinking.",
	Long: `Aporia uses a local LLM to challenge your assumptions and
guide your thinking through Socratic dialogue. All conversations are
stored in a local PostgreSQL database.

Examples:
  helpme start
  helpme start --title "Designing a new API"
  helpme list
  helpme resume 1`,
	Version: "2.0.0",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
