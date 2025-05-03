package chatpanel

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color("#FAFAFA"))
	chatStyle = lipgloss.NewStyle().Align(lipgloss.Bottom)
)

type (
	Model struct {
		titleText   string
		chatText    string
		viewport    viewport.Model
		chatInput   textinput.Model
		height      int
		width       int
		focused     bool
		chatInputCh chan string
	}
)

func New(title string, width, height int, chatInputCh chan string) Model {
	m := Model{
		titleText: title,
		chatText:  "Hello world...",
		viewport:  viewport.New(width, height),
		chatInput: textinput.New(),
		height:    height,
		width:     width,
		chatInputCh: chatInputCh,
	}

	m.viewport.SetContent(m.chatText)

	m.chatInput.Placeholder = "Say hello..."
	m.chatInput.Cursor.Blink = false

	return m
}

func (m *Model) SetTitle(title string) *Model {
	m.titleText = title
	return m
}

func (m *Model) SetHeight(height int) *Model {
	m.height = height
	return m
}

func (m *Model) SetWidth(width int) *Model {
	m.width = width
	return m
}

func (m Model) Focus() {
	m.focused = true
	m.chatInput.Focus()
}

func (m Model) Unfocus() {
	m.focused = false
	m.chatInput.Blur()
}

func (m Model) Focused() bool {
	return m.chatInput.Focused()
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			m.chatInput.Blur()
		case tea.KeyEnter:
			if !m.chatInput.Focused() {
				m.chatInput.Focus()
			} else {
				msgText := m.chatInput.Value()
				// Send the text on a message channel I suppose
				m.chatInputCh <- msgText
				m.chatInput.Reset()
			}
		}
	case tea.WindowSizeMsg:
		m.viewport = viewport.New(m.width, m.height-1)
		m.viewport.SetContent(m.chatText)

		m.chatInput.Width = m.width - 5

		titleStyle = titleStyle.Width(m.width)
	case ChatTextMsg:
		if msg.String() == "" {
			break
		}
		m.chatText = msg.String()
		m.viewport.SetContent(m.chatText)
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	if m.chatInput.Focused() {
		var cmd tea.Cmd
		m.chatInput, cmd = m.chatInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Top,
		titleStyle.Render(m.titleText),
		m.viewport.View(),
		chatStyle.Render(m.chatInput.View()),
	)
}

type (
	ChatTextMsg struct {
		Content string
	}

	ErrMsg struct {
		err error
	}
)

func NewChatTextMsg(content string) ChatTextMsg {
	return ChatTextMsg{
		Content: content,
	}
}

func (c ChatTextMsg) String() string {
	return c.Content
}

func (e ErrMsg) Error() string {
	return e.err.Error()
}
