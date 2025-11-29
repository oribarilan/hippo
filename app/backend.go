package main

// Backend defines the interface for work item data sources.
// Implementations include AzureDevOpsClient (production) and DummyBackend (development).
type Backend interface {
	// Work Item CRUD Operations
	GetWorkItemByID(id int) (*WorkItem, error)
	GetWorkItems() ([]WorkItem, error)
	GetWorkItemsForSprint(sprintPath string, excludeIDs []int, limit int) ([]WorkItem, error)
	GetWorkItemsExcluding(excludeIDs []int, sprintPath string, limit int) ([]WorkItem, error)
	GetWorkItemsCount() (int, error)
	GetWorkItemsCountForSprint(sprintPath string) (int, error)
	UpdateWorkItemState(workItemID int, newState string) error
	UpdateWorkItem(workItemID int, updates map[string]interface{}) error
	CreateWorkItem(title string, workItemType string, iterationPath string, parentID *int, areaPath string) (*WorkItem, error)
	DeleteWorkItem(workItemID int) error
	MoveWorkItemToSprint(workItemID int, iterationPath string) error
	GetWorkItemTypeStates(workItemType string) ([]string, map[string]string, error)

	// Sprint Operations
	GetCurrentAndAdjacentSprints() (prev *Sprint, curr *Sprint, next *Sprint, err error)

	// Backlog Operations
	GetRecentBacklogItems(limit int) ([]WorkItem, error)
	GetRecentBacklogItemsExcluding(excludeIDs []int, limit int) ([]WorkItem, error)
	GetRecentBacklogItemsCount() (int, error)
	GetAbandonedWorkItems(currentSprintPath string, limit int) ([]WorkItem, error)
	GetAbandonedWorkItemsExcluding(excludeIDs []int, currentSprintPath string, limit int) ([]WorkItem, error)
	GetAbandonedWorkItemsCount(currentSprintPath string) (int, error)
}

// Compile-time check that AzureDevOpsClient implements Backend
var _ Backend = (*AzureDevOpsClient)(nil)
