package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/microsoft/azure-devops-go-api/azuredevops/workitemtracking"
)

// =============================================================================
// BACKLOG OPERATIONS
// =============================================================================

// GetRecentBacklogItems returns work items not assigned to any sprint, created or updated in the last N days
func (c *AzureDevOpsClient) GetRecentBacklogItems(limit int) ([]WorkItem, error) {
	return c.GetRecentBacklogItemsExcluding(nil, limit)
}

// GetRecentBacklogItemsExcluding returns recent backlog items excluding specific IDs
func (c *AzureDevOpsClient) GetRecentBacklogItemsExcluding(excludeIDs []int, limit int) ([]WorkItem, error) {
	if limit > 30 {
		limit = 30
	}

	// Calculate date 30 days ago
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30).Format("2006-01-02")

	query := fmt.Sprintf(`
		SELECT [System.Id]
		FROM WorkItems
		WHERE [System.TeamProject] = '%s'
		AND [System.AssignedTo] = @Me
		AND [System.State] <> 'Closed'
		AND [System.State] <> 'Removed'
		AND [System.State] <> 'Completed'
		AND [System.State] <> 'Done'
		AND [System.IterationPath] = '%s'
		AND ([System.CreatedDate] >= '%s' OR [System.ChangedDate] >= '%s')`, c.project, c.project, thirtyDaysAgo, thirtyDaysAgo)

	// Add exclusion clause if there are IDs to exclude
	if len(excludeIDs) > 0 {
		var idStrs []string
		for _, id := range excludeIDs {
			idStrs = append(idStrs, fmt.Sprintf("%d", id))
		}
		query += fmt.Sprintf("\nAND [System.Id] NOT IN (%s)", strings.Join(idStrs, ","))
	}

	query += "\nORDER BY [System.ChangedDate] DESC"

	return c.executeWorkItemQuery(query, limit)
}

// GetRecentBacklogItemsCount returns the count of recent backlog items
func (c *AzureDevOpsClient) GetRecentBacklogItemsCount() (int, error) {
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30).Format("2006-01-02")

	query := fmt.Sprintf(`
		SELECT [System.Id]
		FROM WorkItems
		WHERE [System.TeamProject] = '%s'
		AND [System.AssignedTo] = @Me
		AND [System.State] <> 'Closed'
		AND [System.State] <> 'Removed'
		AND [System.State] <> 'Completed'
		AND [System.State] <> 'Done'
		AND [System.IterationPath] = '%s'
		AND ([System.CreatedDate] >= '%s' OR [System.ChangedDate] >= '%s')`, c.project, c.project, thirtyDaysAgo, thirtyDaysAgo)

	return c.executeCountQuery(query)
}

// GetAbandonedWorkItems returns work items not in current sprint, not updated in 14+ days
// TODO: Make staleDays (14) configurable when we add configuration support
func (c *AzureDevOpsClient) GetAbandonedWorkItems(currentSprintPath string, limit int) ([]WorkItem, error) {
	return c.GetAbandonedWorkItemsExcluding(nil, currentSprintPath, limit)
}

// GetAbandonedWorkItemsExcluding returns abandoned work items excluding specific IDs
func (c *AzureDevOpsClient) GetAbandonedWorkItemsExcluding(excludeIDs []int, currentSprintPath string, limit int) ([]WorkItem, error) {
	if limit > 30 {
		limit = 30
	}

	// Calculate date 14 days ago
	// TODO: Make this configurable
	fourteenDaysAgo := time.Now().AddDate(0, 0, -14).Format("2006-01-02")

	query := fmt.Sprintf(`
		SELECT [System.Id]
		FROM WorkItems
		WHERE [System.TeamProject] = '%s'
		AND [System.AssignedTo] = @Me
		AND [System.State] <> 'Closed'
		AND [System.State] <> 'Removed'
		AND [System.State] <> 'Completed'
		AND [System.State] <> 'Done'
		AND [System.ChangedDate] < '%s'`, c.project, fourteenDaysAgo)

	// Exclude items in current sprint if provided
	if currentSprintPath != "" {
		query += fmt.Sprintf("\nAND [System.IterationPath] <> '%s'", currentSprintPath)
	}

	// Add exclusion clause if there are IDs to exclude
	if len(excludeIDs) > 0 {
		var idStrs []string
		for _, id := range excludeIDs {
			idStrs = append(idStrs, fmt.Sprintf("%d", id))
		}
		query += fmt.Sprintf("\nAND [System.Id] NOT IN (%s)", strings.Join(idStrs, ","))
	}

	query += "\nORDER BY [System.ChangedDate] ASC"

	return c.executeWorkItemQuery(query, limit)
}

// GetAbandonedWorkItemsCount returns the count of abandoned work items
func (c *AzureDevOpsClient) GetAbandonedWorkItemsCount(currentSprintPath string) (int, error) {
	// TODO: Make staleDays (14) configurable when we add configuration support
	fourteenDaysAgo := time.Now().AddDate(0, 0, -14).Format("2006-01-02")

	query := fmt.Sprintf(`
		SELECT [System.Id]
		FROM WorkItems
		WHERE [System.TeamProject] = '%s'
		AND [System.AssignedTo] = @Me
		AND [System.State] <> 'Closed'
		AND [System.State] <> 'Removed'
		AND [System.State] <> 'Completed'
		AND [System.State] <> 'Done'
		AND [System.ChangedDate] < '%s'`, c.project, fourteenDaysAgo)

	if currentSprintPath != "" {
		query += fmt.Sprintf("\nAND [System.IterationPath] <> '%s'", currentSprintPath)
	}

	return c.executeCountQuery(query)
}

// =============================================================================
// HELPER FUNCTIONS FOR QUERIES
// =============================================================================

// executeWorkItemQuery is a helper to execute a WIQL query and return WorkItems
func (c *AzureDevOpsClient) executeWorkItemQuery(query string, limit int) ([]WorkItem, error) {
	wiql := workitemtracking.Wiql{
		Query: strPtr(query),
	}

	queryArgs := workitemtracking.QueryByWiqlArgs{
		Wiql: &wiql,
		Top:  &limit,
	}

	result, err := c.workItemClient.QueryByWiql(c.ctx, queryArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to query work items: %w", err)
	}

	if result.WorkItems == nil || len(*result.WorkItems) == 0 {
		return []WorkItem{}, nil
	}

	// Extract work item IDs
	var ids []int
	for _, ref := range *result.WorkItems {
		if ref.Id != nil {
			ids = append(ids, *ref.Id)
		}
	}

	if len(ids) == 0 {
		return []WorkItem{}, nil
	}

	// Get full work item details
	workItemsArgs := workitemtracking.GetWorkItemsArgs{
		Ids:    &ids,
		Expand: &workitemtracking.WorkItemExpandValues.All,
	}

	workItems, err := c.workItemClient.GetWorkItems(c.ctx, workItemsArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to get work item details: %w", err)
	}

	// Convert to our WorkItem struct
	var tasks []WorkItem
	for _, wi := range *workItems {
		task := c.convertWorkItem(wi)
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// executeCountQuery is a helper to execute a WIQL query and return count
func (c *AzureDevOpsClient) executeCountQuery(query string) (int, error) {
	wiql := workitemtracking.Wiql{
		Query: strPtr(query),
	}

	queryArgs := workitemtracking.QueryByWiqlArgs{
		Wiql: &wiql,
	}

	result, err := c.workItemClient.QueryByWiql(c.ctx, queryArgs)
	if err != nil {
		return 0, fmt.Errorf("failed to query work items count: %w", err)
	}

	if result.WorkItems == nil {
		return 0, nil
	}

	return len(*result.WorkItems), nil
}
