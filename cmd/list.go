package cmd

import (
	"context"
	"fmt"

	"github.com/matt/audit/internal/config"
	"github.com/matt/audit/internal/db"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all past Socratic sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load("")

		if cfg.DatabaseURL == "" {
			return fmt.Errorf("DATABASE_URL environment variable is not set")
		}

		ctx := context.Background()

		pool, err := db.NewPool(ctx, cfg.DatabaseURL)
		if err != nil {
			return fmt.Errorf("database connection: %w", err)
		}
		defer pool.Close()

		store := db.NewSessionStore(pool)
		sessions, err := store.ListSessions(ctx)
		if err != nil {
			return fmt.Errorf("list sessions: %w", err)
		}

		if len(sessions) == 0 {
			fmt.Println("No past sessions found.")
			return nil
		}

		fmt.Println("\nPast Socratic Sessions:")
		fmt.Println("-----------------------")
		for _, s := range sessions {
			fmt.Printf("ID: %d | Title: %s | Updated: %s\n",
				s.ID, s.Title, s.UpdatedAt.Format("2006-01-02 15:04:05"))
		}
		fmt.Println()
		return nil
	},
}
