package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) renderListView() string {
	var content strings.Builder

	// Title bar
	title := "Azure DevOps - Work Items"
	if m.filterActive {
		title += fmt.Sprintf(" (filtered: %d results)", len(m.filteredTasks))
	}
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
		// Backlog mode tabs
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

	// Show loader if loading (but not loadingMore and not initial loading)
	// During initial loading (m.initialLoading > 0), we want to show the list view without a spinner
	if m.loading && !m.loadingMore && m.initialLoading == 0 {
		content.WriteString(m.styles.Loader.Render(fmt.Sprintf("%s %s", m.spinner.View(), m.statusMessage)) + "\n\n")

		// Footer with keybindings
		keybindings := "tab: cycle tabs • →/l: details • ↑/↓ or j/k: navigate\nenter: details • o: open in browser • /: search • f: filter • r: refresh • q: quit"
		content.WriteString(m.renderFooter(keybindings))

		return content.String()
	}

	treeItems := m.getVisibleTreeItems()
	if len(treeItems) == 0 {
		content.WriteString("  No tasks found.\n")
	}

	// Calculate visible range based on scroll offset
	contentHeight := m.getContentHeight()
	startIdx := m.scrollOffset
	endIdx := m.scrollOffset + contentHeight

	// Total items including potential "Load More" item
	totalItems := len(treeItems)
	if m.hasMoreItems() {
		totalItems++
	}

	// Clamp end index
	if endIdx > totalItems {
		endIdx = totalItems
	}

	// Render only visible items
	for i := startIdx; i < endIdx; i++ {
		// Check if this is the "Load More" item
		if i >= len(treeItems) {
			// This is the "Load More" item
			if m.hasMoreItems() {
				remaining := m.getRemainingCount()

				cursor := " "
				loadMoreIdx := len(treeItems)
				if m.cursor == loadMoreIdx {
					cursor = ">"
				}

				var loadMoreText string

				// Show spinner inline if loading more
				if m.loadingMore {
					loadMoreText = fmt.Sprintf("%s %s Loading more items...", cursor, m.spinner.View())
					if m.cursor == loadMoreIdx {
						loadMoreText = m.styles.Selected.Render(loadMoreText)
					} else {
						loadMoreText = m.styles.LoadMore.Render(loadMoreText)
					}
				} else {
					if remaining > 30 {
						loadMoreText = fmt.Sprintf("%s Load More (+30)", cursor)
					} else {
						loadMoreText = fmt.Sprintf("%s Load All (+%d)", cursor, remaining)
					}

					if m.cursor == loadMoreIdx {
						loadMoreText = m.styles.Selected.Render(loadMoreText)
					} else {
						loadMoreText = m.styles.LoadMore.Render(loadMoreText)
					}
				}

				content.WriteString(loadMoreText + "\n")
			}
			continue
		}

		treeItem := treeItems[i]
		isSelected := m.cursor == i
		isBatchSelected := m.selectedItems[treeItem.WorkItem.ID]

		cursor := " "
		if isSelected {
			cursor = "❯"
		}

		// Orange bar for batch selected items - thicker bar
		batchIndicator := "  "
		if isBatchSelected {
			batchIndicator = m.styles.BatchIndicator.Render("█ ") // Block character for thicker bar
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
			line = fmt.Sprintf("%s%s%s%s%s%s%s%s",
				batchIndicator,
				cursorStyled,
				spacer,
				treePrefix,
				styledIcon,
				spacer,
				taskTitle,
				spacer+state)
		} else {
			line = fmt.Sprintf("%s%s %s%s %s %s",
				batchIndicator,
				cursor,
				treePrefix,
				styledIcon,
				taskTitle,
				state)
		}

		content.WriteString(line + "\n")
	}

	// Footer with keybindings
	batchInfo := ""
	if len(m.selectedItems) > 0 {
		batchInfo = fmt.Sprintf(" • %d items selected", len(m.selectedItems))
	}
	keybindings := fmt.Sprintf("tab: cycle tabs • →/l: details • ↑/↓ or j/k: navigate • space: select/deselect%s\ni: insert • a: append • d: delete • s: change state • enter: details • o: open • /: filter • f: find • r: refresh • ?: help • q: quit", batchInfo)
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}
