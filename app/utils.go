package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
