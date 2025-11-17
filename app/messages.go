package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Message Types
// These messages are sent by commands to communicate asynchronous results back to the Update function

type tasksLoadedMsg struct {
	tasks         []WorkItem
	err           error
	client        *AzureDevOpsClient
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
	client         *AzureDevOpsClient
	forceReload    bool // Force reload even if sprints already exist
}

// Command Functions
// These functions return tea.Cmd that perform asynchronous operations and return messages

func loadTasks(client *AzureDevOpsClient) tea.Cmd {
	return loadTasksForSprint(client, nil, "", defaultLoadLimit, nil)
}

func loadTasksForSprint(client *AzureDevOpsClient, excludeIDs []int, sprintPath string, limit int, forTab *sprintTab) tea.Cmd {
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

func loadInitialTasksForSprint(client *AzureDevOpsClient, sprintPath string, tab sprintTab) tea.Cmd {
	tabCopy := tab
	return loadTasksForSprint(client, nil, sprintPath, defaultLoadLimit, &tabCopy)
}

func loadTasksForBacklogTab(client *AzureDevOpsClient, tab backlogTab, currentSprintPath string) tea.Cmd {
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

func loadMoreBacklogItems(client *AzureDevOpsClient, tab backlogTab, currentSprintPath string, excludeIDs []int) tea.Cmd {
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

func loadWorkItemStates(client *AzureDevOpsClient, workItemType string) tea.Cmd {
	return func() tea.Msg {
		states, categories, err := client.GetWorkItemTypeStates(workItemType)
		return statesLoadedMsg{states: states, stateCategories: categories, err: err}
	}
}

func loadSprints(client *AzureDevOpsClient) tea.Cmd {
	return loadSprintsWithReload(client, false)
}

func loadSprintsWithReload(client *AzureDevOpsClient, forceReload bool) tea.Cmd {
	return func() tea.Msg {
		prev, curr, next, err := client.GetCurrentAndAdjacentSprints()

		var prevSprint, currSprint, nextSprint *Sprint

		if prev != nil && prev.Name != nil && prev.Path != nil {
			prevSprint = &Sprint{
				Name: *prev.Name,
				Path: *prev.Path,
			}
			if prev.Attributes != nil {
				if prev.Attributes.StartDate != nil {
					prevSprint.StartDate = prev.Attributes.StartDate.Time.Format("2006-01-02")
				}
				if prev.Attributes.FinishDate != nil {
					prevSprint.EndDate = prev.Attributes.FinishDate.Time.Format("2006-01-02")
				}
			}
		}

		if curr != nil && curr.Name != nil && curr.Path != nil {
			currSprint = &Sprint{
				Name: *curr.Name,
				Path: *curr.Path,
			}
			if curr.Attributes != nil {
				if curr.Attributes.StartDate != nil {
					currSprint.StartDate = curr.Attributes.StartDate.Time.Format("2006-01-02")
				}
				if curr.Attributes.FinishDate != nil {
					currSprint.EndDate = curr.Attributes.FinishDate.Time.Format("2006-01-02")
				}
			}
		}

		if next != nil && next.Name != nil && next.Path != nil {
			nextSprint = &Sprint{
				Name: *next.Name,
				Path: *next.Path,
			}
			if next.Attributes != nil {
				if next.Attributes.StartDate != nil {
					nextSprint.StartDate = next.Attributes.StartDate.Time.Format("2006-01-02")
				}
				if next.Attributes.FinishDate != nil {
					nextSprint.EndDate = next.Attributes.FinishDate.Time.Format("2006-01-02")
				}
			}
		}

		return sprintsLoadedMsg{
			previousSprint: prevSprint,
			currentSprint:  currSprint,
			nextSprint:     nextSprint,
			err:            err,
			client:         client,
			forceReload:    forceReload,
		}
	}
}

func updateWorkItemState(client *AzureDevOpsClient, workItemID int, newState string) tea.Cmd {
	return func() tea.Msg {
		err := client.UpdateWorkItemState(workItemID, newState)
		return stateUpdatedMsg{err: err}
	}
}

func updateWorkItem(client *AzureDevOpsClient, workItemID int, updates map[string]interface{}) tea.Cmd {
	return func() tea.Msg {
		err := client.UpdateWorkItem(workItemID, updates)
		return workItemUpdatedMsg{err: err}
	}
}

func refreshWorkItem(client *AzureDevOpsClient, workItemID int) tea.Cmd {
	return func() tea.Msg {
		workItem, err := client.GetWorkItemByID(workItemID)
		return workItemRefreshedMsg{workItem: workItem, err: err}
	}
}

func createWorkItem(client *AzureDevOpsClient, title string, workItemType string, iterationPath string, parentID *int, areaPath string) tea.Cmd {
	return func() tea.Msg {
		workItem, err := client.CreateWorkItem(title, workItemType, iterationPath, parentID, areaPath)
		return workItemCreatedMsg{workItem: workItem, err: err}
	}
}

func deleteWorkItem(client *AzureDevOpsClient, workItemID int) tea.Cmd {
	return func() tea.Msg {
		err := client.DeleteWorkItem(workItemID)
		return workItemDeletedMsg{workItemID: workItemID, err: err}
	}
}

func moveWorkItemToSprint(client *AzureDevOpsClient, workItemID int, iterationPath string) tea.Cmd {
	return func() tea.Msg {
		err := client.MoveWorkItemToSprint(workItemID, iterationPath)
		return sprintUpdatedMsg{workItemID: workItemID, err: err}
	}
}
