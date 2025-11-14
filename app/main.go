package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
)

type viewState int

const (
	listView viewState = iota
	detailView
	statePickerView
	searchView
	filterView
)

type model struct {
	tasks           []WorkItem
	filteredTasks   []WorkItem
	cursor          int
	state           viewState
	selectedTask    *WorkItem
	loading         bool
	err             error
	client          *AzureDevOpsClient
	spinner         spinner.Model
	searchInput     textinput.Model
	filterInput     textinput.Model
	stateCursor     int
	availableStates []string
	organizationURL string
	projectName     string
	searchActive    bool
	statusMessage   string
}

type WorkItem struct {
	ID           int
	Title        string
	State        string
	AssignedTo   string
	WorkItemType string
	Description  string
	Tags         string
	Priority     int
	CreatedDate  string
	ChangedDate  string
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	searchInput := textinput.New()
	searchInput.Placeholder = "Search by title or ID..."
	searchInput.Focus()

	filterInput := textinput.New()
	filterInput.Placeholder = "Enter filter (e.g., assigned to me, all items)..."
	filterInput.Focus()

	return model{
		tasks:           []WorkItem{},
		filteredTasks:   []WorkItem{},
		cursor:          0,
		state:           listView,
		loading:         true,
		spinner:         s,
		searchInput:     searchInput,
		filterInput:     filterInput,
		availableStates: []string{"New", "Active", "Closed", "Removed"},
		organizationURL: os.Getenv("AZURE_DEVOPS_ORG_URL"),
		projectName:     os.Getenv("AZURE_DEVOPS_PROJECT"),
	}
}

type tasksLoadedMsg struct {
	tasks  []WorkItem
	err    error
	client *AzureDevOpsClient
}

type stateUpdatedMsg struct {
	err error
}

type statesLoadedMsg struct {
	states []string
	err    error
}

func loadTasks(client *AzureDevOpsClient) tea.Cmd {
	return func() tea.Msg {
		tasks, err := client.GetWorkItems()
		return tasksLoadedMsg{tasks: tasks, err: err, client: client}
	}
}

func loadWorkItemStates(client *AzureDevOpsClient, workItemType string) tea.Cmd {
	return func() tea.Msg {
		states, err := client.GetWorkItemTypeStates(workItemType)
		return statesLoadedMsg{states: states, err: err}
	}
}

func updateWorkItemState(client *AzureDevOpsClient, workItemID int, newState string) tea.Cmd {
	return func() tea.Msg {
		err := client.UpdateWorkItemState(workItemID, newState)
		return stateUpdatedMsg{err: err}
	}
}

func (m model) Init() tea.Cmd {
	client, err := NewAzureDevOpsClient()
	if err != nil {
		return func() tea.Msg {
			return tasksLoadedMsg{err: err}
		}
	}
	return tea.Batch(loadTasks(client), m.spinner.Tick)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle search input
		if m.state == searchView {
			switch msg.String() {
			case "esc":
				m.state = listView
				m.searchActive = false
				m.searchInput.SetValue("")
				m.filteredTasks = nil
				m.cursor = 0
				return m, nil
			case "enter":
				m.state = listView
				m.searchActive = true
				m.cursor = 0
				return m, nil
			default:
				m.searchInput, cmd = m.searchInput.Update(msg)
				m.filterSearch()
				return m, cmd
			}
		}

		// Handle filter input
		if m.state == filterView {
			switch msg.String() {
			case "esc":
				m.state = listView
				m.filterInput.SetValue("")
				return m, nil
			case "enter":
				// For now, just go back to list view
				// Could implement custom queries here
				m.state = listView
				m.filterInput.SetValue("")
				if m.client != nil {
					m.loading = true
					return m, tea.Batch(loadTasks(m.client), m.spinner.Tick)
				}
				return m, nil
			default:
				m.filterInput, cmd = m.filterInput.Update(msg)
				return m, cmd
			}
		}

		// Handle state picker
		if m.state == statePickerView {
			switch msg.String() {
			case "esc":
				m.state = detailView
				m.stateCursor = 0
				return m, nil
			case "up", "k":
				if m.stateCursor > 0 {
					m.stateCursor--
				}
			case "down", "j":
				if m.stateCursor < len(m.availableStates)-1 {
					m.stateCursor++
				}
			case "enter":
				if m.selectedTask != nil && m.client != nil {
					newState := m.availableStates[m.stateCursor]
					m.loading = true
					m.statusMessage = fmt.Sprintf("Updating state to %s...", newState)
					return m, tea.Batch(
						updateWorkItemState(m.client, m.selectedTask.ID, newState),
						m.spinner.Tick,
					)
				}
				return m, nil
			}
			return m, nil
		}

		// Global hotkeys
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "r":
			// Refresh
			if m.client != nil {
				m.loading = true
				m.statusMessage = "Refreshing..."
				m.searchActive = false
				m.searchInput.SetValue("")
				m.filteredTasks = nil
				return m, tea.Batch(loadTasks(m.client), m.spinner.Tick)
			}
			return m, nil

		case "o":
			// Open in browser
			var workItemID int
			if m.state == detailView && m.selectedTask != nil {
				workItemID = m.selectedTask.ID
			} else if m.state == listView && len(m.getVisibleTasks()) > 0 {
				workItemID = m.getVisibleTasks()[m.cursor].ID
			}
			if workItemID > 0 {
				openInBrowser(m.organizationURL, m.projectName, workItemID)
			}
			return m, nil

		case "s":
			// Change state - load states from API
			if m.state == listView && len(m.getVisibleTasks()) > 0 && m.client != nil {
				m.selectedTask = &m.getVisibleTasks()[m.cursor]
				m.loading = true
				m.statusMessage = "Loading states..."
				return m, tea.Batch(
					loadWorkItemStates(m.client, m.selectedTask.WorkItemType),
					m.spinner.Tick,
				)
			} else if m.state == detailView && m.selectedTask != nil && m.client != nil {
				m.loading = true
				m.statusMessage = "Loading states..."
				return m, tea.Batch(
					loadWorkItemStates(m.client, m.selectedTask.WorkItemType),
					m.spinner.Tick,
				)
			}
			return m, nil

		case "/":
			// Search within results
			if m.state == listView {
				m.state = searchView
				m.searchInput.Focus()
			}
			return m, nil

		case "f":
			// New filter/query
			if m.state == listView {
				m.state = filterView
				m.filterInput.Focus()
			}
			return m, nil
		}

		// List view navigation
		if m.state == listView {
			switch msg.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				visibleTasks := m.getVisibleTasks()
				if m.cursor < len(visibleTasks)-1 {
					m.cursor++
				}
			case "enter":
				visibleTasks := m.getVisibleTasks()
				if len(visibleTasks) > 0 {
					m.selectedTask = &visibleTasks[m.cursor]
					m.state = detailView
				}
			}
		}

		// Detail view navigation
		if m.state == detailView {
			switch msg.String() {
			case "esc", "backspace":
				m.state = listView
			}
		}

	case tasksLoadedMsg:
		m.loading = false
		m.statusMessage = ""
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.tasks = msg.tasks
			m.filteredTasks = nil
			m.cursor = 0
			// Store the client if it was passed
			if msg.client != nil {
				m.client = msg.client
			}
		}

	case stateUpdatedMsg:
		m.loading = false
		m.statusMessage = ""
		m.state = listView
		m.stateCursor = 0
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error updating state: %v", msg.err)
		} else {
			m.statusMessage = "State updated successfully!"
			// Refresh the list
			if m.client != nil {
				m.loading = true
				return m, tea.Batch(loadTasks(m.client), m.spinner.Tick)
			}
		}

	case statesLoadedMsg:
		m.loading = false
		m.statusMessage = ""
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error loading states: %v", msg.err)
		} else {
			m.availableStates = msg.states
			m.state = statePickerView
			m.stateCursor = 0
		}

	case spinner.TickMsg:
		if m.loading {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *model) filterSearch() {
	query := strings.ToLower(m.searchInput.Value())
	if query == "" {
		m.filteredTasks = nil
		return
	}

	var filtered []WorkItem
	for _, task := range m.tasks {
		if strings.Contains(strings.ToLower(task.Title), query) ||
			strings.Contains(fmt.Sprintf("%d", task.ID), query) {
			filtered = append(filtered, task)
		}
	}
	m.filteredTasks = filtered
}

func (m model) getVisibleTasks() []WorkItem {
	if m.searchActive && m.filteredTasks != nil {
		return m.filteredTasks
	}
	return m.tasks
}

func (m model) View() string {
	if m.loading {
		return fmt.Sprintf("\n  %s %s\n\n", m.spinner.View(), m.statusMessage)
	}

	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n", m.err)
	}

	switch m.state {
	case detailView:
		return m.renderDetailView()
	case statePickerView:
		return m.renderStatePickerView()
	case searchView:
		return m.renderSearchView()
	case filterView:
		return m.renderFilterView()
	default:
		return m.renderListView()
	}
}

func (m model) renderListView() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Bold(true)

	stateStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86"))

	title := "Azure DevOps - Work Items"
	if m.searchActive {
		title += fmt.Sprintf(" (filtered: %d results)", len(m.filteredTasks))
	}
	s := titleStyle.Render(title) + "\n\n"

	if m.statusMessage != "" {
		msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
		s += msgStyle.Render(m.statusMessage) + "\n\n"
	}

	visibleTasks := m.getVisibleTasks()
	if len(visibleTasks) == 0 {
		s += "  No tasks found.\n"
	}

	for i, task := range visibleTasks {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		line := fmt.Sprintf("%s [%d] %s - %s", cursor, task.ID, task.Title, stateStyle.Render(task.State))

		if m.cursor == i {
			line = selectedStyle.Render(line)
		}

		s += line + "\n"
	}

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	s += helpStyle.Render("\n↑/↓ or j/k: navigate • enter: details • o: open in browser • s: change state\n/: search • f: filter • r: refresh • q: quit")

	return s
}

func (m model) renderDetailView() string {
	if m.selectedTask == nil {
		return "No task selected"
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212"))

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230"))

	task := m.selectedTask

	s := titleStyle.Render(fmt.Sprintf("Work Item #%d", task.ID)) + "\n\n"

	s += labelStyle.Render("Title: ") + valueStyle.Render(task.Title) + "\n"
	s += labelStyle.Render("Type: ") + valueStyle.Render(task.WorkItemType) + "\n"
	s += labelStyle.Render("State: ") + valueStyle.Render(task.State) + "\n"
	s += labelStyle.Render("Assigned To: ") + valueStyle.Render(task.AssignedTo) + "\n"

	if task.Priority > 0 {
		s += labelStyle.Render("Priority: ") + valueStyle.Render(fmt.Sprintf("%d", task.Priority)) + "\n"
	}

	if task.Tags != "" {
		s += labelStyle.Render("Tags: ") + valueStyle.Render(task.Tags) + "\n"
	}

	if task.CreatedDate != "" {
		s += labelStyle.Render("Created: ") + valueStyle.Render(task.CreatedDate) + "\n"
	}

	if task.ChangedDate != "" {
		s += labelStyle.Render("Last Updated: ") + valueStyle.Render(task.ChangedDate) + "\n"
	}

	if task.Description != "" {
		s += "\n" + labelStyle.Render("Description:") + "\n"
		s += valueStyle.Render(task.Description) + "\n"
	}

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	s += helpStyle.Render("\nesc/backspace: back • o: open in browser • s: change state • r: refresh • q: quit")

	return s
}

func (m model) renderStatePickerView() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Bold(true)

	s := titleStyle.Render("Select New State") + "\n\n"

	if m.selectedTask != nil {
		s += fmt.Sprintf("Current state: %s\n\n", m.selectedTask.State)
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

		s += line + "\n"
	}

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	s += helpStyle.Render("\n↑/↓ or j/k: navigate • enter: select • esc: cancel")

	return s
}

func (m model) renderSearchView() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	s := titleStyle.Render("Search Work Items") + "\n\n"
	s += m.searchInput.View() + "\n\n"

	if m.searchInput.Value() != "" {
		s += fmt.Sprintf("Found %d results\n", len(m.filteredTasks))
	}

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	s += helpStyle.Render("\nenter: apply search • esc: cancel")

	return s
}

func (m model) renderFilterView() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	s := titleStyle.Render("Filter Work Items") + "\n\n"
	s += m.filterInput.View() + "\n\n"
	s += "Examples:\n"
	s += "  - Press Enter to query all items\n"
	s += "  - (Custom filters coming soon)\n"

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)

	s += helpStyle.Render("\nenter: apply filter • esc: cancel")

	return s
}

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

func main() {
	// Load .env file if it exists (ignore error if it doesn't)
	_ = godotenv.Load()

	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
