package main

import (
	"fmt"
	"time"

	"github.com/microsoft/azure-devops-go-api/azuredevops/work"
)

// =============================================================================
// SPRINT OPERATIONS
// =============================================================================

// GetTeamIterations fetches iterations for the team
func (c *AzureDevOpsClient) GetTeamIterations() ([]work.TeamSettingsIteration, error) {
	iterations, err := c.workClient.GetTeamIterations(c.ctx, work.GetTeamIterationsArgs{
		Project: &c.project,
		Team:    &c.team,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get team iterations: %w", err)
	}

	if iterations == nil {
		return []work.TeamSettingsIteration{}, nil
	}

	return *iterations, nil
}

// GetCurrentAndAdjacentSprints returns previous, current, and next sprint
func (c *AzureDevOpsClient) GetCurrentAndAdjacentSprints() (prev *Sprint, curr *Sprint, next *Sprint, err error) {
	iterations, err := c.GetTeamIterations()
	if err != nil {
		return nil, nil, nil, err
	}

	if len(iterations) == 0 {
		return nil, nil, nil, nil
	}

	now := time.Now()
	var currentIdx = -1
	var currentIter *work.TeamSettingsIteration

	// Find current sprint
	for i, iter := range iterations {
		if iter.Attributes == nil {
			continue
		}

		startDate := iter.Attributes.StartDate
		finishDate := iter.Attributes.FinishDate

		if startDate != nil && finishDate != nil {
			start := startDate.Time
			finish := finishDate.Time

			// Truncate to start of day (removes time component) for date-only comparison
			startDateTrunc := start.Truncate(24 * time.Hour)
			finishDateTrunc := finish.Truncate(24 * time.Hour)
			nowDate := now.Truncate(24 * time.Hour)

			// Check if today falls within the sprint (inclusive of start and end dates)
			if !nowDate.Before(startDateTrunc) && !nowDate.After(finishDateTrunc) {
				currentIdx = i
				currentIter = &iterations[i]
				break
			}
		}
	}

	// If no current sprint found, use the most recent one
	if currentIdx == -1 && len(iterations) > 0 {
		currentIdx = len(iterations) - 1
		currentIter = &iterations[currentIdx]
	}

	// Convert to Sprint structs
	if currentIdx > 0 {
		prev = convertIterationToSprint(&iterations[currentIdx-1])
	}

	if currentIter != nil {
		curr = convertIterationToSprint(currentIter)
	}

	if currentIdx >= 0 && currentIdx < len(iterations)-1 {
		next = convertIterationToSprint(&iterations[currentIdx+1])
	}

	return prev, curr, next, nil
}

// convertIterationToSprint converts an Azure DevOps TeamSettingsIteration to our Sprint struct
func convertIterationToSprint(iter *work.TeamSettingsIteration) *Sprint {
	if iter == nil || iter.Name == nil || iter.Path == nil {
		return nil
	}

	sprint := &Sprint{
		Name: *iter.Name,
		Path: *iter.Path,
	}

	if iter.Attributes != nil {
		if iter.Attributes.StartDate != nil {
			sprint.StartDate = iter.Attributes.StartDate.Time.Format("2006-01-02")
		}
		if iter.Attributes.FinishDate != nil {
			sprint.EndDate = iter.Attributes.FinishDate.Time.Format("2006-01-02")
		}
	}

	return sprint
}
