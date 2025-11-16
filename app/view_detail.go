package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renderDetailView() string {
	if m.selectedTask == nil {
		return "No task selected"
	}

	var content strings.Builder

	// Title bar
	titleText := fmt.Sprintf("Work Item Details")
	content.WriteString(m.renderTitleBar(titleText))

	// Render the card
	content.WriteString(m.buildDetailContent())
	content.WriteString("\n")

	// Footer with keybindings
	keybindings := "←/h/esc: back • r: refresh • e: edit • o: open in browser • s: change state • ?: help • q: quit"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}

func (m model) renderEditView() string {
	var content strings.Builder

	// Title bar
	titleText := "Edit Work Item"
	if m.selectedTask != nil {
		titleText = fmt.Sprintf("Edit Work Item #%d", m.selectedTask.ID)
	}
	content.WriteString(m.renderTitleBar(titleText))

	// Styles
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true).
		Width(15)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	sectionStyle := lipgloss.NewStyle().
		MarginTop(1).
		MarginBottom(1)

	// Show loading spinner if saving
	if m.loading {
		loaderStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			MarginLeft(2)
		content.WriteString(loaderStyle.Render(fmt.Sprintf("%s %s", m.spinner.View(), m.statusMessage)) + "\n\n")

		// Footer with keybindings
		keybindings := "Saving changes..."
		content.WriteString(m.renderFooter(keybindings))

		return content.String()
	}

	// Title field
	content.WriteString(sectionStyle.Render(labelStyle.Render("Title:") + "\n" + m.editTitleInput.View()))
	content.WriteString("\n")
	if m.editFieldCursor == 0 {
		content.WriteString(helpStyle.Render("  Enter the work item title") + "\n")
	}
	content.WriteString("\n")

	// Description field
	content.WriteString(sectionStyle.Render(labelStyle.Render("Description:") + "\n" + m.editDescriptionInput.View()))
	content.WriteString("\n")
	if m.editFieldCursor == 1 {
		content.WriteString(helpStyle.Render("  Multi-line text editor (HTML will be stripped)") + "\n")
	}
	content.WriteString("\n")

	// Footer with keybindings
	keybindings := "tab/shift+tab: switch field • ctrl+s: save • esc: cancel • ?: help"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}
