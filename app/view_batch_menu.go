package main

import (
	"fmt"
	"strings"
)

// renderBatchEditMenuView renders the batch edit menu where users choose which field to edit
func (m model) renderBatchEditMenuView() string {
	var content strings.Builder

	title := fmt.Sprintf("Edit (%d items)", len(m.batch.selectedItems))
	content.WriteString(m.renderTitleBar(title))

	// Show list of items being edited
	content.WriteString(m.styles.Section.Render("Editing items:") + "\n")

	// Get all tasks from current list to display selected items
	tasks := m.getVisibleTasks()
	selectedCount := 0
	maxDisplay := 5 // Limit display to avoid cluttering the screen

	for _, task := range tasks {
		if m.batch.selectedItems[task.ID] {
			selectedCount++
			if selectedCount <= maxDisplay {
				itemText := fmt.Sprintf("#%d: %s", task.ID, task.Title)
				if len(itemText) > 60 {
					itemText = itemText[:57] + "..."
				}
				content.WriteString(m.styles.Dim.Render("  • "+itemText) + "\n")
			}
		}
	}

	if selectedCount > maxDisplay {
		remaining := selectedCount - maxDisplay
		content.WriteString(m.styles.Dim.Render(fmt.Sprintf("  ... and %d more", remaining)) + "\n")
	}

	content.WriteString("\n")
	content.WriteString("  What would you like to edit?\n\n")

	// Menu options
	options := []struct {
		name string
		desc string
	}{
		{"State", "Change work item state (New, Active, Resolved, etc.)"},
		{"Sprint", "Move items to a specific sprint (Previous, Current, Next, or Backlog)"},
		// Future: Assigned To, Priority, etc.
	}

	for i, opt := range options {
		cursor := "  "
		nameStyle := m.styles.Dim
		if i == m.stateCursor {
			cursor = "→ "
			nameStyle = m.styles.Selected
		}

		content.WriteString(fmt.Sprintf("%s%s\n",
			cursor, nameStyle.Render(opt.name)))
		content.WriteString(fmt.Sprintf("     %s\n\n",
			m.styles.Dim.Render(opt.desc)))
	}

	content.WriteString(m.renderFooter("↑/↓ or j/k: navigate • enter: select • esc: cancel"))

	return content.String()
}
