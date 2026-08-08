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
	"github.com/matt/sussout/internal/config"
	"github.com/matt/sussout/internal/db"
	"github.com/matt/sussout/internal/llm"
	"github.com/matt/sussout/internal/ui"
	"github.com/spf13/cobra"
)

var pickerBox = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#5B9BD5")).
	Padding(0, 1)

var pickerEditBox = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#E04040")).
	Padding(0, 1)

var pickerLogoLines = []string{
	"●●●●●●●╗●●╗   ●●╗●●●●●●●╗●●●●●●●╗ ●●●●●●╗ ●●╗   ●●╗●●●●●●●●╗",
	"●●╔════╝●●║   ●●║●●╔════╝●●╔════╝●●╔═══●●╗●●║   ●●║╚══●●╔══╝",
	"●●●●●●●╗●●║   ●●║●●●●●●●╗●●●●●●●╗●●║   ●●║●●║   ●●║   ●●║   ",
	"╚════●●║●●║   ●●║╚════●●║╚════●●║●●║   ●●║●●║   ●●║   ●●║   ",
	"●●●●●●●║╚●●●●●●╔╝●●●●●●●║●●●●●●●║╚●●●●●●╔╝╚●●●●●●╔╝   ●●║   ",
	"╚══════╝ ╚═════╝ ╚══════╝╚══════╝ ╚═════╝  ╚═════╝    ╚═╝   ",
}

var pickerLogoFill = lipgloss.NewStyle().Foreground(lipgloss.Color("#E04040"))
var pickerLogoOutline = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))

var pickerPrompt = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FFFFFF"))

var promptBorder = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#FFFFFF")).
	Padding(0, 1)

var pickerStatusBar = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#666666")).
	Padding(0, 1)

var pickerStatusText = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#888888")).
	Italic(true)

var pickerHeader = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FFD700")).
	Bold(true).
	Render("Recent Sessions")

var pickerError = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FF6B6B"))

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
	startCmd.Flags().StringVar(&startPreset, "preset", "", "LLM preset to use (see 'sussout config list')")
	startCmd.Flags().StringVar(&startModel, "model", "", "Override the LLM model (e.g. 'gpt-4o')")
	startCmd.Flags().StringVar(&startAPIKey, "api-key", "", "Override the LLM API key")
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a new Socratic session, or pick up a recent one",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprint(os.Stderr, "\033[2J\033[H")
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

		extractor := llm.NewAssumptionExtractor(llmClient)

		reader := bufio.NewReader(os.Stdin)

		for {
			sessionID, quit, err := pickSession(ctx, store, reader, displayModel)
			if err != nil {
				return fmt.Errorf("session selection: %w", err)
			}
			if quit {
				return nil
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
				var sb strings.Builder
				sb.WriteString(renderLogoForPicker())
				sb.WriteString("\n\n")
				sb.WriteString(pickerHeader + "\n\n")
				sb.WriteString(fmt.Sprintf("Session %d has no messages.", sessionID) + "\n")
				sb.WriteString("Delete it? [y/N]")

				fmt.Fprintln(os.Stderr)
				bw := boxWidth
				fmt.Fprintln(os.Stderr, pickerBox.Copy().Width(bw).Render(sb.String()))
				printBorderedPrompt(bw)

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

			store.TouchSession(ctx, sessionID)

			var recapSummary string
			renderStatusOverlay(fmt.Sprintf("Looking up conversation %d...", sessionID))
			summary, recapErr := brain.Recap(ctx)
			clearStatusOverlay()
			if recapErr != nil {
				recapSummary = fmt.Sprintf("(Unable to generate recap: %s)", recapErr)
			} else if summary != "" {
				recapSummary = summary
				store.SaveMessage(ctx, sessionID, "assistant", summary)
				history = append(history, llm.Message{Role: "assistant", Content: summary})
				brain.LoadHistory(history)
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

func pickSession(ctx context.Context, store *db.SessionStore, reader *bufio.Reader, model string) (int, bool, error) {
	all, err := store.ListSessions(ctx)
	if err != nil || len(all) == 0 {
		if err == nil {
			showWelcome(model)
		}
		return 0, false, nil
	}

	offset := 0
	batchSize := 3
	recentID := all[0].ID
	var statusErr string

	for {
		end := offset + batchSize
		if end > len(all) {
			end = len(all)
		}
		batch := all[offset:end]

		var sb strings.Builder
		sb.WriteString(renderLogoForPicker())
		sb.WriteString("\n")
		sb.WriteString(pickerStatusText.Render("Using model: " + model))
		sb.WriteString("\n\n")
		sb.WriteString(pickerHeader + "\n\n")
		for _, s := range batch {
			prefix := ""
			if s.ID == recentID {
				prefix = "★ "
			}
			sb.WriteString(fmt.Sprintf("[%d]  %s%s  (%s)\n",
				s.ID, prefix, s.Title, s.UpdatedAt.Format("Jan 2 15:04")))
		}

		hasPrev := offset > 0
		hasMore := end < len(all)
		if hasPrev || hasMore {
			sb.WriteString("\n")
			if hasPrev {
				sb.WriteString("[b]  Back\n")
			}
			if hasMore {
				sb.WriteString("[m]  More sessions...\n")
			}
		}
		sb.WriteString("[⏎]  Start new conversation\n")
		sb.WriteString("[e]  Edit conversations\n")
		sb.WriteString("[q]  Quit\n")

		if statusErr != "" {
			sb.WriteString("\n" + pickerError.Render("  " + statusErr) + "\n")
			statusErr = ""
		}

	fmt.Fprintln(os.Stderr)
	bw := boxWidth
	fmt.Fprintln(os.Stderr, pickerBox.Copy().Width(bw).Render(sb.String()))
	printBorderedPrompt(bw)

		input, err := reader.ReadString('\n')
		if err != nil {
			return 0, false, nil
		}
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "" {
			return 0, false, nil
		}

		if input == "q" {
			fmt.Fprint(os.Stderr, "\033[2J\033[H")
			return 0, true, nil
		}

		if input == "e" {
			editSessions(ctx, store, reader, model)
			all, _ = store.ListSessions(ctx)
			offset = 0
			if len(all) > 0 {
				recentID = all[0].ID
			}
			continue
		}

		if input == "m" && hasMore {
			offset += batchSize
			continue
		}

		if input == "b" && hasPrev {
			offset -= batchSize
			continue
		}

		id, err := strconv.Atoi(input)
		if err != nil {
			statusErr = "Invalid input — type a session number."
			continue
		}

		for _, s := range batch {
			if s.ID == id {
				return id, false, nil
			}
		}
		for _, s := range all {
			if s.ID == id {
				return id, false, nil
			}
		}
		statusErr = fmt.Sprintf("Session %d not found.", id)
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

func renderPickerLogo() string {
	var sb strings.Builder
	for _, line := range pickerLogoLines {
		for _, c := range line {
			switch c {
			case '●':
				sb.WriteString(pickerLogoFill.Render(string(c)))
			case '╭', '╮', '╰', '╯', '│', '─', '╗', '╔', '╝', '║', '╚', '═':
				sb.WriteString(pickerLogoOutline.Render(string(c)))
			default:
				sb.WriteRune(c)
			}
		}
		sb.WriteRune('\n')
	}
	return sb.String()
}

func printBorderedPrompt(width int) {
	fmt.Fprint(os.Stderr, "\n"+promptBorder.Copy().Width(width).Render(pickerPrompt.Render("> ")))
	fmt.Fprint(os.Stderr, "\033[1A\033[6G")
}

func renderStatusOverlay(msg string) {
	w := boxWidth
	bar := pickerStatusBar.Copy().Width(w).Render("\n" + pickerStatusText.Render(msg) + "\n")
	fmt.Fprint(os.Stderr, "\r\033[3A"+bar)
	fmt.Fprint(os.Stderr, "\033[2A\033[3G")
}

func clearStatusOverlay() {
	fmt.Fprint(os.Stderr, "\r\033[K\033[A\r\033[K\033[A\r\033[K\033[A\r\033[K\033[A\r\033[K")
}

const boxWidth = 64

func showWelcome(model string) {
	bw := boxWidth
	fmt.Fprintln(os.Stderr)
	content := renderLogoForPicker() + "\n" + pickerStatusText.Render("Using model: "+model) + "\n\nStarting a new session..."
	fmt.Fprintln(os.Stderr, pickerBox.Copy().Width(bw).Render(content))
	fmt.Fprintln(os.Stderr)
}

func editSessions(ctx context.Context, store *db.SessionStore, reader *bufio.Reader, model string) {
	all, err := store.ListSessions(ctx)
	if err != nil || len(all) == 0 {
		return
	}

	const pageSize = 10
	offset := 0

	for {
		end := offset + pageSize
		if end > len(all) {
			end = len(all)
		}
		page := all[offset:end]

		var sb strings.Builder
		sb.WriteString(renderLogoForPicker())
		sb.WriteString("\n")
		sb.WriteString(pickerStatusText.Render("Using model: " + model))
		sb.WriteString("\n\n")
		sb.WriteString(pickerHeader + " (editing)\n\n")
		for _, s := range page {
			sb.WriteString(fmt.Sprintf("[%d]  %s  (%s)\n",
				s.ID, s.Title, s.UpdatedAt.Format("Jan 2 15:04")))
		}

		hasPrev := offset > 0
		hasNext := offset+pageSize < len(all)
		if hasPrev || hasNext {
			sb.WriteString("\n")
			if hasPrev {
				sb.WriteString("[b]  Back\n")
			}
			if hasNext {
				sb.WriteString("[n]  Next\n")
			}
		}
		sb.WriteString("\n" + pickerStatusText.Render("Type a number to select, Enter to go back"))

		fmt.Fprintln(os.Stderr)
		bw := boxWidth
		fmt.Fprintln(os.Stderr, pickerEditBox.Copy().Width(bw).Render(sb.String()))
		printBorderedPrompt(bw)

		input, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "" {
			return
		}
		if input == "n" && hasNext {
			offset += pageSize
			continue
		}
		if input == "b" && hasPrev {
			offset -= pageSize
			continue
		}

		id, err := strconv.Atoi(input)
		if err != nil {
			continue
		}

		var session *db.Session
		for i := range all {
			if all[i].ID == id {
				session = &all[i]
				break
			}
		}
		if session == nil {
			continue
		}

		var subSb strings.Builder
		subSb.WriteString(renderLogoForPicker())
		subSb.WriteString("\n")
		subSb.WriteString(pickerStatusText.Render("Using model: " + model))
		subSb.WriteString("\n\n")
		subSb.WriteString(pickerHeader + "\n\n")
		subSb.WriteString(fmt.Sprintf("[%d]  %s  (%s)\n\n", session.ID, session.Title, session.UpdatedAt.Format("Jan 2 15:04")))
		subSb.WriteString("[d]  Delete this conversation\n")
		subSb.WriteString("[r]  Rename this conversation\n")
		subSb.WriteString("\n" + pickerStatusText.Render("Type d or r, Enter to go back"))

		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, pickerEditBox.Copy().Width(bw).Render(subSb.String()))
		printBorderedPrompt(bw)

		sub, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		sub = strings.TrimSpace(strings.ToLower(sub))

		switch sub {
		case "d":
			var delSb strings.Builder
			delSb.WriteString(renderLogoForPicker())
			delSb.WriteString("\n")
			delSb.WriteString(pickerStatusText.Render("Using model: " + model))
			delSb.WriteString("\n\n")
			delSb.WriteString(pickerHeader + "\n\n")
			delSb.WriteString(fmt.Sprintf("Delete session %d: %s?\n\n", session.ID, session.Title))
			delSb.WriteString(pickerError.Render("[y]  Yes, delete\n"))
			delSb.WriteString(pickerStatusText.Render("Enter to cancel"))

			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, pickerEditBox.Copy().Width(bw).Render(delSb.String()))
			printBorderedPrompt(bw)

			confirm, _ := reader.ReadString('\n')
			if strings.ToLower(strings.TrimSpace(confirm)) == "y" {
				store.DeleteSession(ctx, session.ID)
			}
			all, _ = store.ListSessions(ctx)
			if offset >= len(all) && offset > 0 {
				offset = max(0, len(all)-pageSize)
			}
		case "r":
			var renSb strings.Builder
			renSb.WriteString(renderLogoForPicker())
			renSb.WriteString("\n")
			renSb.WriteString(pickerStatusText.Render("Using model: " + model))
			renSb.WriteString("\n\n")
			renSb.WriteString(pickerHeader + "\n\n")
			renSb.WriteString(fmt.Sprintf("Rename session %d\nCurrent: %s\n\n", session.ID, session.Title))
			renSb.WriteString(pickerStatusText.Render("Type new title, or Enter to cancel"))

			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, pickerEditBox.Copy().Width(bw).Render(renSb.String()))
			printBorderedPrompt(bw)

			title, _ := reader.ReadString('\n')
			title = strings.TrimSpace(title)
			if title != "" {
				store.SetTitle(ctx, session.ID, title)
			}
			all, _ = store.ListSessions(ctx)
		}
	}
}

func renderLogoForPicker() string {
	return renderPickerLogo()
}
