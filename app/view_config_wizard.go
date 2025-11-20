package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renderConfigWizardView() string {
	var content strings.Builder

	// Title bar - different message based on whether config exists
	var title string
	hasExistingConfig := m.config != nil && (m.config.OrganizationURL != "" || m.config.Project != "")

	if hasExistingConfig {
		title = "Hippo - Reconfigure Settings"
	} else {
		title = "Hippo - No configuration found, please configure first"
	}
	content.WriteString(m.renderTitleBar(title))

	// Check if there's an existing config and show warning
	if hasExistingConfig {
		warningStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorOrange)).
			Bold(true)
		content.WriteString(warningStyle.Render("⚠  Existing configuration will be overwritten") + "\n\n")
	}

	// Instructions
	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorLightGray)).
		Italic(true)
	content.WriteString(instructionStyle.Render("Configure your Azure DevOps connection") + "\n\n")

	// Organization URL field
	content.WriteString(m.styles.EditSection.Render(
		m.styles.EditLabel.Render("Organization URL:") + "\n" +
			"  " + m.wizard.orgInput.View()))
	content.WriteString("\n")
	if m.wizard.fieldCursor == 0 {
		content.WriteString(m.styles.EditHelp.Render("  e.g., https://dev.azure.com/your-org") + "\n")
	}
	content.WriteString("\n")

	// Project field
	content.WriteString(m.styles.EditSection.Render(
		m.styles.EditLabel.Render("Project:") + "\n" +
			"  " + m.wizard.projectInput.View()))
	content.WriteString("\n")
	if m.wizard.fieldCursor == 1 {
		content.WriteString(m.styles.EditHelp.Render("  Your Azure DevOps project name") + "\n")
	}
	content.WriteString("\n")

	// Team field
	content.WriteString(m.styles.EditSection.Render(
		m.styles.EditLabel.Render("Team (optional):") + "\n" +
			"  " + m.wizard.teamInput.View()))
	content.WriteString("\n")
	if m.wizard.fieldCursor == 2 {
		content.WriteString(m.styles.EditHelp.Render("  Defaults to project name if not specified") + "\n")
	}
	content.WriteString("\n")

	// Show validation error if any
	if m.wizard.err != "" {
		content.WriteString(m.styles.Error.Render("✗ "+m.wizard.err) + "\n\n")
	}

	// Footer with keybindings - removed up/down/? since they type characters
	keybindings := "tab/shift+tab: switch field • enter: save • esc: cancel"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}
