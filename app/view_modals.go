package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renderStatePickerView() string {
	var content strings.Builder

	// Title bar
	titleText := "Select New State"
	content.WriteString(m.renderTitleBar(titleText))

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Bold(true)

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
			line = selectedStyle.Render(line)
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
	content.WriteString(m.filterInput.View() + "\n\n")

	treeItems := m.getVisibleTreeItems()
	resultCount := len(treeItems)

	// Show result count
	currentTasks := m.getCurrentTasks()
	if m.filterInput.Value() != "" {
		content.WriteString(m.styles.Dim.Render(fmt.Sprintf("  %d/%d", resultCount, len(currentTasks))) + "\n\n")
	} else {
		content.WriteString(m.styles.Dim.Render(fmt.Sprintf("  %d items", len(currentTasks))) + "\n\n")
	}

	// Display results (limit to 15 visible items for performance)
	const maxVisible = 15
	startIdx := 0
	endIdx := min(resultCount, maxVisible)

	// Adjust visible window if cursor is out of view
	if m.cursor >= maxVisible {
		startIdx = m.cursor - maxVisible + 1
		endIdx = m.cursor + 1
	}

	for i := startIdx; i < endIdx && i < resultCount; i++ {
		treeItem := treeItems[i]
		isSelected := m.cursor == i

		cursor := "  "
		if isSelected {
			cursor = "❯ "
		}

		// Get tree drawing prefix and color it
		var treePrefix string
		if isSelected {
			treePrefix = m.styles.TreeEdgeSelected.Render(getTreePrefix(treeItem))
		} else {
			treePrefix = m.styles.TreeEdge.Render(getTreePrefix(treeItem))
		}

		// Get work item type icon
		icon := getWorkItemIcon(treeItem.WorkItem.WorkItemType)
		var styledIcon string
		if isSelected {
			styledIcon = m.styles.IconSelected.Render(icon)
		} else {
			styledIcon = m.styles.Icon.Render(icon)
		}

		// Get state category to determine styling
		category := m.getStateCategory(treeItem.WorkItem.State)

		// Get styles based on category and selection
		stateStyle := m.styles.GetStateStyle(category, isSelected)
		itemTitleStyle := m.styles.GetItemTitleStyle(category, isSelected, len(treeItem.WorkItem.Children) > 0)

		// Apply the styling
		taskTitle := itemTitleStyle.Render(treeItem.WorkItem.Title)
		state := stateStyle.Render(treeItem.WorkItem.State)

		// Build the line with icon
		var line string
		if isSelected {
			// Apply background to cursor and spacing too
			cursorStyled := m.styles.Selected.Render(cursor)
			spacer := m.styles.Selected.Render(" ")
			line = fmt.Sprintf("%s%s%s%s%s%s",
				cursorStyled,
				treePrefix,
				styledIcon,
				spacer,
				taskTitle,
				spacer+state)
		} else {
			line = fmt.Sprintf("%s%s%s %s %s",
				cursor,
				treePrefix,
				styledIcon,
				taskTitle,
				state)
		}

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
	content.WriteString(m.findInput.View() + "\n\n")

	// Note about query behavior
	noteStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)
	content.WriteString(noteStyle.Render("Note: Queries all work items assigned to @Me") + "\n")

	// Footer with keybindings
	keybindings := "enter: apply find • esc: cancel"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}

func (m model) renderErrorView() string {
	var content strings.Builder

	// Title bar
	content.WriteString(m.renderTitleBar("Error Details"))

	// Error message style
	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		MarginBottom(1)

	detailStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Width(m.width - 4).
		PaddingLeft(2)

	// Display the error
	content.WriteString(errorStyle.Render("An error occurred:") + "\n\n")
	content.WriteString(detailStyle.Render(m.statusMessage) + "\n\n")

	// Instructions
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	content.WriteString(helpStyle.Render("This error typically means:") + "\n")
	content.WriteString(helpStyle.Render("1. Missing required permissions in Azure DevOps") + "\n")
	content.WriteString(helpStyle.Render("2. Invalid area path or iteration path") + "\n")
	content.WriteString(helpStyle.Render("3. Work item type not allowed in this project") + "\n\n")

	content.WriteString(helpStyle.Render("Check your Azure DevOps permissions and project settings.") + "\n\n")

	// Footer with keybindings
	keybindings := "esc: back to list • q: quit"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}

func (m model) renderDeleteConfirmView() string {
	var content strings.Builder

	// Title bar
	content.WriteString(m.renderTitleBar("Delete Work Item"))

	// Warning style
	warningStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		MarginBottom(1)

	itemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Bold(true)

	questionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		MarginTop(1).
		MarginBottom(1)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true)

	// Display the confirmation message
	content.WriteString(warningStyle.Render("⚠ Warning: This action cannot be undone!") + "\n\n")

	// Check if batch delete or single delete
	if len(m.selectedItems) > 0 {
		// Batch delete - show list of items
		content.WriteString(fmt.Sprintf("Are you sure you want to delete %d work items?\n\n", len(m.selectedItems)))

		// Get current tasks and show titles of selected items
		currentTasks := m.getCurrentTasks()
		taskMap := make(map[int]*WorkItem)
		for i := range currentTasks {
			taskMap[currentTasks[i].ID] = &currentTasks[i]
		}

		// List selected items (limit to 10 for readability)
		listStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("248"))
		count := 0
		maxDisplay := 10
		for itemID := range m.selectedItems {
			if count >= maxDisplay {
				remaining := len(m.selectedItems) - maxDisplay
				content.WriteString("  " + listStyle.Render(fmt.Sprintf("... and %d more", remaining)) + "\n")
				break
			}
			if task, ok := taskMap[itemID]; ok {
				content.WriteString("  " + listStyle.Render(fmt.Sprintf("• #%d - %s", task.ID, task.Title)) + "\n")
			} else {
				content.WriteString("  " + listStyle.Render(fmt.Sprintf("• #%d", itemID)) + "\n")
			}
			count++
		}

		content.WriteString(questionStyle.Render("\nConfirm batch deletion?") + "\n\n")
	} else {
		// Single delete
		content.WriteString("Are you sure you want to delete this work item?\n\n")
		content.WriteString("  " + itemStyle.Render(fmt.Sprintf("#%d - %s", m.deleteItemID, m.deleteItemTitle)) + "\n")
		content.WriteString(questionStyle.Render("\nConfirm deletion?") + "\n\n")
	}

	content.WriteString("  " + keyStyle.Render("[y]") + " Yes, delete it\n")
	content.WriteString("  " + keyStyle.Render("[n]") + " No, cancel\n\n")

	// Footer with keybindings
	keybindings := "y: confirm delete • n/esc: cancel"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}
