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
	"github.com/matt/sussout/internal/config"
	"github.com/matt/sussout/internal/db"
	"github.com/matt/sussout/internal/llm"
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

	humanStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Italic(true)

	blockSeparator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#444444"))
)

var validCommands = map[string]bool{
	"/?": true, "/help": true, "/new": true, "/reset": true,
	"/model": true, "/write": true, "/export": true, "/quit": true,
	"/close": true, "/delete": true, "/title": true,
	"/assumptions": true, "/about": true, "/save": true,
	"/db": true,
}

type helpEntry struct {
	cmd, label, desc string
	needsInput       bool
	inputPlaceholder string
}

var helpEntries = []helpEntry{
	{"/?", "/?", "Show this help", false, ""},
	{"/new", "/new", "Start a new conversation", false, ""},
	{"/close", "/close", "Save and return to start screen", false, ""},
	{"/delete", "/delete", "Delete this conversation entirely", false, ""},
	{"/title", "/title <text>", "Rename the current session", true, "New session title..."},
	{"/reset", "/reset", "Clear history in this session", false, ""},
	{"/assumptions", "/assumptions", "List assumptions identified by the agent", false, ""},
	{"/model", "/model", "Select a model from the current server", false, ""},
	{"/write", "/write [file] [instructions]", "Export session to a Markdown file", true, "File path [instructions]..."},
	{"/save", "/save", "Show save status", false, ""},
	{"/about", "/about", "Show version and model info", false, ""},
	{"/db", "/db", "Show current database info", false, ""},
	{"/quit", "/quit", "Exit the application", false, ""},
}

var logoSplash = []string{
	"◉◉◉◉◉◉◉╗    ◉◉◉◉◉◉◉╗",
	"◉◉╔════╝    ◉◉╔══◉◉╗",
	"◉◉◉◉◉◉◉╗    ◉◉║  ◉◉║",
	"╚════◉◉║ ◉◉ ◉◉║  ◉◉║",
	"◉◉◉◉◉◉◉║ ◉○ ╚◉◉◉◉◉◉║",
	"╚══════╝    ╚═════╝ ",
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

type msgLine struct {
	text   string
	isHuman bool
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
	showAbout     bool
	helpCursor    int
	pendingCmd    string
	modelList     []string
	modelIndex    int
	editServer    bool
	editAPIKey    bool
	pendingURL    string
	NewSession    bool
	QuitApp       bool
	statusError   string
	statusSuccess string

	logoPrefix    string
	rawLines    []msgLine
}

func NewTUI(brain *llm.SocraticBrain, store *db.SessionStore, extractor *llm.AssumptionExtractor) *TUI {
	ta := textarea.New()
	ta.Placeholder = ""
	ta.Prompt = ""
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
	logoBuf.WriteString(logoSubStyle.Render("Suss-out ideas."))
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

		if t.pendingCmd != "" && msg.String() == "esc" {
			t.pendingCmd = ""
			t.textarea.Placeholder = ""
			t.textarea.Reset()
			t.textarea.Focus()
			return t, nil
		}

		if (t.showHelp || t.showModels || t.showAbout) && !t.editServer && !t.editAPIKey {
			if msg.String() == "esc" {
				t.showHelp = false
				t.showModels = false
				t.showAbout = false
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
			if t.showHelp {
				switch msg.String() {
				case "up", "k":
					if t.helpCursor > 0 {
						t.helpCursor--
					}
				case "down", "j":
					if t.helpCursor < len(helpEntries)-1 {
						t.helpCursor++
					}
				case "enter":
					if cmd := t.doHelpCommand(t.helpCursor); cmd != nil {
						return t, cmd
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

			if t.pendingCmd != "" {
				arg := strings.TrimSpace(t.textarea.Value())
				if arg == "" {
					return t, nil
				}
				t.textarea.Reset()
				t.textarea.Placeholder = ""
				cmd := t.pendingCmd
				t.pendingCmd = ""
				t.textarea.Focus()
				switch cmd {
				case "/title":
					t.store.SetTitle(context.Background(), t.sessionID, arg)
					t.statusSuccess = "Title updated."
					return t, nil
				case "/write":
					filepath, instructions := parseWriteArgs(arg)
					t.thinking = true
					t.textarea.Blur()
					t.statusSuccess = fmt.Sprintf("Generating document with %s...", t.brain.CurrentModel())
					return t, t.writeDocument(filepath, instructions)
				}
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
				t.QuitApp = true
				return t, tea.Quit
			}

			if input == "/?" || strings.ToLower(input) == "/help" {
				t.showHelp = true
				t.helpCursor = 0
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
				t.thinking = true
				t.textarea.Blur()
				t.statusSuccess = fmt.Sprintf("Generating document with %s...", t.brain.CurrentModel())
				return t, t.writeDocument(filepath, instructions)
			}

			if input == "/reset" {
				t.store.ClearMessages(context.Background(), t.sessionID)
				t.store.ClearAssumptions(context.Background(), t.sessionID)
				t.brain.ResetHistory()
				t.textarea.Reset()
				t.rawLines = nil
				t.appendLine("", false)
				t.appendLine("", false)
				return t, nil
			}

			if input == "/close" {
				t.NewSession = true
				t.statusSuccess = "Saved. Returning to start screen..."
				return t, tea.Quit
			}

			if input == "/delete" {
				t.store.DeleteSession(context.Background(), t.sessionID)
				t.NewSession = true
				return t, tea.Quit
			}

			if input == "/title" {
				t.statusError = "Usage: /title <new title>"
				t.textarea.Reset()
				return t, nil
			}

			if strings.HasPrefix(input, "/title ") {
				title := strings.TrimSpace(strings.TrimPrefix(input, "/title "))
				if title != "" {
					t.store.SetTitle(context.Background(), t.sessionID, title)
					t.statusSuccess = "Title updated."
				}
				t.textarea.Reset()
				return t, nil
			}

			if input == "/assumptions" {
				assumptions, err := t.store.GetAssumptions(context.Background(), t.sessionID)
				t.textarea.Reset()
				if err != nil {
					t.statusError = "Could not load assumptions."
					return t, nil
				}
				if len(assumptions) == 0 {
					t.appendLine("No assumptions recorded yet.", false)
					t.appendLine("", false)
				} else {
					var sb strings.Builder
					sb.WriteString("## Assumptions\n\n")
					for _, a := range assumptions {
						sb.WriteString(fmt.Sprintf("- %s *(status: %s)*\n", a.Content, a.Status))
					}
					t.appendLine(sb.String(), false)
					t.appendLine("", false)
				}
				return t, nil
			}

			if input == "/about" {
				t.showAbout = true
				t.textarea.Blur()
				t.textarea.Reset()
				return t, nil
			}

			if input == "/save" {
				t.textarea.Reset()
				t.statusSuccess = "All progress is saved automatically."
				return t, nil
			}

			if input == "/db" {
				t.textarea.Reset()
				t.statusSuccess = "Database: SQLite (~/.sussout/sussout.db)"
				return t, nil
			}

			t.appendLine(fmt.Sprintf("\nYou: %s", input), true)
			t.appendLine("", false)

			if err := t.store.SaveMessage(context.Background(), t.sessionID, "user", input); err != nil {
				t.err = err
				return t, nil
			}

			t.textarea.Reset()
			t.thinking = true
			t.textarea.Blur()
			t.statusSuccess = fmt.Sprintf("Sending to %s...", t.brain.CurrentModel())
			t.viewport.GotoBottom()
			cmds = append(cmds, t.callLLM(input))
		}

	case llmResponseMsg:
		t.thinking = false
		t.statusSuccess = ""
		t.textarea.Focus()
		if msg.err != nil {
			t.appendLine(fmt.Sprintf("Error: %s", msg.err), false)
			t.appendLine("", false)
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

		t.appendLine(cleaned, false)
		t.appendLine("", false)

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
		t.thinking = false
		t.textarea.Focus()
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
		thinkText := t.statusSuccess
		if thinkText == "" {
			thinkText = "Thinking..."
		}
		status = statusStyle.Render(thinkingDot + "  " + thinkText)
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
	if (t.showHelp || t.showModels || t.showAbout) && !t.editServer && !t.editAPIKey {
		inputArea = inputBorderDim.Render(t.textarea.View())
	} else {
		inputArea = inputBorder.Render(t.textarea.View())
	}

	var centerArea string
	if t.showHelp {
		centerArea = t.renderHelp()
	} else if t.showModels {
		centerArea = t.renderModels()
	} else if t.showAbout {
		centerArea = t.renderAbout()
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

func (t *TUI) appendLine(line string, isHuman bool) {
	t.rawLines = append(t.rawLines, msgLine{text: line, isHuman: isHuman})
	t.rerenderContent()
}

func (t *TUI) rerenderContent() {
	var result strings.Builder
	vw := t.viewport.Width - 2

	var aiBlock []string
	flushAI := func() {
		if len(aiBlock) == 0 {
			return
		}
		blockText := strings.Join(aiBlock, "\n")
		rendered, err := t.renderer.Render(blockText)
		if err != nil {
			result.WriteString(blockText)
		} else {
			result.WriteString(rendered)
		}
		result.WriteString("\n")
		aiBlock = aiBlock[:0]
	}

	hasPrev := false
	for _, l := range t.rawLines {
		if l.isHuman {
			flushAI()
			if hasPrev {
				result.WriteString(blockSeparator.Render(strings.Repeat("─", vw)))
				result.WriteString("\n\n")
			}
			styled := humanStyle.Width(vw).Render(l.text)
			result.WriteString(styled)
			result.WriteString("\n\n")
			hasPrev = true
		} else {
			aiBlock = append(aiBlock, l.text)
		}
	}
	flushAI()

	t.viewport.SetContent(t.logoPrefix + result.String())
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
		case '╭', '╮', '╰', '╯', '│', '─', '╗', '╔', '╝', '║', '╚', '═', '○':
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

	selected := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFD700")).
		Bold(true)

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString("\n")
	sb.WriteString(divider)
	sb.WriteString("\n\n")
	for i, e := range helpEntries {
		if i == t.helpCursor {
			sb.WriteString(selected.Render("> " + e.label))
		} else {
			sb.WriteString(helpCmd.Render("  " + e.label))
		}
		sb.WriteString("\n")
		sb.WriteString(helpDesc.Render("    " + e.desc))
		sb.WriteString("\n\n")
	}
	sb.WriteString(helpDesc.Render("↑↓ to navigate  enter to select  esc to close"))

	content := lipgloss.NewStyle().
		Width(t.width - 8).
		Render(sb.String())

	return helpBorder.Width(t.width - 4).Height(t.viewport.Height).Render(content)
}

func (t *TUI) doHelpCommand(idx int) tea.Cmd {
	if idx < 0 || idx >= len(helpEntries) {
		return nil
	}
	entry := helpEntries[idx]

	if entry.needsInput {
		t.showHelp = false
		t.pendingCmd = entry.cmd
		t.textarea.Placeholder = entry.inputPlaceholder
		t.textarea.Focus()
		t.textarea.Reset()
		return nil
	}

	t.showHelp = false
	t.textarea.Focus()

	switch entry.cmd {
	case "/?":
		t.showHelp = true
		return nil
	case "/new":
		t.NewSession = true
		t.statusSuccess = "Starting a new conversation..."
		return tea.Quit
	case "/close":
		t.NewSession = true
		t.statusSuccess = "Saved. Returning to start screen..."
		return tea.Quit
	case "/delete":
		t.store.DeleteSession(context.Background(), t.sessionID)
		t.NewSession = true
		return tea.Quit
	case "/reset":
		t.store.ClearMessages(context.Background(), t.sessionID)
		t.store.ClearAssumptions(context.Background(), t.sessionID)
		t.brain.ResetHistory()
		t.textarea.Reset()
		t.rawLines = nil
		t.appendLine("", false)
		t.appendLine("", false)
		return nil
	case "/assumptions":
		assumptions, err := t.store.GetAssumptions(context.Background(), t.sessionID)
		if err != nil {
			t.statusError = "Could not load assumptions."
			return nil
		}
		if len(assumptions) == 0 {
			t.appendLine("No assumptions recorded yet.", false)
			t.appendLine("", false)
		} else {
			var sb strings.Builder
			sb.WriteString("## Assumptions\n\n")
			for _, a := range assumptions {
				sb.WriteString(fmt.Sprintf("- %s *(status: %s)*\n", a.Content, a.Status))
			}
			t.appendLine(sb.String(), false)
			t.appendLine("", false)
		}
		return nil
	case "/model":
		t.showModels = true
		t.modelIndex = 0
		models, err := t.brain.ListModels(context.Background())
		if err != nil {
			t.modelList = []string{"(error loading models)"}
		} else {
			t.modelList = models
		}
		return nil
	case "/save":
		t.statusSuccess = "All progress is saved automatically."
		return nil
	case "/db":
		t.statusSuccess = "Database: SQLite (~/.sussout/sussout.db)"
		return nil
	case "/about":
		t.showAbout = true
		return nil
	case "/quit":
		t.QuitApp = true
		return tea.Quit
	}
	return nil
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

func (t *TUI) renderAbout() string {
	if t.width == 0 {
		return ""
	}

	title := helpTitle.Render("About Sussout")
	divider := strings.Repeat("─", 40)

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString("\n")
	sb.WriteString(divider)
	sb.WriteString("\n\n")
	sb.WriteString("Sussout v2.0.0 — Socratic stress-testing of ideas.")
	sb.WriteString("\n\n")
	sb.WriteString("Model: " + t.brain.CurrentModel())
	sb.WriteString("\n\n")
	sb.WriteString("Server: " + t.brain.ServerURL())
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf("Session: %d", t.sessionID))
	sb.WriteString("\n\n")
	sb.WriteString(helpDesc.Render("Press esc to close"))

	content := lipgloss.NewStyle().
		Width(t.width - 8).
		Render(sb.String())

	return helpBorder.Width(t.width - 4).Height(t.viewport.Height).Render(content)
}

func (t *TUI) Run(sessionID int, recapSummary string) error {
	t.sessionID = sessionID

	if recapSummary != "" {
		t.appendLine("─ Resuming conversation ─", false)
		t.appendLine("", false)
		t.appendLine(recapSummary, false)
		t.appendLine("", false)
	}

	_, err := tea.NewProgram(t, tea.WithAltScreen()).Run()
	if t.QuitApp {
		fmt.Print("\033[2J\033[H")
	}
	return err
}

func trimTrailingPerLine(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
}
