package ui

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/matt/audit/internal/config"
	"github.com/matt/audit/internal/db"
	"github.com/matt/audit/internal/llm"
)

var logoStyles = `{
	"document": {
		"margin": 0,
		"color": "#FFA500"
	},
	"heading": {
		"bold": true,
		"color": "#FFD700"
	},
	"blockquote": {
		"color": "#FFA500",
		"prefix": "\u2502 "
	},
	"code_block": {
		"color": "#FFA500",
		"background_color": "#333333"
	},
	"list": {
		"color": "#FFA500"
	},
	"paragraph": {
		"color": "#FFA500"
	},
	"strong": {
		"bold": true,
		"color": "#FFD700"
	},
	"em": {
		"color": "#FFD700"
	},
	"link": {
		"color": "#FFD700",
		"underline": true
	}
}`

var (
	chatBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#5B9BD5")).
			Padding(0, 1)

	inputBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FFA500")).
			Padding(0, 1)

	inputBorderDim = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#444444")).
			Padding(0, 1)

	statusBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#666666")).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA500")).
			Background(lipgloss.Color("#333333")).
			Padding(0, 1).
			Italic(true)

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Padding(0, 1)

	errorPromptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E04040")).
			Padding(0, 1)

	successPromptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#40A040")).
			Padding(0, 1)

	thinkingDot = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700")).
			Blink(true).
			Render("●")

	logoFillStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E04040"))

	logoOutlineStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	logoSubStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA500")).
			Italic(true)

	helpBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#5B9BD5")).
			Padding(0, 2)

	helpTitle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700")).
			Bold(true)

	helpCmd = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA500"))

	helpDesc = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0"))
)

var validCommands = map[string]bool{
	"/?": true, "/help": true, "/new": true, "/reset": true,
	"/model": true, "/write": true, "/export": true, "/quit": true,
}

var logoSplash = []string{
	" ◉◉◉◉◉╮ ◉◉╮   ◉◉╮◉◉◉◉◉◉╮ ◉◉╮◉◉◉◉◉◉◉◉╮",
	"◉◉╭──◉◉╮◉◉│   ◉◉│◉◉╭──◉◉╮◉◉│╰──◉◉╭──╯",
	"◉◉◉◉◉◉◉│◉◉│   ◉◉│◉◉│  ◉◉│◉◉│   ◉◉│   ",
	"◉◉╭──◉◉│◉◉│   ◉◉│◉◉│  ◉◉│◉◉│   ◉◉│   ",
	"◉◉│  ◉◉│╰◉◉◉◉◉◉╭╯◉◉◉◉◉◉╭╯◉◉│   ◉◉│   ",
	"╰─╯  ╰─╯ ╰─────╯ ╰─────╯ ╰─╯   ╰─╯   ",
}

type llmResponseMsg struct {
	response string
	err      error
	noSave   bool
}

type serverResultMsg struct {
	models  []string
	err     error
	needKey bool
}

type statusMsg struct {
	text  string
	isErr bool
}

type TUI struct {
	brain     *llm.SocraticBrain
	store     *db.SessionStore
	extractor *llm.AssumptionExtractor
	sessionID int

	viewport viewport.Model
	textarea textarea.Model
	renderer *glamour.TermRenderer
	thinking bool
	err      error
	width    int

	showHelp      bool
	showModels    bool
	modelList     []string
	modelIndex    int
	editServer    bool
	editAPIKey    bool
	pendingURL    string
	NewSession    bool
	statusError   string
	statusSuccess string

	logoPrefix    string
	conversation  strings.Builder
}

func NewTUI(brain *llm.SocraticBrain, store *db.SessionStore, extractor *llm.AssumptionExtractor) *TUI {
	ta := textarea.New()
	ta.Placeholder = ""
	ta.Prompt = "? "
	ta.SetHeight(2)
	ta.Focus()
	ta.ShowLineNumbers = false
	ta.CharLimit = 0

	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFD700"))
	ta.FocusedStyle.Text = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E0E0E0"))
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666"))
	ta.BlurredStyle.Text = ta.FocusedStyle.Text
	ta.BlurredStyle.Placeholder = ta.FocusedStyle.Placeholder

	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle()

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes([]byte(logoStyles)),
		glamour.WithWordWrap(76),
	)

	t := &TUI{
		brain:     brain,
		store:     store,
		extractor: extractor,
		textarea:  ta,
		viewport:  vp,
		renderer:  renderer,
	}

	var logoBuf strings.Builder
	logoBuf.WriteString("\n\n")
	for _, line := range logoSplash {
		logoBuf.WriteString(styleLogoLine(line))
		logoBuf.WriteRune('\n')
	}
	logoBuf.WriteString(logoSubStyle.Render("      Socratic auditing of ideas."))
	logoBuf.WriteString("\n\n")
	t.logoPrefix = logoBuf.String()

	return t
}

func (t *TUI) Init() tea.Cmd {
	return textarea.Blink
}

func (t *TUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.viewport.Width = msg.Width - 4
		t.viewport.Height = msg.Height - 9
		t.textarea.SetWidth(msg.Width - 4)

		newRenderer, err := glamour.NewTermRenderer(
			glamour.WithStylesFromJSONBytes([]byte(logoStyles)),
			glamour.WithWordWrap(t.viewport.Width - 2),
		)
		if err == nil {
			t.renderer = newRenderer
			t.rerenderContent()
		}
		return t, nil

case tea.KeyMsg:
		if (t.editServer || t.editAPIKey) && msg.String() == "esc" {
			t.editServer = false
			t.editAPIKey = false
			t.textarea.Placeholder = ""
			t.textarea.Reset()
			t.textarea.Blur()
			return t, nil
		}

		if (t.showHelp || t.showModels) && !t.editServer && !t.editAPIKey {
			if msg.String() == "esc" {
				t.showHelp = false
				t.showModels = false
				t.textarea.Focus()
				return t, nil
			}
			if t.showModels {
				switch msg.String() {
				case "up", "k":
					if t.modelIndex > 0 {
						t.modelIndex--
					}
				case "down", "j":
					if t.modelIndex < len(t.modelList) {
						t.modelIndex++
					}
				case "enter":
					if t.modelIndex == 0 {
						t.editServer = true
						t.textarea.Placeholder = "Server URL..."
						t.textarea.Focus()
						t.textarea.Reset()
						return t, nil
					}
					mi := t.modelIndex - 1
					if mi >= 0 && mi < len(t.modelList) {
						t.brain.SetModel(t.modelList[mi])
						t.showModels = false
						t.textarea.Focus()
						t.statusSuccess = fmt.Sprintf("Switched to: %s", t.modelList[mi])
						t.saveConfig()
					}
				}
			}
			return t, nil
		}

		switch msg.String() {
		case "ctrl+c":
			return t, tea.Quit

		case "enter":
			if t.thinking {
				return t, nil
			}

			if t.editServer {
				url := strings.TrimSpace(t.textarea.Value())
				if url == "" {
					return t, nil
				}
				t.pendingURL = url
				t.textarea.Reset()
				t.textarea.Placeholder = ""
				t.editServer = false
				return t, t.tryServerURL(url)
			}

			if t.editAPIKey {
				key := strings.TrimSpace(t.textarea.Value())
				if key == "" {
					return t, nil
				}
				t.textarea.Reset()
				t.textarea.Placeholder = ""
				t.editAPIKey = false
				return t, t.tryServerKey(t.pendingURL, key)
			}

			input := strings.TrimSpace(t.textarea.Value())
			if input == "" {
				return t, nil
			}
			t.statusError = ""
			t.statusSuccess = ""

			if strings.HasPrefix(input, "/") {
				parts := strings.SplitN(input, " ", 2)
				cmd := strings.ToLower(parts[0])
				if !validCommands[cmd] {
					t.statusError = fmt.Sprintf("Unknown command: %s", parts[0])
					t.textarea.Reset()
					return t, nil
				}
			}

			if strings.ToLower(input) == "/quit" {
				return t, tea.Quit
			}

			if input == "/?" || strings.ToLower(input) == "/help" {
				t.showHelp = true
				t.textarea.Blur()
				t.textarea.Reset()
				return t, nil
			}

			if input == "/new" {
				t.NewSession = true
				t.statusSuccess = "Starting a new conversation..."
				return t, tea.Quit
			}

			if input == "/model" {
				t.showModels = true
				t.modelIndex = 0
				t.textarea.Blur()
				t.textarea.Reset()
				models, err := t.brain.ListModels(context.Background())
				if err != nil {
					t.modelList = []string{"(error loading models)"}
				} else {
					t.modelList = models
				}
				return t, nil
			}

			if strings.HasPrefix(input, "/write ") || input == "/write" || input == "/export" {
				args := strings.TrimSpace(strings.TrimPrefix(input, "/write"))
				if args == "" && input == "/export" {
					args = ""
				}
				filepath, instructions := parseWriteArgs(args)
				t.textarea.Reset()
				return t, t.writeDocument(filepath, instructions)
			}

			if input == "/reset" {
				t.store.ClearMessages(context.Background(), t.sessionID)
				t.store.ClearAssumptions(context.Background(), t.sessionID)
				t.brain.ResetHistory()
				t.textarea.Reset()
				t.conversation.Reset()
				t.appendLine("")
				t.appendLine("")
				return t, nil
			}

			t.appendLine(fmt.Sprintf("\nYou: %s", input))
			t.appendLine("")

			if err := t.store.SaveMessage(context.Background(), t.sessionID, "user", input); err != nil {
				t.err = err
				return t, nil
			}

			t.textarea.Reset()
			t.thinking = true
			t.textarea.Blur()
			t.appendLine("(Sending to LLM...)")
			t.viewport.GotoBottom()
			cmds = append(cmds, t.callLLM(input))
		}

	case llmResponseMsg:
		t.thinking = false
		t.textarea.Focus()
		if msg.err != nil {
			t.appendLine(fmt.Sprintf("Error: %s", msg.err))
			t.appendLine("")
			return t, nil
		}

		if msg.response == "" {
			return t, nil
		}

		cleaned := trimTrailingPerLine(strings.TrimRight(msg.response, " \t\n\r"))
		if !msg.noSave {
			if err := t.store.SaveMessage(context.Background(), t.sessionID, "assistant", cleaned); err != nil {
				t.err = err
				return t, nil
			}
		}

		t.appendLine(cleaned)
		t.appendLine("")

		t.viewport.GotoBottom()

		if !msg.noSave {
			go func() {
			messages, _ := t.store.GetMessages(context.Background(), t.sessionID)
			llmMessages := make([]llm.Message, len(messages))
			for i, m := range messages {
				llmMessages[i] = llm.Message{Role: m.Role, Content: m.Content}
			}
			assumptions, _ := t.extractor.Extract(context.Background(), llmMessages)
			for _, a := range assumptions {
				_ = t.store.SaveAssumption(context.Background(), t.sessionID, a)
			}
		}()
		}

	case serverResultMsg:
		if msg.err != nil {
			t.statusError = fmt.Sprintf("Connection failed: %s", msg.err)
			t.showModels = false
			t.textarea.Focus()
			return t, nil
		}
		if msg.needKey {
			t.editAPIKey = true
			t.textarea.Placeholder = "API key..."
			t.textarea.Focus()
			t.textarea.Reset()
			return t, nil
		}
		t.modelList = msg.models
		t.modelIndex = 1
		t.editServer = false
		t.editAPIKey = false
		t.textarea.Placeholder = ""
		t.textarea.Blur()
		t.statusSuccess = "Connected — select a model"

	case statusMsg:
		if msg.isErr {
			t.statusError = msg.text
		} else {
			t.statusSuccess = msg.text
		}
	}

	var taCmd tea.Cmd
	t.textarea, taCmd = t.textarea.Update(msg)
	cmds = append(cmds, taCmd)

	var vpCmd tea.Cmd
	t.viewport, vpCmd = t.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return t, tea.Batch(cmds...)
}

func (t *TUI) View() string {
	if t.err != nil {
		return fmt.Sprintf("Error: %s\n", t.err)
	}

	var status string
	if t.thinking {
		status = statusStyle.Render(thinkingDot + "  Thinking...")
	} else {
		agentStatus := statusStyle.Render("Ready")
		var userPrompt string
		if t.statusError != "" {
			userPrompt = errorPromptStyle.Render(t.statusError)
		} else if t.statusSuccess != "" {
			userPrompt = successPromptStyle.Render(t.statusSuccess)
		} else {
			userPrompt = promptStyle.Render("Type /? for help")
		}
		status = lipgloss.JoinHorizontal(lipgloss.Left, agentStatus, userPrompt)
	}
	status = lipgloss.NewStyle().Width(t.width - 4).Render(status)
	statusBar := statusBorder.Render(status)

	var inputArea string
	if (t.showHelp || t.showModels) && !t.editServer && !t.editAPIKey {
		inputArea = inputBorderDim.Render(t.textarea.View())
	} else {
		inputArea = inputBorder.Render(t.textarea.View())
	}

	var centerArea string
	if t.showHelp {
		centerArea = t.renderHelp()
	} else if t.showModels {
		centerArea = t.renderModels()
	} else {
		centerArea = chatBorder.Render(t.viewport.View())
	}

	return lipgloss.JoinVertical(
		lipgloss.Top,
		centerArea,
		statusBar,
		inputArea,
	)
}

func (t *TUI) saveConfig() {
	url := t.brain.ServerURL()
	model := t.brain.CurrentModel()
	key := t.brain.APIKey()
	_ = config.SaveCurrentPreset(url, model, key)
}

func (t *TUI) appendLine(line string) {
	t.conversation.WriteString(line + "\n")
	t.rerenderContent()
}

func (t *TUI) rerenderContent() {
	rendered, err := t.renderer.Render(t.conversation.String())
	if err != nil {
		t.viewport.SetContent(t.logoPrefix + t.conversation.String())
	} else {
		t.viewport.SetContent(t.logoPrefix + rendered)
	}
	t.viewport.GotoBottom()
}

func (t *TUI) callLLM(input string) tea.Cmd {
	return func() tea.Msg {
		response, err := t.brain.Ask(context.Background(), input)
		return llmResponseMsg{response: response, err: err}
	}
}

func (t *TUI) tryServerURL(url string) tea.Cmd {
	return func() tea.Msg {
		tmp := llm.NewLMStudioClient(llm.ClientConfig{BaseURL: url})
		models, err := tmp.ListModels(context.Background())
		if err != nil {
			if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
				return serverResultMsg{needKey: true}
			}
			return serverResultMsg{err: err}
		}
		t.brain.SetServerURL(url)
		t.saveConfig()
		return serverResultMsg{models: models}
	}
}

func (t *TUI) tryServerKey(url, key string) tea.Cmd {
	return func() tea.Msg {
		tmp := llm.NewLMStudioClient(llm.ClientConfig{BaseURL: url, APIKey: key})
		models, err := tmp.ListModels(context.Background())
		if err != nil {
			return serverResultMsg{err: err}
		}
		t.brain.SetServerURL(url)
		t.brain.SetAPIKey(key)
		t.saveConfig()
		return serverResultMsg{models: models}
	}
}

func parseWriteArgs(args string) (string, string) {
	args = strings.TrimSpace(args)
	if args == "" {
		return "", ""
	}
	parts := strings.SplitN(args, " ", 2)
	filepath := strings.Trim(parts[0], "\"")
	instructions := ""
	if len(parts) > 1 {
		instructions = strings.TrimSpace(parts[1])
	}
	return filepath, instructions
}

func (t *TUI) writeDocument(filepath, instructions string) tea.Cmd {
	return func() tea.Msg {
		msgs, _ := t.store.GetMessages(context.Background(), t.sessionID)
		llmMsgs := make([]llm.Message, len(msgs))
		for i, m := range msgs {
			llmMsgs[i] = llm.Message{Role: m.Role, Content: m.Content}
		}
		t.brain.LoadHistory(llmMsgs)

		doc, err := t.brain.GenerateDocument(context.Background(), instructions, nil, nil)
		if err != nil {
			return llmResponseMsg{response: "", err: err, noSave: true}
		}

		if filepath == "" {
			return llmResponseMsg{response: doc, err: nil, noSave: true}
		}

		if err := os.WriteFile(filepath, []byte(doc), 0644); err != nil {
			return statusMsg{text: fmt.Sprintf("Export failed: %s", err), isErr: true}
		}
		return statusMsg{text: fmt.Sprintf("Exported to %s", filepath)}
	}
}

func styleLogoLine(line string) string {
	var sb strings.Builder
	for _, c := range line {
		switch c {
		case '◉':
			sb.WriteString(logoFillStyle.Render(string(c)))
		case '╭', '╮', '╰', '╯', '│', '─':
			sb.WriteString(logoOutlineStyle.Render(string(c)))
		default:
			sb.WriteRune(c)
		}
	}
	return sb.String()
}

func (t *TUI) renderHelp() string {
	if t.width == 0 {
		return ""
	}

	title := helpTitle.Render("Commands")
	divider := strings.Repeat("─", 40)

	entries := []struct {
		cmd, desc string
	}{
		{"/?", "Show this help"},
		{"/new", "Save and start a new conversation"},
		{"/reset", "Clear conversation history in this session"},
		{"/model", "Select a model from the current server"},
		{"/write [file] [instructions]", "Export session to a Markdown file"},
		{"/quit", "Exit the session"},
	}

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString("\n")
	sb.WriteString(divider)
	sb.WriteString("\n\n")
	for _, e := range entries {
		sb.WriteString(helpCmd.Render(e.cmd))
		sb.WriteString("\n")
		sb.WriteString(helpDesc.Render("  " + e.desc))
		sb.WriteString("\n\n")
	}
	sb.WriteString(helpDesc.Render("Press esc to close help"))

	content := lipgloss.NewStyle().
		Width(t.width - 8).
		Render(sb.String())

	return helpBorder.Width(t.width - 4).Height(t.viewport.Height).Render(content)
}

func (t *TUI) renderModels() string {
	if t.width == 0 {
		return ""
	}

	title := helpTitle.Render("Select a Model")
	divider := strings.Repeat("─", 40)

	selected := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFD700")).
		Bold(true)

	urlStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888"))

	serverLabel := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#AAAAAA"))

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString("\n")
	sb.WriteString(divider)
	sb.WriteString("\n")
	sb.WriteString(serverLabel.Render("Server:"))
	sb.WriteString("\n")

	if t.modelIndex == 0 {
		sb.WriteString(selected.Render("> " + t.brain.ServerURL()))
	} else {
		sb.WriteString(urlStyle.Render("  " + t.brain.ServerURL()))
	}
	sb.WriteString("\n\n")
	sb.WriteString(divider)
	sb.WriteString("\n")
	sb.WriteString(serverLabel.Render("Models:"))
	sb.WriteString("\n\n")

	for i, m := range t.modelList {
		mi := i + 1
		if mi == t.modelIndex {
			sb.WriteString(selected.Render("> " + m))
		} else {
			sb.WriteString(helpCmd.Render("  " + m))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(helpDesc.Render("↑↓ to navigate  enter to select  esc to close"))

	content := lipgloss.NewStyle().
		Width(t.width - 8).
		Render(sb.String())

	return helpBorder.Width(t.width - 4).Height(t.viewport.Height).Render(content)
}

func (t *TUI) Run(sessionID int, recapSummary string) error {
	t.sessionID = sessionID

	if recapSummary != "" {
		t.appendLine("─ Resuming conversation ─")
		t.appendLine("")
		t.appendLine(recapSummary)
		t.appendLine("")
	}

	_, err := tea.NewProgram(t, tea.WithAltScreen()).Run()
	return err
}

func trimTrailingPerLine(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
}
