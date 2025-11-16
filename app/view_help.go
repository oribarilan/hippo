package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renderHelpView() string {
	var content strings.Builder

	// Title bar
	titleText := "Keybindings Help"
	content.WriteString(m.renderTitleBar(titleText))

	// Styles
	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		MarginTop(1).
		MarginBottom(1)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true).
		Width(20)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230"))

	// Global keybindings
	content.WriteString(sectionStyle.Render("Global") + "\n")
	content.WriteString(keyStyle.Render("?") + descStyle.Render("Show/hide this help") + "\n")
	content.WriteString(keyStyle.Render("q, ctrl+c") + descStyle.Render("Quit application") + "\n")
	content.WriteString(keyStyle.Render("ctrl+u/d, pgup/pgdn") + descStyle.Render("Jump half page up/down (works in all views)") + "\n")
	content.WriteString(keyStyle.Render("r") + descStyle.Render("Refresh (all data in list, single item in detail)") + "\n")
	content.WriteString(keyStyle.Render("o") + descStyle.Render("Open current item in browser") + "\n\n")

	// List view keybindings
	content.WriteString(sectionStyle.Render("List View") + "\n")
	content.WriteString(keyStyle.Render("tab") + descStyle.Render("Cycle through tabs (sprint or backlog)") + "\n")
	content.WriteString(keyStyle.Render("↑/↓, j/k") + descStyle.Render("Navigate up/down") + "\n")
	content.WriteString(keyStyle.Render("space") + descStyle.Render("Select/deselect item for batch operations") + "\n")
	content.WriteString(keyStyle.Render("→/l, enter") + descStyle.Render("Open item details") + "\n")
	content.WriteString(keyStyle.Render("i") + descStyle.Render("Insert new item before current") + "\n")
	content.WriteString(keyStyle.Render("a") + descStyle.Render("Append new item after current (or as first child if parent)") + "\n")
	content.WriteString(keyStyle.Render("d") + descStyle.Render("Delete current item or selected items (with confirmation)") + "\n")
	content.WriteString(keyStyle.Render("s") + descStyle.Render("Change state of selected items (batch operation)") + "\n")
	content.WriteString(keyStyle.Render("/") + descStyle.Render("Filter items in current list") + "\n")
	content.WriteString(keyStyle.Render("f") + descStyle.Render("Find items with dedicated query") + "\n\n")

	// Detail view keybindings
	content.WriteString(sectionStyle.Render("Detail View") + "\n")
	content.WriteString(keyStyle.Render("←/h, esc, backspace") + descStyle.Render("Back to list") + "\n")
	content.WriteString(keyStyle.Render("e") + descStyle.Render("Edit item") + "\n")
	content.WriteString(keyStyle.Render("s") + descStyle.Render("Change item state") + "\n\n")

	// State picker view keybindings
	content.WriteString(sectionStyle.Render("State Picker") + "\n")
	content.WriteString(keyStyle.Render("↑/↓, j/k") + descStyle.Render("Navigate states") + "\n")
	content.WriteString(keyStyle.Render("enter") + descStyle.Render("Select state") + "\n")
	content.WriteString(keyStyle.Render("esc") + descStyle.Render("Cancel") + "\n\n")

	// Filter view keybindings
	content.WriteString(sectionStyle.Render("Filter View") + "\n")
	content.WriteString(keyStyle.Render("esc") + descStyle.Render("Cancel filter") + "\n")
	content.WriteString(keyStyle.Render("enter") + descStyle.Render("Open selected item") + "\n")
	content.WriteString(keyStyle.Render("↑/↓, ctrl+j/k") + descStyle.Render("Navigate results") + "\n\n")

	// Create view keybindings
	content.WriteString(sectionStyle.Render("Create View") + "\n")
	content.WriteString(keyStyle.Render("ctrl+s") + descStyle.Render("Save new item") + "\n")
	content.WriteString(keyStyle.Render("enter") + descStyle.Render("Show save/cancel hint") + "\n")
	content.WriteString(keyStyle.Render("esc") + descStyle.Render("Cancel creation") + "\n\n")

	// Batch operations
	content.WriteString(sectionStyle.Render("Batch Operations") + "\n")
	content.WriteString(keyStyle.Render("space") + descStyle.Render("Select/deselect item (orange bar shows selection)") + "\n")
	content.WriteString(keyStyle.Render("d") + descStyle.Render("Delete all selected items (with confirmation)") + "\n")
	content.WriteString(keyStyle.Render("s") + descStyle.Render("Change state of all selected items") + "\n\n")

	// Footer with keybindings
	keybindings := "?: close help • esc: close help • q: quit"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}
