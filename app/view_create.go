package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renderCreateView() string {
	var content strings.Builder

	// Title bar
	title := "Create New Work Item"
	content.WriteString(m.renderTitleBar(title))

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Bold(true)

	// State styles based on category
	proposedStateStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")) // Normal gray

	inProgressStateStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")). // Green
		Bold(true)

	completedStateStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")). // Dimmed gray
		Italic(true)

	removedStateStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")). // Very dim
		Italic(true)

	// Tree edge and icon styles
	edgeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	iconStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

	// Mode and tab styles (reuse from list view)
	activeModeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("62")).
		Padding(0, 2).
		MarginRight(1)

	inactiveModeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 2).
		MarginRight(1)

	activeTabStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("62")).
		Padding(0, 2)

	inactiveTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 2)

	// Render mode selector
	modes := []string{}
	if m.currentMode == sprintMode {
		modes = append(modes, activeModeStyle.Render("[1] Sprint"))
		modes = append(modes, inactiveModeStyle.Render("[2] Backlog"))
	} else {
		modes = append(modes, inactiveModeStyle.Render("[1] Sprint"))
		modes = append(modes, activeModeStyle.Render("[2] Backlog"))
	}
	content.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, modes...) + "\n\n")

	// Render tabs based on current mode
	tabs := []string{}

	if m.currentMode == sprintMode {
		prevLabel := "Previous Sprint"
		if sprint := m.sprints[previousSprint]; sprint != nil {
			prevLabel = sprint.Name
		}
		if m.currentTab == previousSprint {
			tabs = append(tabs, activeTabStyle.Render(prevLabel))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(prevLabel))
		}

		currLabel := "Current Sprint"
		if sprint := m.sprints[currentSprint]; sprint != nil {
			currLabel = sprint.Name
		}
		if m.currentTab == currentSprint {
			tabs = append(tabs, activeTabStyle.Render(currLabel))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(currLabel))
		}

		nextLabel := "Next Sprint"
		if sprint := m.sprints[nextSprint]; sprint != nil {
			nextLabel = sprint.Name
		}
		if m.currentTab == nextSprint {
			tabs = append(tabs, activeTabStyle.Render(nextLabel))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(nextLabel))
		}
	} else if m.currentMode == backlogMode {
		if m.currentBacklogTab == recentBacklog {
			tabs = append(tabs, activeTabStyle.Render("Recent Backlog"))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render("Recent Backlog"))
		}

		if m.currentBacklogTab == abandonedWork {
			tabs = append(tabs, activeTabStyle.Render("Abandoned Work"))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render("Abandoned Work"))
		}
	}

	content.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tabs...) + "\n\n")

	// Show tab hint
	if hint := m.getTabHint(); hint != "" {
		hintStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)
		content.WriteString(hintStyle.Render(hint) + "\n\n")
	}

	if m.statusMessage != "" {
		msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
		content.WriteString(msgStyle.Render(m.statusMessage) + "\n\n")
	}

	// Show loader if creating
	if m.loading {
		loaderStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			MarginLeft(2)
		content.WriteString(loaderStyle.Render(fmt.Sprintf("%s %s", m.spinner.View(), m.statusMessage)) + "\n\n")

		// Footer with keybindings
		keybindings := "Creating work item..."
		content.WriteString(m.renderFooter(keybindings))

		return content.String()
	}

	treeItems := m.getVisibleTreeItems()

	// Calculate visible range based on scroll offset
	contentHeight := m.getContentHeight()
	startIdx := m.scrollOffset
	endIdx := m.scrollOffset + contentHeight

	// Total items including the create input line
	totalItems := len(treeItems)
	if m.createInsertPos <= len(treeItems) {
		totalItems++
	}
	if m.hasMoreItems() {
		totalItems++
	}

	// Clamp end index
	if endIdx > totalItems {
		endIdx = totalItems
	}

	// Render visible items with the create input inserted at the correct position
	visibleIdx := 0
	for i := startIdx; i < endIdx; i++ {
		// Check if we should render the create input at this position
		if i == m.createInsertPos {
			// Render the create input line with tree prefix
			var prefix strings.Builder

			// Draw tree prefix based on depth
			for d := 0; d < m.createDepth-1; d++ {
				if d < len(m.createIsLast) && m.createIsLast[d] {
					prefix.WriteString("    ")
				} else {
					prefix.WriteString("│   ")
				}
			}

			// Draw connector for this item
			if m.createDepth > 0 {
				if len(m.createIsLast) > 0 && m.createIsLast[len(m.createIsLast)-1] {
					prefix.WriteString("╰── ")
				} else {
					prefix.WriteString("├── ")
				}
			}

			prefixStr := edgeStyle.Render(prefix.String())

			// Build the create line
			var createLine strings.Builder
			createLine.WriteString(prefixStr)
			createLine.WriteString(iconStyle.Render("✓ "))
			createLine.WriteString(selectedStyle.Render("[New] "))
			createLine.WriteString(m.createInput.View())

			content.WriteString(createLine.String() + "\n")
			visibleIdx++
			continue
		}

		// Calculate actual tree item index (accounting for inserted create line)
		treeIdx := i
		if i > m.createInsertPos {
			treeIdx = i - 1
		}

		// Check if this is a regular tree item
		if treeIdx < len(treeItems) {
			item := treeItems[treeIdx]
			task := item.WorkItem

			// Build tree prefix
			treePrefix := getTreePrefix(item)
			icon := getWorkItemIcon(task.WorkItemType)

			// Get state style
			category := m.getStateCategory(task.State)
			var stateStyle lipgloss.Style
			switch category {
			case "Proposed":
				stateStyle = proposedStateStyle
			case "InProgress":
				stateStyle = inProgressStateStyle
			case "Completed":
				stateStyle = completedStateStyle
			case "Removed":
				stateStyle = removedStateStyle
			default:
				stateStyle = proposedStateStyle
			}

			// Build the line
			var line strings.Builder
			line.WriteString(edgeStyle.Render(treePrefix))
			line.WriteString(iconStyle.Render(icon + " "))
			line.WriteString(stateStyle.Render(fmt.Sprintf("[%s] ", task.State)))
			line.WriteString(stateStyle.Render(fmt.Sprintf("#%d - %s", task.ID, task.Title)))

			content.WriteString(line.String() + "\n")
			visibleIdx++
		} else if treeIdx == len(treeItems) && m.hasMoreItems() {
			// "Load More" item
			remaining := m.getRemainingCount()
			loadMoreText := fmt.Sprintf("▼ Load %d more items...", remaining)
			loadMoreStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("86")).
				Italic(true)
			content.WriteString("  " + loadMoreStyle.Render(loadMoreText) + "\n")
			visibleIdx++
		}
	}

	// Footer with keybindings
	keybindings := "ctrl+s: save • enter: show help • esc: cancel • ?: help"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}
