package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the appropriate view based on the current state
func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n", m.err)
	}

	switch m.state {
	case loadingView:
		return m.renderLoadingScreen()
	case detailView:
		return m.renderDetailView()
	case statePickerView:
		return m.renderStatePickerView()
	case filterView:
		return m.renderFilterView()
	case findView:
		return m.renderFindView()
	case helpView:
		return m.renderHelpView()
	case editView:
		return m.renderEditView()
	case createView:
		return m.renderCreateView()
	case errorView:
		return m.renderErrorView()
	case deleteConfirmView:
		return m.renderDeleteConfirmView()
	default:
		return m.renderListView()
	}
}

// renderLoadingScreen renders the initial loading screen with centered text
func (m model) renderLoadingScreen() string {
	hippoArt := `⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣀⢤⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⠖⠓⠲⢤⡀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⣀⡾⡭⢤⡄⠈⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⡇⢠⡾⠛⢲⣿⡀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⢹⣿⠁⢸⣿⠀⢻⡄⢀⣠⠤⠴⠶⠶⢦⡤⠤⠒⠒⠒⠒⠦⣤⡀⠀⠀⣸⠀⢸⡇⠀⢸⢿⡇⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠸⣿⣆⠀⢿⣄⣤⠟⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⡤⠴⠦⢭⣷⣶⠃⢀⡞⠀⣠⠋⡼⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠙⢿⣷⢴⡿⠋⠉⠉⠓⠄⠀⠀⠀⠀⠀⠀⠀⡔⠁⠀⠀⠀⠀⠈⠻⣷⣿⠴⢊⡡⠞⠁⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⣿⠁⣀⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠀⠀⠤⣤⣤⡀⠀⠀⠀⢻⡚⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⡿⠚⠛⠻⣧⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢰⠟⠉⠉⠙⢦⠀⠀⠈⢧⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⡼⢁⣴⣶⣤⢸⡄⠀⠀⠀⠀⠀⠀⠀⠀⠀⡟⢀⣾⣿⠷⣬⡇⠀⠀⠈⢳⡄⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⢀⣼⡇⣼⣿⣧⣽⣾⠃⠀⠀⠀⠀⠀⠀⠀⠀⠀⣿⠈⣿⣷⣴⣿⡟⠀⠀⠀⠀⠹⡆⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⡾⠁⣳⣙⣿⣿⣿⣃⡀⠀⠀⠀⠀⠀⣀⣀⡀⠀⠀⠳⣝⣿⡿⠟⠳⠀⠀⠀⠀⠀⢻⡀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⢸⡇⠀⣠⠿⠏⠉⠉⠉⠉⠉⠙⡿⠛⠉⠉⠀⠉⠁⠀⠀⠜⠁⠀⠀⠀⠀⠀⠀⠀⠀⠸⡇⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⣧⡾⠁⢠⣿⠙⣆⠀⠀⠀⡼⠁⠀⠀⠀⠀⡴⢋⣿⣷⠀⠀⠀⢀⣶⡄⠀⠀⠀⠀⠀⣷⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⣼⠁⠀⠈⠻⣿⡿⠀⠀⢠⡇⠀⠀⠀⠀⠐⢳⠿⠛⠉⠀⠀⠀⢸⣿⠃⠀⠀⠀⠀⢠⡟⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⣿⠀⠀⠀⠀⠈⠁⠀⠀⢸⡇⠀⠀⠀⠀⠀⢀⠀⠀⠀⠀⠀⠀⠈⡇⠀⠀⠀⠀⢠⡾⠁⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠸⣆⠀⠀⠀⠀⠀⠋⠀⠘⣇⠀⠀⠀⠂⠀⠈⠀⠀⠀⠀⠀⠀⢸⠃⠀⠀⣠⡴⠋⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠙⣦⡀⠈⠁⠀⢀⣀⣶⣦⣤⣀⠀⠀⠉⠀⠐⠂⠀⠀⢀⣰⠏⢀⣤⣾⡉⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⡿⣷⡒⠋⠉⠁⠀⣀⣀⠈⠙⠓⠶⠤⠤⠤⠴⠖⣋⣥⠞⠋⢸⠏⣇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣸⣇⠈⠙⠓⠒⠚⠋⠉⠉⠙⠲⢤⣄⣀⣀⣤⠴⠚⠯⠀⠀⣠⠏⢀⡿⣄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣯⠙⢦⣀⠀⠀⠀⠀⠀⠀⠀⠀⠉⠀⠀⠀⠀⠀⠀⠀⠀⠞⠁⢀⣾⠁⠈⠳⡄⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⣸⠛⢧⡀⠉⠓⠦⢤⣀⣀⣀⣀⠀⠀⠀⠀⣀⣀⣀⠄⠀⠀⠀⢠⡞⠁⢠⠀⠀⠹⡄⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⡇⠀⠈⠻⢦⡀⠀⠀⠀⠀⠀⠉⠉⠉⠉⠉⠁⠀⠀⠀⢀⡠⠖⠉⠀⠀⢸⡇⠀⠀⢹⡀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⡇⠀⠀⠀⠀⠙⣶⣤⣀⠀⠀⠀⠀⠀⣀⣠⠀⠀⡀⠘⠉⠀⠀⠀⠀⠀⣸⡇⠀⠀⣀⣷⠀⠀⠀⠀⢀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⢳⡀⠀⠀⠀⠀⠸⣏⠈⠉⠙⠉⠉⠉⠁⠀⠀⣸⠇⠀⠀⠀⠀⠀⠀⢠⣿⠇⠀⠀⠀⠙⢧⠀⠀⢀⠞⠂
⠀⠀⠀⠀⠀⠀⠀⠀⣀⠼⣷⠀⠀⠀⠀⠀⢻⣧⠀⠀⠀⠀⠀⠀⠀⣰⠿⠃⠀⠀⠀⠀⢀⣴⣿⡟⠀⠀⠀⠀⠀⠀⢷⠊⠁⡼⠁
⠀⣠⠴⠒⣦⣴⣶⡋⠁⠀⢨⠀⠀⠀⠀⠀⠀⠹⣷⡄⠀⠀⠀⠀⣰⠏⠀⠀⠀⠀⠀⠀⢋⣿⡟⠀⢀⣠⣤⣤⣤⣴⣿⣶⠋⠀⠀
⡾⢿⠒⠋⠀⢯⣙⣷⡀⠀⠘⣆⠀⠀⠀⠀⠀⠐⢹⡟⠒⠲⠖⠊⣿⡀⠀⠀⠀⠀⠀⠀⣼⠟⠀⢰⡿⢋⡽⠋⠈⠧⣍⣻⡇⠀⠀
⣇⣸⠀⠀⠀⠀⠈⠉⠳⡀⠀⢸⡄⠀⢀⣀⣀⡀⠀⣧⠦⠀⠀⢀⣿⠁⢠⣄⣤⠀⠀⢠⡟⠀⣠⠛⠋⠉⠀⠀⠀⠀⢀⢼⣇⠀⠀
⠙⢧⡀⠀⠀⠀⠀⠀⠀⢹⣆⣸⠇⠐⠛⠉⠉⠀⠀⠹⣿⠀⢠⣾⠃⠈⠁⠀⠉⠀⠀⠸⡇⢠⠇⠀⠀⠀⠀⠀⠀⠀⣇⣾⠏⠀⠀
⠀⠈⠙⢦⡀⠀⠀⠀⢀⣼⣿⠷⠶⣄⠀⣀⣀⡀⢀⡖⠻⣦⣼⣧⠤⢄⠀⣠⣤⣄⣠⠞⣷⣾⠀⠀⠀⠀⠀⠀⢀⡴⠛⠉⠀⠀⠀
⠀⠀⠀⠀⠉⠓⠒⠒⠛⠁⠘⠦⣤⣼⣶⣁⣀⣹⡾⠤⠚⠁⠘⠧⣤⡼⠶⣇⣠⠼⠟⠛⠉⠀⠳⠤⣤⣤⠿⠟⠋⠀⠀⠀⠀⠀⠀`

	// Calculate vertical centering
	artLines := strings.Split(hippoArt, "\n")
	totalHeight := 3 + len(artLines) // spinner line + blank + art
	verticalPadding := (m.height - totalHeight) / 2
	if verticalPadding < 0 {
		verticalPadding = 0
	}

	// Style for text
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("212")).
		Bold(true).
		Width(m.width).
		Align(lipgloss.Center)

	// Style the ASCII art
	artStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("212")).
		Width(m.width).
		Align(lipgloss.Center)

	// Build the centered content
	var content strings.Builder

	// Add vertical padding
	for i := 0; i < verticalPadding; i++ {
		content.WriteString("\n")
	}

	// Add the "Hippo" text with spinners on the sides
	spinnerView := m.spinner.View()
	titleLine := fmt.Sprintf("%s  Hippo  %s", spinnerView, spinnerView)
	content.WriteString(titleStyle.Render(titleLine) + "\n")

	content.WriteString("\n")

	// Add the ASCII art hippo
	for _, line := range artLines {
		content.WriteString(artStyle.Render(line) + "\n")
	}

	return content.String()
}

// renderLogLine renders the action log line if there is one
func (m model) renderLogLine() string {
	if m.lastActionLog == "" {
		return ""
	}

	timestamp := m.lastActionTime.Format("15:04:05")
	return m.styles.Log.Render(fmt.Sprintf("[%s] %s", timestamp, m.lastActionLog))
}

// renderTitleBar renders the title bar with the given title text
func (m model) renderTitleBar(title string) string {
	titleBarStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(ColorPurple)).
		Foreground(lipgloss.Color(ColorWhite)).
		Bold(true).
		Width(m.width).
		Padding(0, 1)

	// Calculate padding to align version to the right
	versionText := version
	availableWidth := m.width - len(title) - len(versionText) - 4 // 4 for padding (2 on each side)
	if availableWidth < 0 {
		availableWidth = 0
	}
	padding := strings.Repeat(" ", availableWidth)

	titleWithVersion := title + padding + versionText
	return titleBarStyle.Render(titleWithVersion) + "\n\n"
}

// renderFooter renders the bottom section with action log and keybindings
func (m model) renderFooter(keybindings string) string {
	var footer strings.Builder

	// Action log line
	footer.WriteString("\n")
	if m.lastActionLog != "" {
		footer.WriteString(m.renderLogLine() + "\n")
	}

	// Separator line
	separatorStyle := m.styles.Separator.Width(m.width)
	separator := separatorStyle.Render(strings.Repeat("─", m.width))

	footer.WriteString(separator + "\n")
	footer.WriteString(m.styles.Help.Render(keybindings))

	return footer.String()
}

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

func (m model) renderDetailView() string {
	if m.selectedTask == nil {
		return "No task selected"
	}

	var content strings.Builder

	// Title bar
	titleText := fmt.Sprintf("Work Item Details")
	content.WriteString(m.renderTitleBar(titleText))

	// Render the card
	content.WriteString(m.buildDetailContent())
	content.WriteString("\n")

	// Footer with keybindings
	keybindings := "←/h/esc: back • r: refresh • e: edit • o: open in browser • s: change state • ?: help • q: quit"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}

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

func (m model) renderHelpView() string {
	var content strings.Builder

	// Title bar
	titleText := "Keybindings Help"
	content.WriteString(m.renderTitleBar(titleText))

	// Styles
	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		MarginTop(1).
		MarginBottom(1)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true).
		Width(20)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230"))

	// Global keybindings
	content.WriteString(sectionStyle.Render("Global") + "\n")
	content.WriteString(keyStyle.Render("?") + descStyle.Render("Show/hide this help") + "\n")
	content.WriteString(keyStyle.Render("q, ctrl+c") + descStyle.Render("Quit application") + "\n")
	content.WriteString(keyStyle.Render("ctrl+u/d, pgup/pgdn") + descStyle.Render("Jump half page up/down (works in all views)") + "\n")
	content.WriteString(keyStyle.Render("r") + descStyle.Render("Refresh (all data in list, single item in detail)") + "\n")
	content.WriteString(keyStyle.Render("o") + descStyle.Render("Open current item in browser") + "\n\n")

	// List view keybindings
	content.WriteString(sectionStyle.Render("List View") + "\n")
	content.WriteString(keyStyle.Render("tab") + descStyle.Render("Cycle through tabs (sprint or backlog)") + "\n")
	content.WriteString(keyStyle.Render("↑/↓, j/k") + descStyle.Render("Navigate up/down") + "\n")
	content.WriteString(keyStyle.Render("space") + descStyle.Render("Select/deselect item for batch operations") + "\n")
	content.WriteString(keyStyle.Render("→/l, enter") + descStyle.Render("Open item details") + "\n")
	content.WriteString(keyStyle.Render("i") + descStyle.Render("Insert new item before current") + "\n")
	content.WriteString(keyStyle.Render("a") + descStyle.Render("Append new item after current (or as first child if parent)") + "\n")
	content.WriteString(keyStyle.Render("d") + descStyle.Render("Delete current item or selected items (with confirmation)") + "\n")
	content.WriteString(keyStyle.Render("s") + descStyle.Render("Change state of selected items (batch operation)") + "\n")
	content.WriteString(keyStyle.Render("/") + descStyle.Render("Filter items in current list") + "\n")
	content.WriteString(keyStyle.Render("f") + descStyle.Render("Find items with dedicated query") + "\n\n")

	// Detail view keybindings
	content.WriteString(sectionStyle.Render("Detail View") + "\n")
	content.WriteString(keyStyle.Render("←/h, esc, backspace") + descStyle.Render("Back to list") + "\n")
	content.WriteString(keyStyle.Render("e") + descStyle.Render("Edit item") + "\n")
	content.WriteString(keyStyle.Render("s") + descStyle.Render("Change item state") + "\n\n")

	// State picker view keybindings
	content.WriteString(sectionStyle.Render("State Picker") + "\n")
	content.WriteString(keyStyle.Render("↑/↓, j/k") + descStyle.Render("Navigate states") + "\n")
	content.WriteString(keyStyle.Render("enter") + descStyle.Render("Select state") + "\n")
	content.WriteString(keyStyle.Render("esc") + descStyle.Render("Cancel") + "\n\n")

	// Filter view keybindings
	content.WriteString(sectionStyle.Render("Filter View") + "\n")
	content.WriteString(keyStyle.Render("esc") + descStyle.Render("Cancel filter") + "\n")
	content.WriteString(keyStyle.Render("enter") + descStyle.Render("Open selected item") + "\n")
	content.WriteString(keyStyle.Render("↑/↓, ctrl+j/k") + descStyle.Render("Navigate results") + "\n\n")

	// Create view keybindings
	content.WriteString(sectionStyle.Render("Create View") + "\n")
	content.WriteString(keyStyle.Render("ctrl+s") + descStyle.Render("Save new item") + "\n")
	content.WriteString(keyStyle.Render("enter") + descStyle.Render("Show save/cancel hint") + "\n")
	content.WriteString(keyStyle.Render("esc") + descStyle.Render("Cancel creation") + "\n\n")

	// Batch operations
	content.WriteString(sectionStyle.Render("Batch Operations") + "\n")
	content.WriteString(keyStyle.Render("space") + descStyle.Render("Select/deselect item (orange bar shows selection)") + "\n")
	content.WriteString(keyStyle.Render("d") + descStyle.Render("Delete all selected items (with confirmation)") + "\n")
	content.WriteString(keyStyle.Render("s") + descStyle.Render("Change state of all selected items") + "\n\n")

	// Footer with keybindings
	keybindings := "?: close help • esc: close help • q: quit"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}

func (m model) renderEditView() string {
	var content strings.Builder

	// Title bar
	titleText := "Edit Work Item"
	if m.selectedTask != nil {
		titleText = fmt.Sprintf("Edit Work Item #%d", m.selectedTask.ID)
	}
	content.WriteString(m.renderTitleBar(titleText))

	// Styles
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true).
		Width(15)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	sectionStyle := lipgloss.NewStyle().
		MarginTop(1).
		MarginBottom(1)

	// Show loading spinner if saving
	if m.loading {
		loaderStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			MarginLeft(2)
		content.WriteString(loaderStyle.Render(fmt.Sprintf("%s %s", m.spinner.View(), m.statusMessage)) + "\n\n")

		// Footer with keybindings
		keybindings := "Saving changes..."
		content.WriteString(m.renderFooter(keybindings))

		return content.String()
	}

	// Title field
	content.WriteString(sectionStyle.Render(labelStyle.Render("Title:") + "\n" + m.editTitleInput.View()))
	content.WriteString("\n")
	if m.editFieldCursor == 0 {
		content.WriteString(helpStyle.Render("  Enter the work item title") + "\n")
	}
	content.WriteString("\n")

	// Description field
	content.WriteString(sectionStyle.Render(labelStyle.Render("Description:") + "\n" + m.editDescriptionInput.View()))
	content.WriteString("\n")
	if m.editFieldCursor == 1 {
		content.WriteString(helpStyle.Render("  Multi-line text editor (HTML will be stripped)") + "\n")
	}
	content.WriteString("\n")

	// Footer with keybindings
	keybindings := "tab/shift+tab: switch field • ctrl+s: save • esc: cancel • ?: help"
	content.WriteString(m.renderFooter(keybindings))

	return content.String()
}

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

// openInBrowser opens the work item in a browser
func openInBrowser(orgURL, project string, workItemID int) error {
	// Clean up org URL
	orgURL = strings.TrimSuffix(orgURL, "/")

	url := fmt.Sprintf("%s/%s/_workitems/edit/%d", orgURL, project, workItemID)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default: // linux and others
		cmd = exec.Command("xdg-open", url)
	}

	return cmd.Start()
}
