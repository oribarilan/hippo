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
	if m.filter.active {
		title += fmt.Sprintf(" (filtered: %d results)", len(m.filter.filteredTasks))
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
	startIdx := m.ui.scrollOffset
	endIdx := m.ui.scrollOffset + contentHeight

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
				loadMoreIdx := len(treeItems)
				isSelected := m.ui.cursor == loadMoreIdx
				loadMoreText := m.renderLoadMoreItem(isSelected, remaining, m.loadingMore)
				content.WriteString(loadMoreText + "\n")
			}
			continue
		}

		treeItem := treeItems[i]
		isSelected := m.ui.cursor == i
		isBatchSelected := m.batch.selectedItems[treeItem.WorkItem.ID]

		line := m.renderTreeItemList(treeItem, isSelected, isBatchSelected)
		content.WriteString(line + "\n")
	}

	// Footer with keybindings
	batchInfo := ""
	if len(m.batch.selectedItems) > 0 {
		batchInfo = fmt.Sprintf(" • %d items selected", len(m.batch.selectedItems))
	}
	keybindings := fmt.Sprintf("tab: cycle tabs • space: select/deselect%s\ni: insert • d: delete • e: edit • enter: details • o: open • /: filter • f: find • r: refresh • ?: help • q: quit", batchInfo)
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}
