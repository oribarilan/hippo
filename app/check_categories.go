// This file is a standalone utility script for checking Azure DevOps state categories
// To run it: go run check_categories.go
// It's not part of the main application build

//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/joho/godotenv"
	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/workitemtracking"
)

func getAzureCliToken() (string, error) {
	cmd := exec.Command("az", "account", "get-access-token", "--resource", "499b84ac-1321-427f-aa17-267ca6975798", "--query", "accessToken", "-o", "tsv")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func main() {
	_ = godotenv.Load()

	organizationURL := os.Getenv("HIPPO_ADO_ORG_URL")
	project := os.Getenv("HIPPO_ADO_PROJECT")

	if organizationURL == "" || project == "" {
		fmt.Println("Error: HIPPO_ADO_ORG_URL and HIPPO_ADO_PROJECT must be set in .env")
		os.Exit(1)
	}

	fmt.Printf("Organization: %s\n", organizationURL)
	fmt.Printf("Project: %s\n\n", project)

	token, err := getAzureCliToken()
	if err != nil {
		fmt.Printf("Error getting Azure CLI token: %v\n", err)
		fmt.Println("Please run: az login")
		os.Exit(1)
	}

	connection := azuredevops.NewPatConnection(organizationURL, token)
	ctx := context.Background()

	client, err := workitemtracking.NewClient(ctx, connection)
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		os.Exit(1)
	}

	// Try common work item types
	types := []string{"Task", "Bug", "User Story", "Product Backlog Item", "Issue", "Epic", "Feature"}

	foundAny := false
	for _, workItemType := range types {
		getTypeArgs := workitemtracking.GetWorkItemTypeArgs{
			Project: &project,
			Type:    &workItemType,
		}

		workItemTypeDef, err := client.GetWorkItemType(ctx, getTypeArgs)
		if err != nil {
			continue
		}

		foundAny = true
		fmt.Printf("╔═══════════════════════════════════════════════════════════════╗\n")
		fmt.Printf("║  Work Item Type: %-40s ║\n", workItemType)
		fmt.Printf("╚═══════════════════════════════════════════════════════════════╝\n\n")

		if workItemTypeDef.States != nil && len(*workItemTypeDef.States) > 0 {
			fmt.Printf("  %-25s  %-20s  %-10s\n", "STATE NAME", "CATEGORY", "COLOR")
			fmt.Printf("  %-25s  %-20s  %-10s\n", "══════════", "════════", "═════")

			for _, state := range *workItemTypeDef.States {
				category := "Unknown"
				if state.Category != nil {
					category = *state.Category
				}
				color := "N/A"
				if state.Color != nil {
					color = *state.Color
				}
				stateName := "N/A"
				if state.Name != nil {
					stateName = *state.Name
				}

				// Add visual indicator for category
				categoryIcon := ""
				switch category {
				case "Proposed":
					categoryIcon = "○"
				case "InProgress":
					categoryIcon = "●"
				case "Resolved":
					categoryIcon = "◐"
				case "Completed":
					categoryIcon = "✓"
				}

				fmt.Printf("  %-25s  %s %-18s  #%s\n", stateName, categoryIcon, category, color)
			}
			fmt.Println()
		} else {
			fmt.Println("  No states found for this work item type.")
			fmt.Println()
		}
	}

	if !foundAny {
		fmt.Println("❌ No work item types found. Possible reasons:")
		fmt.Println("   - Project name is incorrect")
		fmt.Println("   - You don't have access to this project")
		fmt.Println("   - The project uses custom work item types")
	}
}
