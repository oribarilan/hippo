package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the appropriate view based on the current state
func (m model) View() string {
	if m.err != nil {
		// Render a nice error view with instructions
		var content strings.Builder

		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Padding(1, 2)

		hintStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(0, 2)

		content.WriteString("\n")
		content.WriteString(errorStyle.Render("❌ Error"))
		content.WriteString("\n\n")
		content.WriteString(lipgloss.NewStyle().Padding(0, 2).Render(m.err.Error()))
		content.WriteString("\n\n")
		content.WriteString(hintStyle.Render("Press 'q' or 'Ctrl+C' to quit"))
		content.WriteString("\n")

		return content.String()
	}

	switch m.state {
	case loadingView:
		return m.renderLoadingScreen()
	case detailView:
		return m.renderDetailView()
	case statePickerView:
		return m.renderStatePickerView()
	case sprintPickerView:
		return m.renderSprintPickerView()
	case batchEditMenuView:
		return m.renderBatchEditMenuView()
	case filterView:
		return m.renderFilterView()
	case findView:
		return m.renderFindView()
	case helpView:
		return m.renderHelpView()
	case editView:
		return m.renderEditView()
	case createView:
		return m.renderCreateView()
	case errorView:
		return m.renderErrorView()
	case deleteConfirmView:
		return m.renderDeleteConfirmView()
	case moveChildrenConfirmView:
		return m.renderMoveChildrenConfirmView()
	case configWizardView:
		return m.renderConfigWizardView()
	default:
		return m.renderListView()
	}
}

// renderLoadingScreen renders the initial loading screen with centered text
func (m model) renderLoadingScreen() string {
	hippoArt := `⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣀⢤⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⠖⠓⠲⢤⡀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⣀⡾⡭⢤⡄⠈⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⡇⢠⡾⠛⢲⣿⡀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⢹⣿⠁⢸⣿⠀⢻⡄⢀⣠⠤⠴⠶⠶⢦⡤⠤⠒⠒⠒⠒⠦⣤⡀⠀⠀⣸⠀⢸⡇⠀⢸⢿⡇⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠸⣿⣆⠀⢿⣄⣤⠟⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⡤⠴⠦⢭⣷⣶⠃⢀⡞⠀⣠⠋⡼⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠙⢿⣷⢴⡿⠋⠉⠉⠓⠄⠀⠀⠀⠀⠀⠀⠀⡔⠁⠀⠀⠀⠀⠈⠻⣷⣿⠴⢊⡡⠞⠁⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⣿⠁⣀⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠀⠀⠤⣤⣤⡀⠀⠀⠀⢻⡚⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⡿⠚⠛⠻⣧⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢰⠟⠉⠉⠙⢦⠀⠀⠈⢧⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⡼⢁⣴⣶⣤⢸⡄⠀⠀⠀⠀⠀⠀⠀⠀⠀⡟⢀⣾⣿⠷⣬⡇⠀⠀⠈⢳⡄⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⢀⣼⡇⣼⣿⣧⣽⣾⠃⠀⠀⠀⠀⠀⠀⠀⠀⠀⣿⠈⣿⣷⣴⣿⡟⠀⠀⠀⠀⠹⡆⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⡾⠁⣳⣙⣿⣿⣿⣃⡀⠀⠀⠀⠀⠀⣀⣀⡀⠀⠀⠳⣝⣿⡿⠟⠳⠀⠀⠀⠀⠀⢻⡀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⢸⡇⠀⣠⠿⠏⠉⠉⠉⠉⠉⠙⡿⠛⠉⠉⠀⠉⠁⠀⠀⠜⠁⠀⠀⠀⠀⠀⠀⠀⠀⠸⡇⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⣧⡾⠁⢠⣿⠙⣆⠀⠀⠀⡼⠁⠀⠀⠀⠀⡴⢋⣿⣷⠀⠀⠀⢀⣶⡄⠀⠀⠀⠀⠀⣷⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⣼⠁⠀⠈⠻⣿⡿⠀⠀⢠⡇⠀⠀⠀⠀⠐⢳⠿⠛⠉⠀⠀⠀⢸⣿⠃⠀⠀⠀⠀⢠⡟⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⣿⠀⠀⠀⠀⠈⠁⠀⠀⢸⡇⠀⠀⠀⠀⠀⢀⠀⠀⠀⠀⠀⠀⠈⡇⠀⠀⠀⠀⢠⡾⠁⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠸⣆⠀⠀⠀⠀⠀⠋⠀⠘⣇⠀⠀⠀⠂⠀⠈⠀⠀⠀⠀⠀⠀⢸⠃⠀⠀⣠⡴⠋⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠙⣦⡀⠈⠁⠀⢀⣀⣶⣦⣤⣀⠀⠀⠉⠀⠐⠂⠀⠀⢀⣰⠏⢀⣤⣾⡉⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⡿⣷⡒⠋⠉⠁⠀⣀⣀⠈⠙⠓⠶⠤⠤⠤⠴⠖⣋⣥⠞⠋⢸⠏⣇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣸⣇⠈⠙⠓⠒⠚⠋⠉⠉⠙⠲⢤⣄⣀⣀⣤⠴⠚⠯⠀⠀⣠⠏⢀⡿⣄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣯⠙⢦⣀⠀⠀⠀⠀⠀⠀⠀⠀⠉⠀⠀⠀⠀⠀⠀⠀⠀⠞⠁⢀⣾⠁⠈⠳⡄⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⣸⠛⢧⡀⠉⠓⠦⢤⣀⣀⣀⣀⠀⠀⠀⠀⣀⣀⣀⠄⠀⠀⠀⢠⡞⠁⢠⠀⠀⠹⡄⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⡇⠀⠈⠻⢦⡀⠀⠀⠀⠀⠀⠉⠉⠉⠉⠉⠁⠀⠀⠀⢀⡠⠖⠉⠀⠀⢸⡇⠀⠀⢹⡀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⡇⠀⠀⠀⠀⠙⣶⣤⣀⠀⠀⠀⠀⠀⣀⣠⠀⠀⡀⠘⠉⠀⠀⠀⠀⠀⣸⡇⠀⠀⣀⣷⠀⠀⠀⠀⢀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⢳⡀⠀⠀⠀⠀⠸⣏⠈⠉⠙⠉⠉⠉⠁⠀⠀⣸⠇⠀⠀⠀⠀⠀⠀⢠⣿⠇⠀⠀⠀⠙⢧⠀⠀⢀⠞⠂
⠀⠀⠀⠀⠀⠀⠀⠀⣀⠼⣷⠀⠀⠀⠀⠀⢻⣧⠀⠀⠀⠀⠀⠀⠀⣰⠿⠃⠀⠀⠀⠀⢀⣴⣿⡟⠀⠀⠀⠀⠀⠀⢷⠊⠁⡼⠁
⠀⣠⠴⠒⣦⣴⣶⡋⠁⠀⢨⠀⠀⠀⠀⠀⠀⠹⣷⡄⠀⠀⠀⠀⣰⠏⠀⠀⠀⠀⠀⠀⢋⣿⡟⠀⢀⣠⣤⣤⣤⣴⣿⣶⠋⠀⠀
⡾⢿⠒⠋⠀⢯⣙⣷⡀⠀⠘⣆⠀⠀⠀⠀⠀⠐⢹⡟⠒⠲⠖⠊⣿⡀⠀⠀⠀⠀⠀⠀⣼⠟⠀⢰⡿⢋⡽⠋⠈⠧⣍⣻⡇⠀⠀
⣇⣸⠀⠀⠀⠀⠈⠉⠳⡀⠀⢸⡄⠀⢀⣀⣀⡀⠀⣧⠦⠀⠀⢀⣿⠁⢠⣄⣤⠀⠀⢠⡟⠀⣠⠛⠋⠉⠀⠀⠀⠀⢀⢼⣇⠀⠀
⠙⢧⡀⠀⠀⠀⠀⠀⠀⢹⣆⣸⠇⠐⠛⠉⠉⠀⠀⠹⣿⠀⢠⣾⠃⠈⠁⠀⠉⠀⠀⠸⡇⢠⠇⠀⠀⠀⠀⠀⠀⠀⣇⣾⠏⠀⠀
⠀⠈⠙⢦⡀⠀⠀⠀⢀⣼⣿⠷⠶⣄⠀⣀⣀⡀⢀⡖⠻⣦⣼⣧⠤⢄⠀⣠⣤⣄⣠⠞⣷⣾⠀⠀⠀⠀⠀⠀⢀⡴⠛⠉⠀⠀⠀
⠀⠀⠀⠀⠉⠓⠒⠒⠛⠁⠘⠦⣤⣼⣶⣁⣀⣹⡾⠤⠚⠁⠘⠧⣤⡼⠶⣇⣠⠼⠟⠛⠉⠀⠳⠤⣤⣤⠿⠟⠋⠀⠀⠀⠀⠀⠀`

	// Calculate vertical centering
	artLines := strings.Split(hippoArt, "\n")
	totalHeight := 3 + len(artLines) // spinner line + blank + art
	verticalPadding := (m.ui.height - totalHeight) / 2
	if verticalPadding < 0 {
		verticalPadding = 0
	}

	// Style for text
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("212")).
		Bold(true).
		Width(m.ui.width).
		Align(lipgloss.Center)

	// Style the ASCII art
	artStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("212")).
		Width(m.ui.width).
		Align(lipgloss.Center)

	// Build the centered content
	var content strings.Builder

	// Add vertical padding
	for i := 0; i < verticalPadding; i++ {
		content.WriteString("\n")
	}

	// Add the "Hippo" text with spinners on the sides
	spinnerView := m.spinner.View()
	titleLine := fmt.Sprintf("%s  Hippo  %s", spinnerView, spinnerView)
	content.WriteString(titleStyle.Render(titleLine) + "\n")

	content.WriteString("\n")

	// Add the ASCII art hippo
	for _, line := range artLines {
		content.WriteString(artStyle.Render(line) + "\n")
	}

	return content.String()
}

// renderLogLine renders the action log line if there is one
func (m model) renderLogLine() string {
	if m.lastActionLog == "" {
		return ""
	}

	timestamp := m.lastActionTime.Format("15:04:05")
	return m.styles.Log.Render(fmt.Sprintf("[%s] %s", timestamp, m.lastActionLog))
}

// renderTitleBar renders the title bar with the given title text
func (m model) renderTitleBar(title string) string {
	titleBarStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(ColorGray)).
		Foreground(lipgloss.Color(ColorWhite)).
		Bold(true).
		Width(m.ui.width).
		Padding(0, 1)

	// Calculate padding to align version to the right
	versionText := Version
	availableWidth := m.ui.width - len(title) - len(versionText) - 4 // 4 for padding (2 on each side)
	if availableWidth < 0 {
		availableWidth = 0
	}
	padding := strings.Repeat(" ", availableWidth)

	titleWithVersion := title + padding + versionText
	return titleBarStyle.Render(titleWithVersion) + "\n\n"
}

// renderFooter renders the bottom section with action log and keybindings
func (m model) renderFooter(keybindings string) string {
	var footer strings.Builder

	// Action log line
	footer.WriteString("\n")
	if m.lastActionLog != "" {
		footer.WriteString(m.renderLogLine() + "\n")
	}

	// Separator line
	separatorStyle := m.styles.Separator.Width(m.ui.width)
	separator := separatorStyle.Render(strings.Repeat("─", m.ui.width))

	footer.WriteString(separator + "\n")
	footer.WriteString(m.styles.Help.Render(keybindings))

	// Config source bar
	footer.WriteString(m.renderConfigBar())

	return footer.String()
}

// renderConfigBar renders the configuration source information bar
func (m model) renderConfigBar() string {
	if m.configSource == nil || m.config == nil {
		return ""
	}

	var parts []string

	// Organization URL
	if m.config.OrganizationURL != "" && m.configSource.OrganizationURL != "" {
		// Shorten URL for display (remove https://dev.azure.com/ prefix if present)
		displayURL := m.config.OrganizationURL
		if len(displayURL) > 40 {
			displayURL = displayURL[:37] + "..."
		}
		parts = append(parts, fmt.Sprintf("Org:%s", displayURL))
	}

	// Project
	if m.config.Project != "" && m.configSource.Project != "" {
		parts = append(parts, fmt.Sprintf("Proj:%s", m.config.Project))
	}

	// Team
	if m.config.Team != "" && m.configSource.Team != "" {
		parts = append(parts, fmt.Sprintf("Team:%s", m.config.Team))
	}

	// Source information
	sourceInfo := buildSourceInfo(m.configSource)
	if sourceInfo != "" {
		parts = append(parts, sourceInfo)
	}

	if len(parts) == 0 {
		return ""
	}

	configInfo := strings.Join(parts, " • ")

	configBarStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(ColorGray)).
		Foreground(lipgloss.Color(ColorWhite)).
		Width(m.ui.width).
		Padding(0, 1)

	return "\n" + configBarStyle.Render(configInfo)
}

// buildSourceInfo builds the source information string
func buildSourceInfo(source *ConfigSource) string {
	if source == nil {
		return ""
	}

	// Determine the primary source(s)
	sources := make(map[string]bool)
	if source.OrganizationURL != "" {
		sources[source.OrganizationURL] = true
	}
	if source.Project != "" {
		sources[source.Project] = true
	}
	if source.Team != "" {
		sources[source.Team] = true
	}

	// Build source display
	var sourceDesc string

	// Check if all from same source
	if len(sources) == 1 {
		for src := range sources {
			switch src {
			case "flag":
				sourceDesc = "Source:⚑arguments"
			case "env":
				sourceDesc = "Source:$env"
			case "file":
				if source.ConfigPath != "" {
					displayPath := abbreviateHomePath(source.ConfigPath)
					sourceDesc = fmt.Sprintf("Source:%s", displayPath)
				} else {
					sourceDesc = "Source:file"
				}
			}
		}
	} else {
		// Mixed sources - show which fields come from where
		var srcParts []string
		if source.OrganizationURL == "flag" || source.Project == "flag" || source.Team == "flag" {
			srcParts = append(srcParts, "⚑args")
		}
		if source.OrganizationURL == "env" || source.Project == "env" || source.Team == "env" {
			srcParts = append(srcParts, "$env")
		}
		if source.OrganizationURL == "file" || source.Project == "file" || source.Team == "file" {
			if source.ConfigPath != "" {
				displayPath := abbreviateHomePath(source.ConfigPath)
				srcParts = append(srcParts, displayPath)
			} else {
				srcParts = append(srcParts, "file")
			}
		}
		if len(srcParts) > 0 {
			sourceDesc = "Source:" + strings.Join(srcParts, "+")
		}
	}

	return sourceDesc
}

// abbreviateHomePath replaces home directory with ~
func abbreviateHomePath(path string) string {
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(path, home) {
		return strings.Replace(path, home, "~", 1)
	}
	return path
}
