package main

import (
	"fmt"
	"os"
	"sweetspeak/chatpanel"
	"sweetspeak/client"
	log "sweetspeak/logging"
	"sweetspeak/user"
	"time"

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

	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	statusStyle = lipgloss.NewStyle().
			Align(lipgloss.Left)

	serverStatusConnected = statusStyle.Foreground(lipgloss.Color("10")).Render("CONNECTED\n")
	serverStatusPending   = statusStyle.Foreground(lipgloss.Color("8")).Render("PENDING - connecting to server\n")
)

type (
	MainDisplay struct {
		state            sessionState
		sideText         string
		index            int
		ready            bool
		ChatPanel        chatpanel.Model
		client           *client.Client
		serverStatusView string
	}

	SideDisplay struct {
		list []string
	}

	ChatDisplay struct {
		viewport string
		textarea string
	}
)

func newMainDisplay(clientUser *user.User) MainDisplay {
	var (
		chatInputCh = make(chan string)
		m           = MainDisplay{
			sideText: "this is the side view :P",
			ChatPanel: chatpanel.New(
				fmt.Sprintf("%s's Chat", clientUser.Name),
				chatPanelStyle.GetWidth(),
				chatPanelStyle.GetHeight(),
				chatInputCh,
			),
			client: client.New(
				uuid.NewString(),
				clientUser,
				nil,
				chatInputCh,
			),
		}
	)

	go m.client.Start()

	return m
}

func (m MainDisplay) tickEvery() tea.Cmd {
	return tea.Every(time.Second, func(t time.Time) tea.Msg {
		return m.CheckClientConnection(t)
	})
}

func (m MainDisplay) checkChatUpdateEvery() tea.Cmd {
	return tea.Every(time.Second, func(t time.Time) tea.Msg {
		return m.CheckChatText(t)
	})
}

func (m MainDisplay) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.tickEvery(),
		m.checkChatUpdateEvery(),
	}

	return tea.Batch(cmds...)
}

func (m MainDisplay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if !m.ChatPanel.Focused() {
				return m, tea.Quit
			}
		case "tab":
			if m.state == sideView {
				m.state = chatView
			} else if !m.ChatPanel.Focused() {
				m.state = sideView
			}
		default:
			if m.state == chatView {
				m, cmds = m.UpdateChatPanel(msg, cmds)
			}
		}
	case tea.WindowSizeMsg:
		termWidth := msg.Width - 5
		termHeight := msg.Height - 5

		sidePanelWidth := (termWidth / 5)
		chatPanelWidth := termWidth - sidePanelWidth

		sidePanelStyle = styleWithWidthAndHeight(sidePanelStyle, sidePanelWidth, termHeight)
		chatPanelStyle = styleWithWidthAndHeight(chatPanelStyle, chatPanelWidth, termHeight)

		// Update the internal components too
		m.ChatPanel.SetHeight(chatPanelStyle.GetHeight() - 1)
		m.ChatPanel.SetWidth(chatPanelStyle.GetWidth())

		m, cmds = m.UpdateChatPanel(msg, cmds)

		m.ready = true
	case chatpanel.ChatTextMsg:
		m, cmds = m.UpdateChatPanel(msg, cmds)
		cmds = append(cmds, m.checkChatUpdateEvery())
	case ServerStatusMsg:
		if msg.Connected {
			m.serverStatusView = serverStatusConnected
		} else {
			m.serverStatusView = serverStatusPending
		}
		cmds = append(cmds, m.tickEvery())
	}

	return m, tea.Batch(cmds...)
}

func styleWithWidthAndHeight(style lipgloss.Style, width, height int) lipgloss.Style {
	var s lipgloss.Style
	return s.Inherit(style).
		Width(width).
		Height(height)
}

func (m MainDisplay) UpdateChatPanel(msg tea.Msg, cmds []tea.Cmd) (MainDisplay, []tea.Cmd) {
	var cmd tea.Cmd
	m.ChatPanel, cmd = m.ChatPanel.Update(msg)
	cmds = append(cmds, cmd)
	return m, cmds
}

func (m MainDisplay) View() string {
	if !m.ready {
		return "intializing...\n"
	}

	var s string
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
		chatPanelStyle.Render(
			m.ChatPanel.View(),
		),
	)

	s += "\n"
	s += m.serverStatusView

	return s
}

func (m MainDisplay) focusedModel() string {
	if m.state == sideView {
		return "side"
	}
	return "chat"
}

func (m MainDisplay) CheckClientConnection(t time.Time) tea.Msg {
	return NewServerStatusMsg(m.client.Connected)
}

func (m MainDisplay) CheckChatText(t time.Time) tea.Msg {
	if m.client == nil {
		return chatpanel.ChatTextMsg{}
	}

	if m.client.Chat == nil {
		return chatpanel.ChatTextMsg{}
	}

	allMsg := m.client.Chat.GetWithFormat()
	var content string
	for _, m := range allMsg {
		content += m
	}
	return chatpanel.ChatTextMsg{Content: content}
}

func main() {
	if len(os.Args) < 2 {
		log.Warn("need more args! (username)")
		panic(0)
	}
	var (
		userName string
		userColor lipgloss.Color
	)

	userName = os.Args[1]

	if len(os.Args) < 3 {
		userColor = lipgloss.Color("#4287f5")
	} else {
		userColor = lipgloss.Color(fmt.Sprintf("#%s", os.Args[2]))
	}

	clientUser := user.New(userName, userColor)

	log.SetGlobalFile(fmt.Sprintf("sweetspeak-client-%s.log", userName))
	log.SetConsoleOutput(false)

	log.Info("starting user client...")
	p := tea.NewProgram(newMainDisplay(clientUser), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		panic(err)
	}
}

type (
	ErrMsg struct {
		err error
	}

	ServerStatusMsg struct {
		Connected bool
	}

	ChatTextMsg struct {
		content string
	}
)

func NewServerStatusMsg(connected bool) ServerStatusMsg {
	return ServerStatusMsg{
		Connected: connected,
	}
}

func (s ServerStatusMsg) String() string {
	return fmt.Sprintf("%t", s.Connected)
}

func (e ErrMsg) Error() string {
	return e.err.Error()
}
