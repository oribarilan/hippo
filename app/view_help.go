package main

import (
	"strings"
)

// prepareHelpViewport builds the help content and sets it in the viewport
func (m model) prepareHelpViewport() model {
	var helpContent strings.Builder

	// Global keybindings
	helpContent.WriteString(m.styles.SectionHeader.Render("Global") + "\n")
	helpContent.WriteString(m.styles.Key.Render("?") + m.styles.Desc.Render("Show/hide this help") + "\n")
	helpContent.WriteString(m.styles.Key.Render("q, ctrl+c") + m.styles.Desc.Render("Quit application") + "\n")
	helpContent.WriteString(m.styles.Key.Render("ctrl+u/d, pgup/pgdn") + m.styles.Desc.Render("Jump half page up/down (works in all views)") + "\n")
	helpContent.WriteString(m.styles.Key.Render("r") + m.styles.Desc.Render("Refresh (all data in list, single item in detail)") + "\n")
	helpContent.WriteString(m.styles.Key.Render("o") + m.styles.Desc.Render("Open current item in browser") + "\n\n")

	// List view keybindings
	helpContent.WriteString(m.styles.SectionHeader.Render("List View") + "\n")
	helpContent.WriteString(m.styles.Key.Render("tab") + m.styles.Desc.Render("Cycle through tabs (sprint or backlog)") + "\n")
	helpContent.WriteString(m.styles.Key.Render("↑/↓, j/k") + m.styles.Desc.Render("Navigate up/down") + "\n")
	helpContent.WriteString(m.styles.Key.Render("space") + m.styles.Desc.Render("Select/deselect item for batch operations") + "\n")
	helpContent.WriteString(m.styles.Key.Render("→/l, enter") + m.styles.Desc.Render("Open item details") + "\n")
	helpContent.WriteString(m.styles.Key.Render("i") + m.styles.Desc.Render("Insert new item before current") + "\n")
	helpContent.WriteString(m.styles.Key.Render("a") + m.styles.Desc.Render("Append new item after current (or as first child if parent)") + "\n")
	helpContent.WriteString(m.styles.Key.Render("d") + m.styles.Desc.Render("Delete current item or selected items (with confirmation)") + "\n")
	helpContent.WriteString(m.styles.Key.Render("e") + m.styles.Desc.Render("Edit current or selected items (shows menu: state, sprint, etc.)") + "\n")
	helpContent.WriteString(m.styles.Key.Render("/") + m.styles.Desc.Render("Filter items in current list") + "\n")
	helpContent.WriteString(m.styles.Key.Render("f") + m.styles.Desc.Render("Find items with dedicated query") + "\n\n")

	// Detail view keybindings
	helpContent.WriteString(m.styles.SectionHeader.Render("Detail View") + "\n")
	helpContent.WriteString(m.styles.Key.Render("←/h, esc, backspace") + m.styles.Desc.Render("Back to list") + "\n")
	helpContent.WriteString(m.styles.Key.Render("e") + m.styles.Desc.Render("Edit item (shows menu: state, sprint, etc.)") + "\n")
	helpContent.WriteString(m.styles.Key.Render("s") + m.styles.Desc.Render("Quick change state (skips menu)") + "\n\n")

	// State picker view keybindings
	helpContent.WriteString(m.styles.SectionHeader.Render("State Picker") + "\n")
	helpContent.WriteString(m.styles.Key.Render("↑/↓, j/k") + m.styles.Desc.Render("Navigate states") + "\n")
	helpContent.WriteString(m.styles.Key.Render("enter") + m.styles.Desc.Render("Select state") + "\n")
	helpContent.WriteString(m.styles.Key.Render("esc") + m.styles.Desc.Render("Cancel") + "\n\n")

	// Filter view keybindings
	helpContent.WriteString(m.styles.SectionHeader.Render("Filter View") + "\n")
	helpContent.WriteString(m.styles.Key.Render("esc") + m.styles.Desc.Render("Cancel filter") + "\n")
	helpContent.WriteString(m.styles.Key.Render("enter") + m.styles.Desc.Render("Open selected item") + "\n")
	helpContent.WriteString(m.styles.Key.Render("↑/↓, ctrl+j/k") + m.styles.Desc.Render("Navigate results") + "\n\n")

	// Create view keybindings
	helpContent.WriteString(m.styles.SectionHeader.Render("Create View") + "\n")
	helpContent.WriteString(m.styles.Key.Render("ctrl+s") + m.styles.Desc.Render("Save new item") + "\n")
	helpContent.WriteString(m.styles.Key.Render("enter") + m.styles.Desc.Render("Show save/cancel hint") + "\n")
	helpContent.WriteString(m.styles.Key.Render("esc") + m.styles.Desc.Render("Cancel creation") + "\n\n")

	// Edit operations
	helpContent.WriteString(m.styles.SectionHeader.Render("Edit Operations") + "\n")
	helpContent.WriteString(m.styles.Key.Render("space") + m.styles.Desc.Render("Select/deselect item (orange bar shows selection)") + "\n")
	helpContent.WriteString(m.styles.Key.Render("e") + m.styles.Desc.Render("Edit item(s) - shows menu for single or multiple items") + "\n")
	helpContent.WriteString(m.styles.Key.Render("d") + m.styles.Desc.Render("Delete item(s) - works for single or multiple items") + "\n\n")

	// Configure viewport for help content
	// Title bar takes ~3 lines, footer takes ~4 lines
	helpHeight := m.ui.height - 7
	if helpHeight < 10 {
		helpHeight = 10
	}

	m.viewport.Width = m.ui.width
	m.viewport.Height = helpHeight
	m.viewport.YPosition = 3 // Position after title bar

	// Set viewport content
	m.viewport.SetContent(helpContent.String())
	m.viewport.GotoTop()

	return m
}

func (m model) renderHelpView() string {
	var content strings.Builder

	// Title bar
	titleText := "Keybindings Help"
	content.WriteString(m.renderTitleBar(titleText))

	// Render viewport
	content.WriteString(m.viewport.View())
	content.WriteString("\n")

	// Footer with keybindings and scroll indicator
	scrollInfo := ""
	if m.viewport.TotalLineCount() > 0 {
		scrollPercent := int(m.viewport.ScrollPercent() * 100)
		if scrollPercent < 100 {
			scrollInfo = m.styles.Dim.Render(" • ↑/↓ or j/k: scroll")
		}
	}
	keybindings := "?: close help • esc: close help • q: quit" + scrollInfo
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}
