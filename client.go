package main

import (
	"fmt"
	"os"
	"sweetspeak/client"
	log "sweetspeak/logging"
	"sweetspeak/user"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
)

type sessionState int

const (
	sideView sessionState = iota
	chatView
)

var (
	mainUpdatePeriod = 10 * time.Millisecond

	focusColor   = lipgloss.Color("69")
	unfocusColor = lipgloss.Color("241")

	sidePanelStyle = lipgloss.NewStyle().
			Align(lipgloss.Center, lipgloss.Center).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("#FAFAFA"))
	chatPanelStyle = lipgloss.NewStyle().
			Align(lipgloss.Center, lipgloss.Center).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("#FAFAFA"))

	chatStyle = lipgloss.NewStyle().Align(lipgloss.Bottom)

	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	statusStyle = lipgloss.NewStyle().
			Align(lipgloss.Left)
)

type (
	MainDisplay struct {
		state        sessionState
		sideText     string
		chatText     string
		viewport     viewport.Model
		chatInput    textinput.Model
		index        int
		ready        bool
		client       *client.Client
		updateTicker time.Ticker
	}

	SideDisplay struct {
		list []string
	}

	ChatDisplay struct {
		viewport string
		textarea string
	}
)

func newMainDisplay(userName string) MainDisplay {
	newUser := user.New(userName, lipgloss.Color("#4287f5"))
        if userName == "void_star" {
                newUser = user.New(userName, lipgloss.Color("#61eb34"))
        }
	m := MainDisplay{
		sideText:     "this is the side view :P",
		chatText:     "",
		client:       client.New(uuid.NewString(), newUser, nil),
		updateTicker: *time.NewTicker(mainUpdatePeriod),
	}

	m.viewport = viewport.New(chatPanelStyle.GetWidth(), 20)
	m.viewport.SetContent(m.chatText)

	m.chatInput = textinput.New()
	m.chatInput.Placeholder = "Type a message here..."
	m.chatInput.Cursor.Blink = false

	// Connect the client to start
	if err := m.client.Start(); err != nil {
		log.Error("client connect failed: %v", err)
	} else {
		// Introduction was already sent, now send
		// a default chat request.

		if userName == "Michael" {
			to := "void_star"
			m.client.SendChatRequest(to)
		}
	}

	return m
}

func (m MainDisplay) Init() tea.Cmd {
	return m.updateTextMsgs
}

func styleWithWidthAndHeight(style lipgloss.Style, width, height int) lipgloss.Style {
	var s lipgloss.Style
	return s.Inherit(style).
		Width(width).
		Height(height)
}

func (m MainDisplay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if !m.chatInput.Focused() {
				return m, tea.Quit
			}
		case "tab":
			if m.state == sideView {
				m.state = chatView
			} else if !m.chatInput.Focused() {
				m.state = sideView
			}
		case "esc":
			m.chatInput.Blur()
		case "enter":
			if m.state == chatView {
				if !m.chatInput.Focused() {
					m.chatInput.Focus()
				} else {
					// Just send the text and reset the text input
					msgText := m.chatInput.Value()
					m.client.SendChatMessage(msgText)
					m.chatInput.Reset()
				}
			}
		}
	case textMsgStatus:
		content := msg.String()
		m.chatText = content
		m.viewport.SetContent(m.chatText)
	case tea.WindowSizeMsg:
		termWidth := msg.Width - 5
		termHeight := msg.Height - 5

		sidePanelWidth := (termWidth / 5)
		chatPanelWidth := termWidth - sidePanelWidth

		sidePanelStyle = styleWithWidthAndHeight(sidePanelStyle, sidePanelWidth, termHeight)
		chatPanelStyle = styleWithWidthAndHeight(chatPanelStyle, chatPanelWidth, termHeight)

		// Update the internal components too
		viewPortHeight := chatPanelStyle.GetHeight() - 5
		m.viewport = viewport.New(chatPanelStyle.GetWidth(), viewPortHeight)
		m.viewport.SetContent(m.chatText)

		chatStyle.Width(chatPanelStyle.GetWidth())
		m.chatInput.Width = chatPanelStyle.GetWidth() - 5

		m.ready = true
	}

	if m.chatInput.Focused() {
		var cmd tea.Cmd
		m.chatInput, cmd = m.chatInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Additional default commands to execute...

	select {
	case <-m.updateTicker.C:
		cmds = append(cmds, m.updateTextMsgs)
	default:
	}

	return m, tea.Batch(cmds...)
}

func (m MainDisplay) View() string {
	if !m.ready {
		return "intializing...\n"
	}

	var (
		s string
	)

	if m.state == sideView {
		// Side panel is focused
		sidePanelStyle = sidePanelStyle.BorderForeground(focusColor)
		chatPanelStyle = chatPanelStyle.BorderForeground(unfocusColor)
	} else {
		// Chat panel is focused
		sidePanelStyle = sidePanelStyle.BorderForeground(unfocusColor)
		chatPanelStyle = chatPanelStyle.BorderForeground(focusColor)
	}

	s += lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidePanelStyle.Render( // Side Display
			m.sideText,
		),
		chatPanelStyle.Render( // Chat Display
			lipgloss.JoinVertical(
				lipgloss.Top,
				m.viewport.View(),
				chatStyle.Render(m.chatInput.View()),
			),
		),
	)

	s += "\n"

	if m.client.Connected {
                statusStyle.UnsetForeground()
                s += statusStyle.Foreground(lipgloss.Color("4")).Render("CONNECTED\n")
	} else {
                s += statusStyle.Foreground(lipgloss.Color("8")).Render("PENDING - connecting to server\n")
	}

	return s
}

func (m MainDisplay) focusedModel() string {
	if m.state == sideView {
		return "side"
	}
	return "chat"
}

func main() {
	if len(os.Args) < 2 {
		log.Warn("need more args! (username)")
		panic(0)
	}
	userName := os.Args[1]

	log.SetGlobalFile(fmt.Sprintf("sweetspeak-client-%s.log", userName))
	log.SetConsoleOutput(false)

	log.Info("starting user client...")
	p := tea.NewProgram(newMainDisplay(userName), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		panic(err)
	}
}

func (m MainDisplay) updateTextMsgs() tea.Msg {
	chatText := ""
	if m.client.Chat == nil {
		return textMsgStatus{}
	}

	chatMessage := m.client.Chat.GetWithFormat()
	for _, msg := range chatMessage {
		chatText += msg
	}

	return NewTextMsgStatus(chatText)
}

type (
	textMsgStatus struct{ content string }
	errMsg        struct{ err error }
)

func NewTextMsgStatus(content string) textMsgStatus {
	return textMsgStatus{content: content}
}

func (t textMsgStatus) String() string {
	return t.content
}

func (e errMsg) Error() string { return e.err.Error() }
