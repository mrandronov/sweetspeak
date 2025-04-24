package user

import "github.com/charmbracelet/lipgloss"

type (
	User struct {
		ID    string         `yaml:"id"`
		Name  string         `yaml:"name"`
		Color lipgloss.Color `yaml:"color"`
	}
)

func New(name string, color lipgloss.Color) *User {
	return &User{
		Name:  name,
		Color: color,
	}
}
