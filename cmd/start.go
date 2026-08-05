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

	"github.com/charmbracelet/lipgloss"
	"github.com/matt/audit/internal/config"
	"github.com/matt/audit/internal/db"
	"github.com/matt/audit/internal/llm"
	"github.com/matt/audit/internal/ui"
	"github.com/spf13/cobra"
)

var pickerBox = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#5B9BD5")).
	Padding(0, 1).
	Width(60)

var pickerHeader = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FFD700")).
	Bold(true).
	Render("Recent Sessions")

var pickerHint = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#888888")).
	Render("Type a number to resume, 'm' for more, Enter for new")

var (
	startTitle   string
	startContext string
	startPreset  string
	startModel   string
	startAPIKey  string
)

func init() {
	startCmd.Flags().StringVarP(&startTitle, "title", "t", "New Creative Project", "Session title")
	startCmd.Flags().StringVarP(&startContext, "context", "c", "", "Initial context file to read")
	startCmd.Flags().StringVar(&startPreset, "preset", "", "LLM preset to use (see 'helpme config list')")
	startCmd.Flags().StringVar(&startModel, "model", "", "Override the LLM model (e.g. 'gpt-4o')")
	startCmd.Flags().StringVar(&startAPIKey, "api-key", "", "Override the LLM API key")
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a new Socratic session, or pick up a recent one",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load(startPreset)

		if cfg.DatabaseURL == "" {
			return fmt.Errorf("DATABASE_URL environment variable is not set")
		}

		if startModel != "" {
			cfg.LLM.Model = startModel
		}
		if startAPIKey != "" {
			cfg.LLM.APIKey = startAPIKey
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

		extractor := llm.NewAssumptionExtractor(llmClient)

		reader := bufio.NewReader(os.Stdin)

		for {
			sessionID, err := pickSession(ctx, store, reader)
			if err != nil {
				return fmt.Errorf("session selection: %w", err)
			}

			brain := llm.NewSocraticBrain(llmClient)
			tui := ui.NewTUI(brain, store, extractor)

			if sessionID == 0 {
				session, err := store.CreateSession(ctx, startTitle)
				if err != nil {
					return fmt.Errorf("create session: %w", err)
				}
				err = tui.Run(session.ID, loadContextFile())
				if tui.NewSession {
					continue
				}
				return err
			}

			messages, err := store.GetMessages(ctx, sessionID)
			if err != nil {
				return fmt.Errorf("get messages: %w", err)
			}

			if len(messages) == 0 {
				fmt.Fprintf(os.Stderr, "Session %d has no messages. Delete it? [y/N]: ", sessionID)
				resp, _ := reader.ReadString('\n')
				if strings.ToLower(strings.TrimSpace(resp)) == "y" {
					if err := store.DeleteSession(ctx, sessionID); err != nil {
						fmt.Fprintf(os.Stderr, "Delete failed: %s\n", err)
					} else {
						fmt.Fprintf(os.Stderr, "Session %d deleted.\n\n", sessionID)
					}
					continue
				}
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
			}

			if ctxFile := loadContextFile(); ctxFile != "" {
				if recapSummary != "" {
					recapSummary += "\n\n---\n\n" + ctxFile
				} else {
					recapSummary = ctxFile
				}
			}

			err = tui.Run(sessionID, recapSummary)
			if tui.NewSession {
				continue
			}
			return err
		}
	},
}

func pickSession(ctx context.Context, store *db.SessionStore, reader *bufio.Reader) (int, error) {
	all, err := store.ListSessions(ctx)
	if err != nil || len(all) == 0 {
		return 0, nil
	}

	offset := 0
	batchSize := 3

	for {
		end := offset + batchSize
		if end > len(all) {
			end = len(all)
		}
		batch := all[offset:end]

		var sb strings.Builder
		sb.WriteString(pickerHeader + "\n\n")
		for _, s := range batch {
			sb.WriteString(fmt.Sprintf("[%d]  %s  (%s)\n",
				s.ID, s.Title, s.UpdatedAt.Format("Jan 2 15:04")))
		}

		hasMore := end < len(all)
		if hasMore {
			sb.WriteString("\n[m]  More sessions...\n")
		}
		sb.WriteString("\n" + pickerHint)

		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, pickerBox.Render(sb.String()))
		fmt.Fprintf(os.Stderr, "\n> ")

		input, err := reader.ReadString('\n')
		if err != nil {
			return 0, nil
		}
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "" {
			return 0, nil
		}

		if input == "m" && hasMore {
			offset += batchSize
			continue
		}

		id, err := strconv.Atoi(input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Invalid choice. Type a number, 'm', or press Enter.\n")
			continue
		}

		for _, s := range batch {
			if s.ID == id {
				return id, nil
			}
		}
		fmt.Fprintf(os.Stderr, "  Session %d not in this list.\n", id)
	}
}

func loadContextFile() string {
	if startContext == "" {
		return ""
	}
	content, err := os.ReadFile(startContext)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not read context file: %s\n", err)
		return ""
	}
	return string(content)
}
