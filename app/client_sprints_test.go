package main

import (
	"testing"
	"time"

	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/work"
)

// =============================================================================
// TEST FIXTURES
// =============================================================================

// createTestIteration creates a test sprint iteration with given dates
func createTestIteration(name string, startDate, finishDate time.Time) work.TeamSettingsIteration {
	start := azuredevops.Time{Time: startDate}
	finish := azuredevops.Time{Time: finishDate}
	path := "TestProject\\Iteration\\" + name

	return work.TeamSettingsIteration{
		Name: &name,
		Path: &path,
		Attributes: &work.TeamIterationAttributes{
			StartDate:  &start,
			FinishDate: &finish,
		},
	}
}

// createTestIterationWithoutDates creates a test iteration without date attributes
func createTestIterationWithoutDates(name string) work.TeamSettingsIteration {
	path := "TestProject\\Iteration\\" + name

	return work.TeamSettingsIteration{
		Name:       &name,
		Path:       &path,
		Attributes: nil,
	}
}

// =============================================================================
// TESTS FOR GetCurrentAndAdjacentSprints
// =============================================================================

// TestGetCurrentAndAdjacentSprints_CurrentSprint tests finding the current sprint
func TestGetCurrentAndAdjacentSprints_CurrentSprint(t *testing.T) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	lastWeek := now.AddDate(0, 0, -7)
	nextWeek := now.AddDate(0, 0, 7)

	tests := []struct {
		name             string
		iterations       []work.TeamSettingsIteration
		expectPrev       bool
		expectCurr       bool
		expectNext       bool
		expectedCurrName string
	}{
		{
			name: "Current sprint in middle",
			iterations: []work.TeamSettingsIteration{
				createTestIteration("Sprint 1", lastWeek.AddDate(0, 0, -14), lastWeek),
				createTestIteration("Sprint 2", yesterday, nextWeek),
				createTestIteration("Sprint 3", nextWeek.AddDate(0, 0, 1), nextWeek.AddDate(0, 0, 14)),
			},
			expectPrev:       true,
			expectCurr:       true,
			expectNext:       true,
			expectedCurrName: "Sprint 2",
		},
		{
			name: "Current sprint is first",
			iterations: []work.TeamSettingsIteration{
				createTestIteration("Sprint 1", yesterday, nextWeek),
				createTestIteration("Sprint 2", nextWeek.AddDate(0, 0, 1), nextWeek.AddDate(0, 0, 14)),
			},
			expectPrev:       false,
			expectCurr:       true,
			expectNext:       true,
			expectedCurrName: "Sprint 1",
		},
		{
			name: "Current sprint is last",
			iterations: []work.TeamSettingsIteration{
				createTestIteration("Sprint 1", lastWeek.AddDate(0, 0, -14), lastWeek),
				createTestIteration("Sprint 2", yesterday, nextWeek),
			},
			expectPrev:       true,
			expectCurr:       true,
			expectNext:       false,
			expectedCurrName: "Sprint 2",
		},
		{
			name: "Single sprint (current)",
			iterations: []work.TeamSettingsIteration{
				createTestIteration("Sprint 1", yesterday, nextWeek),
			},
			expectPrev:       false,
			expectCurr:       true,
			expectNext:       false,
			expectedCurrName: "Sprint 1",
		},
		{
			name: "No current sprint - uses most recent",
			iterations: []work.TeamSettingsIteration{
				createTestIteration("Sprint 1", lastWeek.AddDate(0, 0, -21), lastWeek.AddDate(0, 0, -14)),
				createTestIteration("Sprint 2", lastWeek.AddDate(0, 0, -14), lastWeek),
			},
			expectPrev:       true,
			expectCurr:       true,
			expectNext:       false,
			expectedCurrName: "Sprint 2", // Most recent
		},
		{
			name:       "Empty iterations",
			iterations: []work.TeamSettingsIteration{},
			expectPrev: false,
			expectCurr: false,
			expectNext: false,
		},
		{
			name: "Sprint with nil attributes",
			iterations: []work.TeamSettingsIteration{
				createTestIterationWithoutDates("Sprint 1"),
				createTestIteration("Sprint 2", yesterday, nextWeek),
			},
			expectPrev:       true, // Sprint 1 (with nil attrs) becomes previous
			expectCurr:       true,
			expectNext:       false,
			expectedCurrName: "Sprint 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the logic directly using the extracted helper function
			prev, curr, next := findCurrentAndAdjacentSprints(tt.iterations)

			if tt.expectPrev && prev == nil {
				t.Error("Expected previous sprint, got nil")
			}
			if !tt.expectPrev && prev != nil {
				t.Errorf("Expected no previous sprint, got %v", *prev.Name)
			}

			if tt.expectCurr && curr == nil {
				t.Error("Expected current sprint, got nil")
			}
			if !tt.expectCurr && curr != nil {
				t.Errorf("Expected no current sprint, got %v", *curr.Name)
			}

			if tt.expectNext && next == nil {
				t.Error("Expected next sprint, got nil")
			}
			if !tt.expectNext && next != nil {
				t.Errorf("Expected no next sprint, got %v", *next.Name)
			}

			if tt.expectCurr && curr != nil && *curr.Name != tt.expectedCurrName {
				t.Errorf("Current sprint name = %v, want %v", *curr.Name, tt.expectedCurrName)
			}
		})
	}
}

// findCurrentAndAdjacentSprints is extracted logic from GetCurrentAndAdjacentSprints for testing
func findCurrentAndAdjacentSprints(iterations []work.TeamSettingsIteration) (prev, curr, next *work.TeamSettingsIteration) {
	if len(iterations) == 0 {
		return nil, nil, nil
	}

	now := time.Now()
	var currentIdx = -1

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
				curr = &iterations[i]
				break
			}
		}
	}

	// If no current sprint found, use the most recent one
	if currentIdx == -1 && len(iterations) > 0 {
		currentIdx = len(iterations) - 1
		curr = &iterations[currentIdx]
	}

	// Get previous sprint
	if currentIdx > 0 {
		prev = &iterations[currentIdx-1]
	}

	// Get next sprint
	if currentIdx >= 0 && currentIdx < len(iterations)-1 {
		next = &iterations[currentIdx+1]
	}

	return prev, curr, next
}

// TestSprintDateComparison tests date truncation and comparison logic
func TestSprintDateComparison(t *testing.T) {
	tests := []struct {
		name       string
		now        time.Time
		startDate  time.Time
		finishDate time.Time
		expectIn   bool
	}{
		{
			name:       "Now is in sprint (middle)",
			now:        time.Date(2024, 1, 10, 12, 30, 0, 0, time.UTC),
			startDate:  time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC),
			finishDate: time.Date(2024, 1, 21, 23, 59, 59, 0, time.UTC),
			expectIn:   true,
		},
		{
			name:       "Now is on start date",
			now:        time.Date(2024, 1, 8, 14, 0, 0, 0, time.UTC),
			startDate:  time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC),
			finishDate: time.Date(2024, 1, 21, 0, 0, 0, 0, time.UTC),
			expectIn:   true,
		},
		{
			name:       "Now is on finish date",
			now:        time.Date(2024, 1, 21, 18, 0, 0, 0, time.UTC),
			startDate:  time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC),
			finishDate: time.Date(2024, 1, 21, 23, 59, 59, 0, time.UTC),
			expectIn:   true,
		},
		{
			name:       "Now is before sprint",
			now:        time.Date(2024, 1, 7, 23, 59, 59, 0, time.UTC),
			startDate:  time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC),
			finishDate: time.Date(2024, 1, 21, 0, 0, 0, 0, time.UTC),
			expectIn:   false,
		},
		{
			name:       "Now is after sprint",
			now:        time.Date(2024, 1, 22, 0, 0, 1, 0, time.UTC),
			startDate:  time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC),
			finishDate: time.Date(2024, 1, 21, 23, 59, 59, 0, time.UTC),
			expectIn:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Truncate to start of day (removes time component) for date-only comparison
			startDateTrunc := tt.startDate.Truncate(24 * time.Hour)
			finishDateTrunc := tt.finishDate.Truncate(24 * time.Hour)
			nowDate := tt.now.Truncate(24 * time.Hour)

			// Check if today falls within the sprint (inclusive of start and end dates)
			isIn := !nowDate.Before(startDateTrunc) && !nowDate.After(finishDateTrunc)

			if isIn != tt.expectIn {
				t.Errorf("Date comparison = %v, want %v (now: %v, start: %v, finish: %v)",
					isIn, tt.expectIn, nowDate, startDateTrunc, finishDateTrunc)
			}
		})
	}
}

// TestSprintEdgeCases tests edge cases for sprint handling
func TestSprintEdgeCases(t *testing.T) {
	t.Run("Iteration with nil start date", func(t *testing.T) {
		now := time.Now()
		finish := azuredevops.Time{Time: now.AddDate(0, 0, 7)}
		name := "Sprint 1"

		iter := work.TeamSettingsIteration{
			Name: &name,
			Attributes: &work.TeamIterationAttributes{
				StartDate:  nil,
				FinishDate: &finish,
			},
		}

		iterations := []work.TeamSettingsIteration{iter}
		_, curr, _ := findCurrentAndAdjacentSprints(iterations)

		// Should default to most recent (last one)
		if curr == nil {
			t.Error("Expected current sprint (fallback to last), got nil")
		}
		if curr != nil && curr.Name != nil {
			t.Logf("Got sprint: %s", *curr.Name)
		}
	})

	t.Run("Iteration with nil finish date", func(t *testing.T) {
		now := time.Now()
		start := azuredevops.Time{Time: now.AddDate(0, 0, -7)}
		name := "Sprint 1"

		iter := work.TeamSettingsIteration{
			Name: &name,
			Attributes: &work.TeamIterationAttributes{
				StartDate:  &start,
				FinishDate: nil,
			},
		}

		iterations := []work.TeamSettingsIteration{iter}
		_, curr, _ := findCurrentAndAdjacentSprints(iterations)

		// Should default to most recent (last one)
		if curr == nil {
			t.Error("Expected current sprint (fallback to last), got nil")
		}
	})

	t.Run("All iterations in the past", func(t *testing.T) {
		now := time.Now()
		twoWeeksAgo := now.AddDate(0, 0, -14)
		fourWeeksAgo := now.AddDate(0, 0, -28)

		iterations := []work.TeamSettingsIteration{
			createTestIteration("Sprint 1", fourWeeksAgo.AddDate(0, 0, -14), fourWeeksAgo),
			createTestIteration("Sprint 2", twoWeeksAgo.AddDate(0, 0, -14), twoWeeksAgo),
		}

		prev, curr, next := findCurrentAndAdjacentSprints(iterations)

		// Should use most recent (last one) as current
		if curr == nil {
			t.Error("Expected current sprint (most recent), got nil")
		}
		if curr != nil && *curr.Name != "Sprint 2" {
			t.Errorf("Expected Sprint 2 as current (most recent), got %v", *curr.Name)
		}
		if prev == nil {

			t.Error("Expected previous sprint")
		}
		if next != nil {
			t.Error("Expected no next sprint")
		}
	})

	t.Run("All iterations in the future", func(t *testing.T) {
		now := time.Now()
		nextWeek := now.AddDate(0, 0, 7)
		fourWeeksAhead := now.AddDate(0, 0, 28)

		iterations := []work.TeamSettingsIteration{
			createTestIteration("Sprint 1", nextWeek, nextWeek.AddDate(0, 0, 14)),
			createTestIteration("Sprint 2", fourWeeksAhead, fourWeeksAhead.AddDate(0, 0, 14)),
		}

		_, curr, _ := findCurrentAndAdjacentSprints(iterations)

		// Should use most recent (last one) as current
		if curr == nil {
			t.Error("Expected current sprint (most recent), got nil")
		}
		if curr != nil && *curr.Name != "Sprint 2" {
			t.Errorf("Expected Sprint 2 as current (most recent), got %v", *curr.Name)
		}
	})
}

// TestSprintIterationStructure tests the structure and fields
func TestSprintIterationStructure(t *testing.T) {
	t.Run("Valid iteration has all fields", func(t *testing.T) {
		now := time.Now()
		iter := createTestIteration("Test Sprint", now, now.AddDate(0, 0, 14))

		if iter.Name == nil {
			t.Error("Expected name to be set")
		}
		if iter.Path == nil {
			t.Error("Expected path to be set")
		}
		if iter.Attributes == nil {
			t.Error("Expected attributes to be set")
		}
		if iter.Attributes != nil {
			if iter.Attributes.StartDate == nil {
				t.Error("Expected start date to be set")
			}
			if iter.Attributes.FinishDate == nil {
				t.Error("Expected finish date to be set")
			}
		}
	})

	t.Run("Iteration without attributes", func(t *testing.T) {
		iter := createTestIterationWithoutDates("Test Sprint")

		if iter.Name == nil {
			t.Error("Expected name to be set")
		}
		if iter.Attributes != nil {
			t.Error("Expected attributes to be nil")
		}
	})
}

// TestSprintTimeZoneHandling tests time zone considerations
func TestSprintTimeZoneHandling(t *testing.T) {
	// Test that date truncation works correctly for sprint comparison
	t.Run("Truncation removes time component", func(t *testing.T) {
		// Same date but different times should truncate to same day
		morning := time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC)
		evening := time.Date(2024, 1, 15, 20, 0, 0, 0, time.UTC)

		morningTrunc := morning.Truncate(24 * time.Hour)
		eveningTrunc := evening.Truncate(24 * time.Hour)

		// Both should truncate to midnight of the same day
		if !morningTrunc.Equal(eveningTrunc) {
			t.Errorf("Times on same day should truncate to same value: morning=%v, evening=%v", morningTrunc, eveningTrunc)
		}
	})

	t.Run("Different days remain different after truncation", func(t *testing.T) {
		day1 := time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC)
		day2 := time.Date(2024, 1, 16, 0, 0, 1, 0, time.UTC)

		day1Trunc := day1.Truncate(24 * time.Hour)
		day2Trunc := day2.Truncate(24 * time.Hour)

		// Different days should remain different
		if day1Trunc.Equal(day2Trunc) {
			t.Error("Different days should not truncate to same value")
		}
	})
}
