package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// wizardField represents a field in the configuration wizard
type wizardField int

const (
	wizardFieldOrg wizardField = iota
	wizardFieldProject
	wizardFieldTeam
	wizardFieldConfirm
)

// wizardModel is the Bubbletea model for the configuration wizard
type wizardModel struct {
	currentField   wizardField
	orgInput       textinput.Model
	projectInput   textinput.Model
	teamInput      textinput.Model
	existingConfig *Config
	err            string
	styles         Styles
	confirmed      bool
	cancelled      bool
}

func initialWizardModel(existingConfig *Config) wizardModel {
	// Organization URL input
	orgInput := textinput.New()
	orgInput.Placeholder = "https://dev.azure.com/your-org"
	orgInput.Focus()
	orgInput.CharLimit = 200
	orgInput.Width = 60
	if existingConfig != nil {
		orgInput.SetValue(existingConfig.OrganizationURL)
	}

	// Project input
	projectInput := textinput.New()
	projectInput.Placeholder = "MyProject"
	projectInput.CharLimit = 100
	projectInput.Width = 60
	if existingConfig != nil {
		projectInput.SetValue(existingConfig.Project)
	}

	// Team input
	teamInput := textinput.New()
	teamInput.Placeholder = "MyTeam (optional, defaults to project name)"
	teamInput.CharLimit = 100
	teamInput.Width = 60
	if existingConfig != nil {
		teamInput.SetValue(existingConfig.Team)
	}

	return wizardModel{
		currentField:   wizardFieldOrg,
		orgInput:       orgInput,
		projectInput:   projectInput,
		teamInput:      teamInput,
		existingConfig: existingConfig,
		styles:         NewStyles(),
	}
}

func (m wizardModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			// Cancel wizard
			m.cancelled = true
			return m, tea.Quit

		case "enter", "tab", "down":
			// Move to next field or confirm
			if m.currentField == wizardFieldConfirm {
				// Validate and save
				orgURL := strings.TrimSpace(m.orgInput.Value())
				project := strings.TrimSpace(m.projectInput.Value())
				team := strings.TrimSpace(m.teamInput.Value())

				// Validate organization URL
				if orgURL == "" {
					m.err = "Organization URL is required"
					m.currentField = wizardFieldOrg
					m.orgInput.Focus()
					return m, nil
				}
				if !strings.HasPrefix(orgURL, "https://") {
					m.err = "URL must start with https://"
					m.currentField = wizardFieldOrg
					m.orgInput.Focus()
					return m, nil
				}

				// Validate project
				if project == "" {
					m.err = "Project name is required"
					m.currentField = wizardFieldProject
					m.projectInput.Focus()
					return m, nil
				}

				// Create config
				config := &Config{
					ConfigVersion:   CurrentConfigVersion,
					OrganizationURL: orgURL,
					Project:         project,
					Team:            team,
				}

				// Save config
				if err := SaveConfig(config); err != nil {
					m.err = fmt.Sprintf("Failed to save configuration: %v", err)
					return m, nil
				}

				m.confirmed = true
				return m, tea.Quit
			}

			// Move to next field
			m.err = ""
			m.currentField++
			m.updateFocus()
			return m, nil

		case "shift+tab", "up":
			// Move to previous field
			if m.currentField > wizardFieldOrg {
				m.err = ""
				m.currentField--
				m.updateFocus()
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		// Handle window resize if needed
		return m, nil
	}

	// Update the active input
	switch m.currentField {
	case wizardFieldOrg:
		m.orgInput, cmd = m.orgInput.Update(msg)
	case wizardFieldProject:
		m.projectInput, cmd = m.projectInput.Update(msg)
	case wizardFieldTeam:
		m.teamInput, cmd = m.teamInput.Update(msg)
	}

	return m, cmd
}

func (m *wizardModel) updateFocus() {
	m.orgInput.Blur()
	m.projectInput.Blur()
	m.teamInput.Blur()

	switch m.currentField {
	case wizardFieldOrg:
		m.orgInput.Focus()
	case wizardFieldProject:
		m.projectInput.Focus()
	case wizardFieldTeam:
		m.teamInput.Focus()
	}
}

func (m wizardModel) View() string {
	var content strings.Builder

	// Title bar
	titleStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(ColorPurple)).
		Foreground(lipgloss.Color(ColorWhite)).
		Bold(true).
		Padding(0, 2).
		Width(80)

	content.WriteString(titleStyle.Render("Hippo Configuration Wizard"))
	content.WriteString("\n\n")

	// Show existing config warning if exists
	if m.existingConfig != nil && m.existingConfig.ConfigVersion > 0 {
		warningStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorOrange)).
			Bold(true)

		content.WriteString(warningStyle.Render("⚠ Configuration file already exists"))
		content.WriteString("\n")

		detailStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorLightGray)).
			Italic(true)

		content.WriteString(detailStyle.Render("  Current settings will be overwritten if you continue."))
		content.WriteString("\n\n")

		// Show current config
		configStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorGray)).
			PaddingLeft(2)

		content.WriteString(configStyle.Render(fmt.Sprintf("Organization: %s", m.existingConfig.OrganizationURL)))
		content.WriteString("\n")
		content.WriteString(configStyle.Render(fmt.Sprintf("Project:      %s", m.existingConfig.Project)))
		content.WriteString("\n")
		content.WriteString(configStyle.Render(fmt.Sprintf("Team:         %s", m.existingConfig.Team)))
		content.WriteString("\n\n")
	}

	// Instructions
	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorLightGray))

	content.WriteString(instructionStyle.Render("Please provide your Azure DevOps configuration:"))
	content.WriteString("\n\n")

	// Organization URL field
	m.renderField(&content, "Organization URL", m.orgInput, m.currentField == wizardFieldOrg,
		"e.g., https://dev.azure.com/your-org")

	// Project field
	m.renderField(&content, "Project Name", m.projectInput, m.currentField == wizardFieldProject,
		"The name of your Azure DevOps project")

	// Team field
	m.renderField(&content, "Team Name", m.teamInput, m.currentField == wizardFieldTeam,
		"Optional - defaults to project name")

	// Confirmation
	content.WriteString("\n")
	if m.currentField == wizardFieldConfirm {
		confirmStyle := lipgloss.NewStyle().
			Background(lipgloss.Color(ColorGreen)).
			Foreground(lipgloss.Color(ColorWhite)).
			Bold(true).
			Padding(0, 2)

		content.WriteString(confirmStyle.Render("▶ Save Configuration"))
	} else {
		confirmStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorGray)).
			Padding(0, 2)

		content.WriteString(confirmStyle.Render("  Save Configuration"))
	}
	content.WriteString("\n\n")

	// Error message
	if m.err != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorRed)).
			Bold(true)

		content.WriteString(errorStyle.Render(fmt.Sprintf("✗ %s", m.err)))
		content.WriteString("\n\n")
	}

	// Footer with help
	separatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorGray))

	content.WriteString(separatorStyle.Render(strings.Repeat("─", 80)))
	content.WriteString("\n")

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorGray))

	helpText := []string{
		m.styles.Key.Render("↑/↓, tab/shift+tab") + " navigate",
		m.styles.Key.Render("enter") + " next/save",
		m.styles.Key.Render("esc") + " cancel",
	}

	content.WriteString(helpStyle.Render(strings.Join(helpText, "  •  ")))
	content.WriteString("\n")

	return content.String()
}

func (m wizardModel) renderField(content *strings.Builder, label string, input textinput.Model, isFocused bool, hint string) {
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorGreen)).
		Bold(true).
		Width(20)

	if isFocused {
		labelStyle = labelStyle.Foreground(lipgloss.Color(ColorPurple))
	}

	content.WriteString(labelStyle.Render(label + ":"))
	content.WriteString(" ")
	content.WriteString(input.View())
	content.WriteString("\n")

	if isFocused && hint != "" {
		hintStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorGray)).
			Italic(true).
			PaddingLeft(22)

		content.WriteString(hintStyle.Render(hint))
		content.WriteString("\n")
	}

	content.WriteString("\n")
}

// RunConfigWizard runs the interactive configuration wizard using Bubbletea
func RunConfigWizard() error {
	// Check if config file already exists
	configPath, err := GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	existingConfig, _ := loadConfigFile(configPath)

	// Run the wizard
	wm := initialWizardModel(existingConfig)
	p := tea.NewProgram(wm)

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run wizard: %w", err)
	}

	// Check result
	if m, ok := finalModel.(wizardModel); ok {
		if m.cancelled {
			return fmt.Errorf("setup cancelled")
		}

		if !m.confirmed {
			return fmt.Errorf("setup incomplete")
		}
	}

	return nil
}
