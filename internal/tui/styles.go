package tui

import "github.com/charmbracelet/lipgloss"

// Styles holds all lipgloss styles used in the TUI
type Styles struct {
	Sender lipgloss.Style
	Bot    lipgloss.Style
	System lipgloss.Style
	Error  lipgloss.Style
	Info   lipgloss.Style
}

// DefaultStyles returns the default TUI styles
func DefaultStyles() Styles {
	return Styles{
		Sender: lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true),
		Bot:    lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true),
		System: lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true),
		Error:  lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true),
		Info:   lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Italic(true),
	}
}
