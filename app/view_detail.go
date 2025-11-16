package main

import (
	"fmt"
	"strings"
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

	// Show loading spinner if saving
	if m.loading {
		content.WriteString(m.styles.Loader.Render(fmt.Sprintf("%s %s", m.spinner.View(), m.statusMessage)) + "\n\n")

		// Footer with keybindings
		keybindings := "Saving changes..."
		content.WriteString(m.renderFooter(keybindings))

		return content.String()
	}

	// Title field
	content.WriteString(m.styles.EditSection.Render(m.styles.EditLabel.Render("Title:") + "\n" + m.edit.titleInput.View()))
	content.WriteString("\n")
	if m.edit.fieldCursor == 0 {
		content.WriteString(m.styles.EditHelp.Render("  Enter the work item title") + "\n")
	}
	content.WriteString("\n")

	// Description field
	content.WriteString(m.styles.EditSection.Render(m.styles.EditLabel.Render("Description:") + "\n" + m.edit.descriptionInput.View()))
	content.WriteString("\n")
	if m.edit.fieldCursor == 1 {
		content.WriteString(m.styles.EditHelp.Render("  Multi-line text editor (HTML will be stripped)") + "\n")
	}
	content.WriteString("\n")

	// Footer with keybindings
	keybindings := "tab/shift+tab: switch field • ctrl+s: save • esc: cancel • ?: help"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}
