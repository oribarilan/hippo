package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Message Types
// These messages are sent by commands to communicate asynchronous results back to the Update function

type tasksLoadedMsg struct {
	tasks         []WorkItem
	err           error
	client        Backend
	totalCount    int
	append        bool
	forTab        *sprintTab  // Which sprint tab these tasks are for
	forBacklogTab *backlogTab // Which backlog tab these tasks are for
}

type stateUpdatedMsg struct {
	err error
}

type workItemUpdatedMsg struct {
	err error
}

type workItemRefreshedMsg struct {
	workItem *WorkItem
	err      error
}

type workItemCreatedMsg struct {
	workItem *WorkItem
	err      error
}

type workItemDeletedMsg struct {
	workItemID int
	err        error
}

type sprintUpdatedMsg struct {
	workItemID int
	err        error
}

type statesLoadedMsg struct {
	states          []string
	stateCategories map[string]string
	err             error
}

type sprintsLoadedMsg struct {
	previousSprint *Sprint
	currentSprint  *Sprint
	nextSprint     *Sprint
	err            error
	client         Backend
	forceReload    bool // Force reload even if sprints already exist
}

// Command Functions
// These functions return tea.Cmd that perform asynchronous operations and return messages

func loadTasks(client Backend) tea.Cmd {
	return loadTasksForSprint(client, nil, "", defaultLoadLimit, nil)
}

func loadTasksForSprint(client Backend, excludeIDs []int, sprintPath string, limit int, forTab *sprintTab) tea.Cmd {
	return func() tea.Msg {
		tasks, err := client.GetWorkItemsExcluding(excludeIDs, sprintPath, limit)
		if err != nil {
			return tasksLoadedMsg{err: err, append: len(excludeIDs) > 0, forTab: forTab}
		}

		totalCount, countErr := client.GetWorkItemsCountForSprint(sprintPath)
		if countErr != nil {
			// If count fails, just use the tasks we got
			totalCount = len(tasks)
		}

		return tasksLoadedMsg{tasks: tasks, err: err, client: client, totalCount: totalCount, append: len(excludeIDs) > 0, forTab: forTab}
	}
}

func loadInitialTasksForSprint(client Backend, sprintPath string, tab sprintTab) tea.Cmd {
	tabCopy := tab
	return loadTasksForSprint(client, nil, sprintPath, defaultLoadLimit, &tabCopy)
}

func loadTasksForBacklogTab(client Backend, tab backlogTab, currentSprintPath string) tea.Cmd {
	return func() tea.Msg {
		var tasks []WorkItem
		var err error
		var totalCount int

		switch tab {
		case recentBacklog:
			tasks, err = client.GetRecentBacklogItems(defaultLoadLimit)
			if err == nil {
				totalCount, _ = client.GetRecentBacklogItemsCount()
			}
		case abandonedWork:
			tasks, err = client.GetAbandonedWorkItems(currentSprintPath, defaultLoadLimit)
			if err == nil {
				totalCount, _ = client.GetAbandonedWorkItemsCount(currentSprintPath)
			}
		}

		if err != nil {
			return tasksLoadedMsg{err: err, append: false, forTab: nil, forBacklogTab: &tab}
		}

		if totalCount == 0 {
			totalCount = len(tasks)
		}

		tabCopy := tab
		return tasksLoadedMsg{tasks: tasks, err: err, client: client, totalCount: totalCount, append: false, forTab: nil, forBacklogTab: &tabCopy}
	}
}

func loadMoreBacklogItems(client Backend, tab backlogTab, currentSprintPath string, excludeIDs []int) tea.Cmd {
	return func() tea.Msg {
		var tasks []WorkItem
		var err error
		var totalCount int

		switch tab {
		case recentBacklog:
			tasks, err = client.GetRecentBacklogItemsExcluding(excludeIDs, defaultLoadLimit)
			if err == nil {
				totalCount, _ = client.GetRecentBacklogItemsCount()
			}
		case abandonedWork:
			tasks, err = client.GetAbandonedWorkItemsExcluding(excludeIDs, currentSprintPath, defaultLoadLimit)
			if err == nil {
				totalCount, _ = client.GetAbandonedWorkItemsCount(currentSprintPath)
			}
		}

		if err != nil {
			return tasksLoadedMsg{err: err, append: true, forTab: nil, forBacklogTab: &tab}
		}

		if totalCount == 0 {
			totalCount = len(tasks)
		}

		tabCopy := tab
		return tasksLoadedMsg{tasks: tasks, err: err, client: client, totalCount: totalCount, append: true, forTab: nil, forBacklogTab: &tabCopy}
	}
}

func loadWorkItemStates(client Backend, workItemType string) tea.Cmd {
	return func() tea.Msg {
		states, categories, err := client.GetWorkItemTypeStates(workItemType)
		return statesLoadedMsg{states: states, stateCategories: categories, err: err}
	}
}

func loadSprints(client Backend) tea.Cmd {
	return loadSprintsWithReload(client, false)
}

func loadSprintsWithReload(client Backend, forceReload bool) tea.Cmd {
	return func() tea.Msg {
		prev, curr, next, err := client.GetCurrentAndAdjacentSprints()

		return sprintsLoadedMsg{
			previousSprint: prev,
			currentSprint:  curr,
			nextSprint:     next,
			err:            err,
			client:         client,
			forceReload:    forceReload,
		}
	}
}

func updateWorkItemState(client Backend, workItemID int, newState string) tea.Cmd {
	return func() tea.Msg {
		err := client.UpdateWorkItemState(workItemID, newState)
		return stateUpdatedMsg{err: err}
	}
}

func updateWorkItem(client Backend, workItemID int, updates map[string]interface{}) tea.Cmd {
	return func() tea.Msg {
		err := client.UpdateWorkItem(workItemID, updates)
		return workItemUpdatedMsg{err: err}
	}
}

func refreshWorkItem(client Backend, workItemID int) tea.Cmd {
	return func() tea.Msg {
		workItem, err := client.GetWorkItemByID(workItemID)
		return workItemRefreshedMsg{workItem: workItem, err: err}
	}
}

func createWorkItem(client Backend, title string, workItemType string, iterationPath string, parentID *int, areaPath string) tea.Cmd {
	return func() tea.Msg {
		workItem, err := client.CreateWorkItem(title, workItemType, iterationPath, parentID, areaPath)
		return workItemCreatedMsg{workItem: workItem, err: err}
	}
}

func deleteWorkItem(client Backend, workItemID int) tea.Cmd {
	return func() tea.Msg {
		err := client.DeleteWorkItem(workItemID)
		return workItemDeletedMsg{workItemID: workItemID, err: err}
	}
}

func moveWorkItemToSprint(client Backend, workItemID int, iterationPath string) tea.Cmd {
	return func() tea.Msg {
		err := client.MoveWorkItemToSprint(workItemID, iterationPath)
		return sprintUpdatedMsg{workItemID: workItemID, err: err}
	}
}
