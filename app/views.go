package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the appropriate view based on the current state
func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n", m.err)
	}

	switch m.state {
	case loadingView:
		return m.renderLoadingScreen()
	case detailView:
		return m.renderDetailView()
	case statePickerView:
		return m.renderStatePickerView()
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
		Background(lipgloss.Color(ColorPurple)).
		Foreground(lipgloss.Color(ColorWhite)).
		Bold(true).
		Width(m.ui.width).
		Padding(0, 1)

	// Calculate padding to align version to the right
	versionText := version
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

	return footer.String()
}
