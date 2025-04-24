package chat

import (
	"sweetspeak/message"
	"sweetspeak/user"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

type (
	Chat struct {
		sync.Mutex
		ID       string
		Name     string
		Users    []user.User
		Messages []message.TextMessage
	}
)

func New(id string, name string, users []user.User) *Chat {
	return &Chat{
		ID:    id,
		Name:  name,
		Users: users,
	}
}

func (c *Chat) AddMessage(tm message.TextMessage) {
	c.Lock()
	defer c.Unlock()

	c.Messages = append(c.Messages, tm)
}

func (c *Chat) GetMessages() []string {
	c.Lock()
	defer c.Unlock()

	var allMsg []string

	for _, m := range c.Messages {
		allMsg = append(allMsg, m.Content)
	}

	return allMsg
}

func (c *Chat) GetWithFormat() []string {
	c.Lock()
	defer c.Unlock()

	var allMsg []string

	for _, m := range c.Messages {
		content := m.Content
		content = lipgloss.NewStyle().Foreground(m.From.Color).Render(m.From.Name+":") + " " + content + "\n"
		allMsg = append(allMsg, content)
	}

	return allMsg
}
