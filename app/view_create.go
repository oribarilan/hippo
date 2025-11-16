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

	// Render mode selector
	modes := []string{}
	if m.currentMode == sprintMode {
		modes = append(modes, m.styles.ActiveMode.Render("[1] Sprint"))
		modes = append(modes, m.styles.InactiveMode.Render("[2] Backlog"))
	} else {
		modes = append(modes, m.styles.InactiveMode.Render("[1] Sprint"))
		modes = append(modes, m.styles.ActiveMode.Render("[2] Backlog"))
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
			tabs = append(tabs, m.styles.ActiveTab.Render(prevLabel))
		} else {
			tabs = append(tabs, m.styles.InactiveTab.Render(prevLabel))
		}

		currLabel := "Current Sprint"
		if sprint := m.sprints[currentSprint]; sprint != nil {
			currLabel = sprint.Name
		}
		if m.currentTab == currentSprint {
			tabs = append(tabs, m.styles.ActiveTab.Render(currLabel))
		} else {
			tabs = append(tabs, m.styles.InactiveTab.Render(currLabel))
		}

		nextLabel := "Next Sprint"
		if sprint := m.sprints[nextSprint]; sprint != nil {
			nextLabel = sprint.Name
		}
		if m.currentTab == nextSprint {
			tabs = append(tabs, m.styles.ActiveTab.Render(nextLabel))
		} else {
			tabs = append(tabs, m.styles.InactiveTab.Render(nextLabel))
		}
	} else if m.currentMode == backlogMode {
		if m.currentBacklogTab == recentBacklog {
			tabs = append(tabs, m.styles.ActiveTab.Render("Recent Backlog"))
		} else {
			tabs = append(tabs, m.styles.InactiveTab.Render("Recent Backlog"))
		}

		if m.currentBacklogTab == abandonedWork {
			tabs = append(tabs, m.styles.ActiveTab.Render("Abandoned Work"))
		} else {
			tabs = append(tabs, m.styles.InactiveTab.Render("Abandoned Work"))
		}
	}

	content.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tabs...) + "\n\n")

	// Show tab hint
	if hint := m.getTabHint(); hint != "" {
		content.WriteString(m.styles.Hint.Render(hint) + "\n\n")
	}

	if m.statusMessage != "" {
		content.WriteString(m.styles.StatusMsg.Render(m.statusMessage) + "\n\n")
	}

	// Show loader if creating
	if m.loading {
		content.WriteString(m.styles.Loader.Render(fmt.Sprintf("%s %s", m.spinner.View(), m.statusMessage)) + "\n\n")

		// Footer with keybindings
		keybindings := "Creating work item..."
		content.WriteString(m.renderFooter(keybindings))

		return content.String()
	}

	treeItems := m.getVisibleTreeItems()

	// Calculate visible range based on scroll offset
	contentHeight := m.getContentHeight()
	startIdx := m.ui.scrollOffset
	endIdx := m.ui.scrollOffset + contentHeight

	// Total items including the create input line
	totalItems := len(treeItems)
	if m.create.insertPos <= len(treeItems) {
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
		if i == m.create.insertPos {
			// Render the create input line with tree prefix
			var prefix strings.Builder

			// Draw tree prefix based on depth
			for d := 0; d < m.create.depth-1; d++ {
				if d < len(m.create.isLast) && m.create.isLast[d] {
					prefix.WriteString("    ")
				} else {
					prefix.WriteString("│   ")
				}
			}

			// Draw connector for this item
			if m.create.depth > 0 {
				if len(m.create.isLast) > 0 && m.create.isLast[len(m.create.isLast)-1] {
					prefix.WriteString("╰── ")
				} else {
					prefix.WriteString("├── ")
				}
			}

			prefixStr := m.styles.TreeEdge.Render(prefix.String())

			// Build the create line
			var createLine strings.Builder
			createLine.WriteString(prefixStr)
			createLine.WriteString(m.styles.Icon.Render("✓ "))
			createLine.WriteString(m.styles.Selected.Render("[New] "))
			createLine.WriteString(m.create.input.View())

			content.WriteString(createLine.String() + "\n")
			visibleIdx++
			continue
		}

		// Calculate actual tree item index (accounting for inserted create line)
		treeIdx := i
		if i > m.create.insertPos {
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
			stateStyle := m.styles.GetStateStyle(category, false)

			// Build the line
			var line strings.Builder
			line.WriteString(m.styles.TreeEdge.Render(treePrefix))
			line.WriteString(m.styles.Icon.Render(icon + " "))
			line.WriteString(stateStyle.Render(fmt.Sprintf("[%s] ", task.State)))
			line.WriteString(stateStyle.Render(fmt.Sprintf("#%d - %s", task.ID, task.Title)))

			content.WriteString(line.String() + "\n")
			visibleIdx++
		} else if treeIdx == len(treeItems) && m.hasMoreItems() {
			// "Load More" item
			remaining := m.getRemainingCount()
			loadMoreText := fmt.Sprintf("▼ Load %d more items...", remaining)
			content.WriteString("  " + m.styles.LoadMore.Render(loadMoreText) + "\n")
			visibleIdx++
		}
	}

	// Footer with keybindings
	keybindings := "ctrl+s: save • enter: show help • esc: cancel • ?: help"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}
