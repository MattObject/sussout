package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/matt/audit/internal/config"
	"github.com/matt/audit/internal/db"
	"github.com/matt/audit/internal/llm"
	"github.com/matt/audit/internal/ui"
	"github.com/spf13/cobra"
)

var resumePreset string

func init() {
	resumeCmd.Flags().StringVar(&resumePreset, "preset", "", "LLM preset to use (see 'helpme config list')")
	rootCmd.AddCommand(resumeCmd)
}

var resumeCmd = &cobra.Command{
	Use:   "resume <id>",
	Short: "Resume a past Socratic session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load(resumePreset)

		if cfg.DatabaseURL == "" {
			return fmt.Errorf("DATABASE_URL environment variable is not set")
		}

		sessionID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid session ID: %s", args[0])
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			cancel()
		}()

		pool, err := db.NewPool(ctx, cfg.DatabaseURL)
		if err != nil {
			return fmt.Errorf("database connection: %w", err)
		}
		defer pool.Close()

		store := db.NewSessionStore(pool)
		llmClient := llm.NewLMStudioClient(cfg.LLM)

		displayModel := cfg.LLM.Model
		if displayModel == "" {
			detected, err := llmClient.Detect()
			if err != nil {
				return fmt.Errorf("model detection: %w", err)
			}
			displayModel = detected
		}
		fmt.Fprintf(os.Stderr, "\nUsing model: %s\n\n", displayModel)

		brain := llm.NewSocraticBrain(llmClient)
		extractor := llm.NewAssumptionExtractor(llmClient)

		for {
			tui := ui.NewTUI(brain, store, extractor)

			messages, err := store.GetMessages(ctx, sessionID)
			if err != nil {
				return fmt.Errorf("get messages: %w", err)
			}

			if len(messages) == 0 {
				reader := bufio.NewReader(os.Stdin)
				fmt.Fprintf(os.Stderr, "Session %d has no messages. Delete it? [y/N]: ", sessionID)
				resp, _ := reader.ReadString('\n')
				if strings.ToLower(strings.TrimSpace(resp)) == "y" {
					if err := store.DeleteSession(ctx, sessionID); err != nil {
						fmt.Fprintf(os.Stderr, "Delete failed: %s\n", err)
					} else {
						fmt.Fprintf(os.Stderr, "Session %d deleted.\n", sessionID)
					}
				}
				return nil
			}

			history := make([]llm.Message, len(messages))
			for i, m := range messages {
				history[i] = llm.Message{Role: m.Role, Content: m.Content}
			}
			brain.LoadHistory(history)

			var recapSummary string
			fmt.Fprintf(os.Stderr, "Looking up conversation...")
			summary, recapErr := brain.Recap(ctx)
			fmt.Fprintf(os.Stderr, "\r\033[K")
			if recapErr != nil {
				recapSummary = fmt.Sprintf("(Unable to generate recap: %s)", recapErr)
			} else if summary != "" {
				recapSummary = summary
			} else if len(brain.GetHistory()) == 0 {
				recapSummary = "This session has no previous messages."
			}

			err = tui.Run(sessionID, recapSummary)
			if !tui.NewSession {
				return err
			}

			// Start a fresh session
			newSession, err := store.CreateSession(ctx, "New Session")
			if err != nil {
				return fmt.Errorf("create session: %w", err)
			}
			sessionID = newSession.ID
			brain = llm.NewSocraticBrain(llmClient)
		}
	},
}
