package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
Use: "sussout",
	Short: "Sussout — a Socratic tool to stress-test ideas.",
	Long: `Sussout uses a local LLM to challenge your assumptions and
guide your thinking through Socratic dialogue. All conversations are
stored in a local PostgreSQL database.

Examples:
  sussout start
  sussout start --title "Designing a new API"
  sussout list
  sussout resume 1`,
	Version: "2.0.0",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
