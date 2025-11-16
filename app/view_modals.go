package main

import (
	"fmt"
	"strings"
)

func (m model) renderStatePickerView() string {
	var content strings.Builder

	// Title bar
	titleText := "Select New State"
	content.WriteString(m.renderTitleBar(titleText))

	if m.selectedTask != nil {
		content.WriteString(fmt.Sprintf("Current state: %s\n\n", m.selectedTask.State))
	}

	for i, state := range m.availableStates {
		cursor := " "
		if m.stateCursor == i {
			cursor = ">"
		}

		line := fmt.Sprintf("%s %s", cursor, state)

		if m.stateCursor == i {
			line = m.styles.Selected.Render(line)
		}

		content.WriteString(line + "\n")
	}

	// Footer with keybindings
	keybindings := "↑/↓ or j/k: navigate • ctrl+u/d or pgup/pgdn: page up/down • enter: select • esc: cancel"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}

func (m model) renderFilterView() string {
	var content strings.Builder

	// Title bar
	titleText := "Filter"
	content.WriteString(m.renderTitleBar(titleText))
	content.WriteString(m.filter.filterInput.View() + "\n\n")

	treeItems := m.getVisibleTreeItems()
	resultCount := len(treeItems)

	// Show result count
	currentTasks := m.getCurrentTasks()
	if m.filter.filterInput.Value() != "" {
		content.WriteString(m.styles.Dim.Render(fmt.Sprintf("  %d/%d", resultCount, len(currentTasks))) + "\n\n")
	} else {
		content.WriteString(m.styles.Dim.Render(fmt.Sprintf("  %d items", len(currentTasks))) + "\n\n")
	}

	// Display results (limit to 15 visible items for performance)
	const maxVisible = 15
	startIdx := 0
	endIdx := min(resultCount, maxVisible)

	// Adjust visible window if cursor is out of view
	if m.ui.cursor >= maxVisible {
		startIdx = m.ui.cursor - maxVisible + 1
		endIdx = m.ui.cursor + 1
	}

	for i := startIdx; i < endIdx && i < resultCount; i++ {
		treeItem := treeItems[i]
		isSelected := m.ui.cursor == i

		line := m.renderTreeItemFilter(treeItem, isSelected)
		content.WriteString(line + "\n")
	}

	// Show scroll indicator if there are more items
	if resultCount > maxVisible {
		if endIdx < resultCount {
			content.WriteString(m.styles.Dim.Render(fmt.Sprintf("  ... %d more ...", resultCount-endIdx)) + "\n")
		}
	}

	// Footer with keybindings
	keybindings := "↑/↓ or ctrl+j/k: navigate • ctrl+u/d or pgup/pgdn: page up/down • enter: open detail • esc: cancel • ?: help"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}

func (m model) renderFindView() string {
	var content strings.Builder

	// Title bar
	titleText := "Find Work Items"
	content.WriteString(m.renderTitleBar(titleText))
	content.WriteString(m.filter.findInput.View() + "\n\n")

	// Note about query behavior
	content.WriteString(m.styles.Hint.Render("Note: Queries all work items assigned to @Me") + "\n")

	// Footer with keybindings
	keybindings := "enter: apply find • esc: cancel"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}

func (m model) renderErrorView() string {
	var content strings.Builder

	// Title bar
	content.WriteString(m.renderTitleBar("Error Details"))

	// Display the error
	content.WriteString(m.styles.Error.Render("An error occurred:") + "\n\n")
	detailStyle := m.styles.Detail.Width(m.ui.width - 4)
	content.WriteString(detailStyle.Render(m.statusMessage) + "\n\n")

	// Instructions
	content.WriteString(m.styles.Hint.Render("This error typically means:") + "\n")
	content.WriteString(m.styles.Hint.Render("1. Missing required permissions in Azure DevOps") + "\n")
	content.WriteString(m.styles.Hint.Render("2. Invalid area path or iteration path") + "\n")
	content.WriteString(m.styles.Hint.Render("3. Work item type not allowed in this project") + "\n\n")

	content.WriteString(m.styles.Hint.Render("Check your Azure DevOps permissions and project settings.") + "\n\n")

	// Footer with keybindings
	keybindings := "esc: back to list • q: quit"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}

func (m model) renderDeleteConfirmView() string {
	var content strings.Builder

	// Title bar
	content.WriteString(m.renderTitleBar("Delete Work Item"))

	// Display the confirmation message
	content.WriteString(m.styles.Warning.Render("⚠ Warning: This action cannot be undone!") + "\n\n")

	// Check if batch delete or single delete
	if len(m.batch.selectedItems) > 0 {
		// Batch delete - show list of items
		content.WriteString(fmt.Sprintf("Are you sure you want to delete %d work items?\n\n", len(m.batch.selectedItems)))

		// Get current tasks and show titles of selected items
		currentTasks := m.getCurrentTasks()
		taskMap := make(map[int]*WorkItem)
		for i := range currentTasks {
			taskMap[currentTasks[i].ID] = &currentTasks[i]
		}

		// List selected items (limit to 10 for readability)
		count := 0
		for itemID := range m.batch.selectedItems {
			if count >= 10 {
				remaining := len(m.batch.selectedItems) - count
				content.WriteString(m.styles.Dim.Render(fmt.Sprintf("  ... and %d more", remaining)) + "\n")
				break
			}
			if task, ok := taskMap[itemID]; ok {
				content.WriteString(m.styles.Dim.Render(fmt.Sprintf("  • #%d - %s", itemID, task.Title)) + "\n")
			} else {
				content.WriteString(m.styles.Dim.Render(fmt.Sprintf("  • #%d", itemID)) + "\n")
			}
			count++
		}

		content.WriteString(m.styles.Value.Render("\nConfirm batch deletion?") + "\n\n")
	} else {
		// Single delete
		content.WriteString("Are you sure you want to delete this work item?\n\n")
		content.WriteString("  " + m.styles.Value.Bold(true).Render(fmt.Sprintf("#%d - %s", m.delete.itemID, m.delete.itemTitle)) + "\n")
		content.WriteString(m.styles.Value.Render("\nConfirm deletion?") + "\n\n")
	}

	content.WriteString("  " + m.styles.Key.Render("[y]") + " Yes, delete it\n")
	content.WriteString("  " + m.styles.Key.Render("[n]") + " No, cancel\n\n")

	// Footer with keybindings
	keybindings := "y: confirm delete • n/esc: cancel"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}
