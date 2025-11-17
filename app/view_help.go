package main

import (
	"strings"
)

func (m model) renderHelpView() string {
	var content strings.Builder

	// Title bar
	titleText := "Keybindings Help"
	content.WriteString(m.renderTitleBar(titleText))

	// Global keybindings
	content.WriteString(m.styles.SectionHeader.Render("Global") + "\n")
	content.WriteString(m.styles.Key.Render("?") + m.styles.Desc.Render("Show/hide this help") + "\n")
	content.WriteString(m.styles.Key.Render("q, ctrl+c") + m.styles.Desc.Render("Quit application") + "\n")
	content.WriteString(m.styles.Key.Render("ctrl+u/d, pgup/pgdn") + m.styles.Desc.Render("Jump half page up/down (works in all views)") + "\n")
	content.WriteString(m.styles.Key.Render("r") + m.styles.Desc.Render("Refresh (all data in list, single item in detail)") + "\n")
	content.WriteString(m.styles.Key.Render("o") + m.styles.Desc.Render("Open current item in browser") + "\n\n")

	// List view keybindings
	content.WriteString(m.styles.SectionHeader.Render("List View") + "\n")
	content.WriteString(m.styles.Key.Render("tab") + m.styles.Desc.Render("Cycle through tabs (sprint or backlog)") + "\n")
	content.WriteString(m.styles.Key.Render("↑/↓, j/k") + m.styles.Desc.Render("Navigate up/down") + "\n")
	content.WriteString(m.styles.Key.Render("space") + m.styles.Desc.Render("Select/deselect item for batch operations") + "\n")
	content.WriteString(m.styles.Key.Render("→/l, enter") + m.styles.Desc.Render("Open item details") + "\n")
	content.WriteString(m.styles.Key.Render("i") + m.styles.Desc.Render("Insert new item before current") + "\n")
	content.WriteString(m.styles.Key.Render("a") + m.styles.Desc.Render("Append new item after current (or as first child if parent)") + "\n")
	content.WriteString(m.styles.Key.Render("d") + m.styles.Desc.Render("Delete current item or selected items (with confirmation)") + "\n")
	content.WriteString(m.styles.Key.Render("e") + m.styles.Desc.Render("Edit current or selected items (shows menu: state, sprint, etc.)") + "\n")
	content.WriteString(m.styles.Key.Render("/") + m.styles.Desc.Render("Filter items in current list") + "\n")
	content.WriteString(m.styles.Key.Render("f") + m.styles.Desc.Render("Find items with dedicated query") + "\n\n")

	// Detail view keybindings
	content.WriteString(m.styles.SectionHeader.Render("Detail View") + "\n")
	content.WriteString(m.styles.Key.Render("←/h, esc, backspace") + m.styles.Desc.Render("Back to list") + "\n")
	content.WriteString(m.styles.Key.Render("e") + m.styles.Desc.Render("Edit item (shows menu: state, sprint, etc.)") + "\n")
	content.WriteString(m.styles.Key.Render("s") + m.styles.Desc.Render("Quick change state (skips menu)") + "\n\n")

	// State picker view keybindings
	content.WriteString(m.styles.SectionHeader.Render("State Picker") + "\n")
	content.WriteString(m.styles.Key.Render("↑/↓, j/k") + m.styles.Desc.Render("Navigate states") + "\n")
	content.WriteString(m.styles.Key.Render("enter") + m.styles.Desc.Render("Select state") + "\n")
	content.WriteString(m.styles.Key.Render("esc") + m.styles.Desc.Render("Cancel") + "\n\n")

	// Filter view keybindings
	content.WriteString(m.styles.SectionHeader.Render("Filter View") + "\n")
	content.WriteString(m.styles.Key.Render("esc") + m.styles.Desc.Render("Cancel filter") + "\n")
	content.WriteString(m.styles.Key.Render("enter") + m.styles.Desc.Render("Open selected item") + "\n")
	content.WriteString(m.styles.Key.Render("↑/↓, ctrl+j/k") + m.styles.Desc.Render("Navigate results") + "\n\n")

	// Create view keybindings
	content.WriteString(m.styles.SectionHeader.Render("Create View") + "\n")
	content.WriteString(m.styles.Key.Render("ctrl+s") + m.styles.Desc.Render("Save new item") + "\n")
	content.WriteString(m.styles.Key.Render("enter") + m.styles.Desc.Render("Show save/cancel hint") + "\n")
	content.WriteString(m.styles.Key.Render("esc") + m.styles.Desc.Render("Cancel creation") + "\n\n")

	// Edit operations
	content.WriteString(m.styles.SectionHeader.Render("Edit Operations") + "\n")
	content.WriteString(m.styles.Key.Render("space") + m.styles.Desc.Render("Select/deselect item (orange bar shows selection)") + "\n")
	content.WriteString(m.styles.Key.Render("e") + m.styles.Desc.Render("Edit item(s) - shows menu for single or multiple items") + "\n")
	content.WriteString(m.styles.Key.Render("d") + m.styles.Desc.Render("Delete item(s) - works for single or multiple items") + "\n\n")

	// Footer with keybindings
	keybindings := "?: close help • esc: close help • q: quit"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}
