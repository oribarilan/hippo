package main

import (
	"fmt"
	"strings"
)

// renderSprintPickerView renders the sprint selection view for batch operations
func (m model) renderSprintPickerView() string {
	var content strings.Builder

	// Title bar
	count := len(m.batch.selectedItems)
	titleText := fmt.Sprintf("Move to Sprint (%d items)", count)
	content.WriteString(m.renderTitleBar(titleText))

	// Show list of items being edited
	if count > 0 {
		content.WriteString(m.styles.Section.Render("Moving items:") + "\n")

		// Get all tasks from current list to display selected items
		tasks := m.getVisibleTasks()
		selectedCount := 0
		maxDisplay := 3 // Limit display to avoid cluttering the screen

		for _, task := range tasks {
			if m.batch.selectedItems[task.ID] {
				selectedCount++
				if selectedCount <= maxDisplay {
					itemText := fmt.Sprintf("#%d: %s", task.ID, task.Title)
					if len(itemText) > 60 {
						itemText = itemText[:57] + "..."
					}
					// Extract sprint name from iteration path (e.g., "Project\\Sprint 1" -> "Sprint 1")
					sprintName := "Backlog"
					if task.IterationPath != "" {
						parts := strings.Split(task.IterationPath, "\\")
						if len(parts) > 0 {
							sprintName = parts[len(parts)-1]
						}
					}
					sprintInfo := fmt.Sprintf("%s → ?", sprintName)
					content.WriteString(m.styles.Dim.Render(fmt.Sprintf("  • %s [%s]", itemText, sprintInfo)) + "\n")
				}
			}
		}

		if selectedCount > maxDisplay {
			remaining := selectedCount - maxDisplay
			content.WriteString(m.styles.Dim.Render(fmt.Sprintf("  ... and %d more", remaining)) + "\n")
		}

		content.WriteString("\n")
	}

	content.WriteString("  Select target sprint:\n\n")

	// Build sprint options list
	options := []struct {
		name string
		path string
	}{
		{"Backlog (no sprint)", ""},
	}

	// Add previous sprint if available
	if m.sprints[previousSprint] != nil {
		options = append(options, struct {
			name string
			path string
		}{
			name: fmt.Sprintf("Previous Sprint - %s", m.sprints[previousSprint].Name),
			path: m.sprints[previousSprint].Path,
		})
	}

	// Add current sprint if available
	if m.sprints[currentSprint] != nil {
		options = append(options, struct {
			name string
			path string
		}{
			name: fmt.Sprintf("Current Sprint - %s", m.sprints[currentSprint].Name),
			path: m.sprints[currentSprint].Path,
		})
	}

	// Add next sprint if available
	if m.sprints[nextSprint] != nil {
		options = append(options, struct {
			name string
			path string
		}{
			name: fmt.Sprintf("Next Sprint - %s", m.sprints[nextSprint].Name),
			path: m.sprints[nextSprint].Path,
		})
	}

	// Render options
	for i, opt := range options {
		cursor := " "
		if m.stateCursor == i {
			cursor = ">"
		}

		line := fmt.Sprintf("%s %s", cursor, opt.name)

		if m.stateCursor == i {
			line = m.styles.Selected.Render(line)
		}

		content.WriteString(line + "\n")
	}

	// Footer with keybindings
	keybindings := "↑/↓ or j/k: navigate • enter: select • esc: cancel"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}
