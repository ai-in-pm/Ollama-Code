package ui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"
)

// Styles for different UI elements
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			PaddingLeft(2).
			PaddingRight(2)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)

	userInputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	assistantStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#88FF88"))

	highlightStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF88FF"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))

	codeBlockStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FFFF")).
			Background(lipgloss.Color("#333333")).
			PaddingLeft(1).
			PaddingRight(1)
)

// Message represents a message in the chat history
type Message struct {
	Role    string
	Content string
	Time    time.Time
}

// TerminalUI represents the terminal UI state
type TerminalUI struct {
	width      int
	height     int
	ready      bool
	viewport   viewport.Model
	textInput  textinput.Model
	spinner    spinner.Model
	loading    bool
	messages   []Message
	mutex      sync.Mutex
	outputChan chan string
	errChan    chan error
	statusMsg  string
	statusType string // "info", "error", "success"
}

// NewTerminalUI creates a new terminal UI
func NewTerminalUI() *TerminalUI {
	ti := textinput.New()
	ti.Placeholder = "Ask Ollama Code something..."
	ti.Focus()
	ti.Width = 80
	ti.CharLimit = 512

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	return &TerminalUI{
		textInput:  ti,
		spinner:    sp,
		messages:   []Message{},
		outputChan: make(chan string, 100),
		errChan:    make(chan error, 10),
		statusMsg:  "Ready",
		statusType: "info",
	}
}

// Start initializes and starts the terminal UI
func (tui *TerminalUI) Start() error {
	p := tea.NewProgram(tui)

	// Handle streaming output and errors
	go func() {
		for {
			select {
			case output, ok := <-tui.outputChan:
				if !ok {
					return
				}
				p.Send(appendMessageMsg{content: output, role: "assistant", append: true})
			case err, ok := <-tui.errChan:
				if !ok {
					return
				}
				p.Send(errorMsg{err})
			}
		}
	}()

	_, err := p.Run()
	return err
}

// StreamOutput provides streaming output of AI responses
func (tui *TerminalUI) StreamOutput(output string) {
	tui.outputChan <- output
}

// ReportError reports an error to the UI
func (tui *TerminalUI) ReportError(err error) {
	tui.errChan <- err
}

// SetLoading sets the loading state and status message
func (tui *TerminalUI) SetLoading(loading bool, message string) {
	if loading {
		if message == "" {
			message = "Loading..."
		}
		tui.outputChan <- fmt.Sprintf("⏳ %s", message)
	}
}

// Custom tea.Msg types
type appendMessageMsg struct {
	content string
	role    string
	append  bool
}

type errorMsg struct {
	err error
}

type loadingMsg struct {
	loading bool
}

// Init initializes the TUI
func (tui *TerminalUI) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		spinner.Tick,
	)
}

// Update updates the TUI state
func (tui *TerminalUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return tui, tea.Quit
		case tea.KeyEnter:
			if tui.loading {
				return tui, nil
			}

			userInput := tui.textInput.Value()
			if userInput == "" {
				return tui, nil
			}

			tui.AddMessage("user", userInput)
			tui.textInput.Reset()

			// Set loading state
			tui.loading = true
			cmds = append(cmds, func() tea.Msg {
				return loadingMsg{loading: true}
			})
		}

	case appendMessageMsg:
		if msg.append {
			// Append to the last message
			if len(tui.messages) > 0 && tui.messages[len(tui.messages)-1].Role == msg.role {
				tui.mutex.Lock()
				tui.messages[len(tui.messages)-1].Content += msg.content
				tui.mutex.Unlock()
			} else {
				tui.AddMessage(msg.role, msg.content)
			}
		} else {
			tui.AddMessage(msg.role, msg.content)
		}

		tui.loading = false
		tui.statusMsg = "Ready"
		tui.statusType = "info"

		// Update viewport to show the latest content
		cmds = append(cmds, func() tea.Msg {
			tui.UpdateViewContent()
			return nil
		})

	case errorMsg:
		tui.loading = false
		tui.statusMsg = fmt.Sprintf("Error: %v", msg.err)
		tui.statusType = "error"
		tui.AddMessage("system", "Error: "+msg.err.Error())

	case loadingMsg:
		tui.loading = msg.loading
		if tui.loading {
			tui.statusMsg = "Thinking..."
			tui.statusType = "info"
			cmds = append(cmds, spinner.Tick)
		} else {
			tui.statusMsg = "Ready"
		}

	case tea.WindowSizeMsg:
		tui.width = msg.Width
		tui.height = msg.Height

		if !tui.ready {
			tui.viewport = viewport.New(msg.Width, msg.Height-6) // Leave room for input and status
			tui.viewport.HighPerformanceRendering = true
			tui.UpdateViewContent()
			tui.ready = true
		} else {
			tui.viewport.Width = msg.Width
			tui.viewport.Height = msg.Height - 6
		}

		tui.textInput.Width = msg.Width - 4

	case spinner.TickMsg:
		if tui.loading {
			var spinnerCmd tea.Cmd
			tui.spinner, spinnerCmd = tui.spinner.Update(msg)
			cmds = append(cmds, spinnerCmd)
		}
	}

	// Update viewport
	if tui.ready {
		viewportCmd := tui.viewport.Update(msg)
		cmds = append(cmds, viewportCmd)
	}

	// Update text input
	tui.textInput, cmd = tui.textInput.Update(msg)
	cmds = append(cmds, cmd)

	return tui, tea.Batch(cmds...)
}

// View renders the TUI
func (tui *TerminalUI) View() string {
	if !tui.ready {
		return "Initializing..."
	}

	var sb strings.Builder

	// Title bar
	title := titleStyle.Render("Ollama Code")
	padding := strings.Repeat(" ", max(0, tui.width-lipgloss.Width(title)))
	sb.WriteString(title + padding + "\n\n")

	// Messages viewport
	sb.WriteString(tui.viewport.View() + "\n\n")

	// Status line
	var statusText string
	switch tui.statusType {
	case "error":
		statusText = errorStyle.Render(tui.statusMsg)
	case "success":
		statusText = assistantStyle.Render(tui.statusMsg)
	default:
		statusText = infoStyle.Render(tui.statusMsg)
	}

	if tui.loading {
		sb.WriteString(fmt.Sprintf("%s %s\n", tui.spinner.View(), statusText))
	} else {
		sb.WriteString(fmt.Sprintf("%s\n", statusText))
	}

	// Input prompt
	promptText := promptStyle.Render(">>> ")
	sb.WriteString(promptText + userInputStyle.Render(tui.textInput.View()))

	return sb.String()
}

// AddMessage adds a message to the chat history
func (tui *TerminalUI) AddMessage(role, content string) {
	tui.mutex.Lock()
	defer tui.mutex.Unlock()

	tui.messages = append(tui.messages, Message{
		Role:    role,
		Content: content,
		Time:    time.Now(),
	})

	tui.UpdateViewContent()
}

// UpdateViewContent updates the viewport with formatted messages
func (tui *TerminalUI) UpdateViewContent() {
	tui.mutex.Lock()
	defer tui.mutex.Unlock()

	if !tui.ready {
		return
	}

	var content strings.Builder

	for i, msg := range tui.messages {
		if i > 0 {
			content.WriteString("\n\n")
		}

		switch msg.Role {
		case "user":
			content.WriteString(promptStyle.Render("You: "))
			content.WriteString(userInputStyle.Render(msg.Content))
		case "assistant":
			content.WriteString(assistantStyle.Render("Ollama Code: "))

			// Format code blocks with proper styling
			formattedContent := formatCodeBlocks(msg.Content, tui.width)
			content.WriteString(formattedContent)
		case "system":
			content.WriteString(infoStyle.Render(msg.Content))
		}
	}

	tui.viewport.SetContent(content.String())
	tui.viewport.GotoBottom()
}

// formatCodeBlocks applies styling to code blocks
func formatCodeBlocks(content string, width int) string {
	parts := strings.Split(content, "```")
	if len(parts) <= 1 {
		return assistantStyle.Render(content)
	}

	var result strings.Builder
	for i, part := range parts {
		if i%2 == 0 {
			// Regular text
			if part != "" {
				result.WriteString(assistantStyle.Render(part))
			}
		} else {
			// Code block
			// Check if there's a language specifier
			lines := strings.SplitN(part, "\n", 2)
			language := ""
			code := part

			if len(lines) > 1 {
				language = strings.TrimSpace(lines[0])
				code = lines[1]
			}

			// Add the code block with styling
			if language != "" {
				result.WriteString("\n" + highlightStyle.Render(language) + "\n")
			} else {
				result.WriteString("\n")
			}

			// Wrap code for better display
			wrappedCode := wrap.String(code, width-4)
			result.WriteString(codeBlockStyle.Render(wrappedCode))
		}
	}

	return result.String()
}

// Helper function since Go doesn't have a built-in max function for ints
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
