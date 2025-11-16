package main

import "github.com/charmbracelet/lipgloss"

// Color palette constants
const (
	ColorPurple      = "62"
	ColorWhite       = "230"
	ColorGray        = "241"
	ColorLightGray   = "251"
	ColorDimGray     = "243"
	ColorVeryDim     = "241"
	ColorGreen       = "86"
	ColorPink        = "212"
	ColorBlue        = "39"
	ColorBrightBlue  = "63"
	ColorOrange      = "208"
	ColorRed         = "196"
	ColorBrightWhite = "255"
)

// Styles holds all the lipgloss styles used throughout the application
type Styles struct {
	// Selection and interaction styles
	Selected lipgloss.Style
	Dim      lipgloss.Style

	// Tree structure styles
	TreeEdge         lipgloss.Style
	TreeEdgeSelected lipgloss.Style
	Icon             lipgloss.Style
	IconSelected     lipgloss.Style

	// State styles (based on work item state categories)
	ProposedState   lipgloss.Style
	InProgressState lipgloss.Style
	CompletedState  lipgloss.Style
	RemovedState    lipgloss.Style

	// Item title styles (for list view)
	ItemTitleProposed   lipgloss.Style
	ItemTitleInProgress lipgloss.Style
	ItemTitleCompleted  lipgloss.Style
	ItemTitleRemoved    lipgloss.Style

	// Mode and tab selector styles
	ActiveMode   lipgloss.Style
	InactiveMode lipgloss.Style
	ActiveTab    lipgloss.Style
	InactiveTab  lipgloss.Style

	// UI element styles
	Hint           lipgloss.Style
	StatusMsg      lipgloss.Style
	Loader         lipgloss.Style
	LoadMore       lipgloss.Style
	BatchIndicator lipgloss.Style

	// Help and messages
	Help      lipgloss.Style
	Separator lipgloss.Style
	Log       lipgloss.Style

	// Detail view styles
	Card        lipgloss.Style
	Header      lipgloss.Style
	Label       lipgloss.Style
	Value       lipgloss.Style
	Section     lipgloss.Style
	Description lipgloss.Style

	// Edit view styles
	EditLabel   lipgloss.Style
	EditHelp    lipgloss.Style
	EditSection lipgloss.Style

	// Error and warning styles
	Error   lipgloss.Style
	Warning lipgloss.Style
	Detail  lipgloss.Style

	// Key style for help text
	Key           lipgloss.Style
	Desc          lipgloss.Style
	SectionHeader lipgloss.Style
}

// NewStyles creates and returns a Styles struct with all styles initialized
func NewStyles() Styles {
	return Styles{
		// Selection and interaction
		Selected: lipgloss.NewStyle().
			Background(lipgloss.Color(ColorPurple)).
			Foreground(lipgloss.Color(ColorWhite)).
			Bold(true),

		Dim: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorGray)),

		// Tree structure
		TreeEdge: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorBrightBlue)),

		TreeEdgeSelected: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorWhite)).
			Background(lipgloss.Color(ColorPurple)),

		Icon: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPink)),

		IconSelected: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorWhite)).
			Background(lipgloss.Color(ColorPurple)),

		// State styles
		ProposedState: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorDimGray)),

		InProgressState: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorGreen)).
			Bold(true),

		CompletedState: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorDimGray)).
			Italic(true),

		RemovedState: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorVeryDim)).
			Italic(true),

		// Item title styles
		ItemTitleProposed: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorLightGray)),

		ItemTitleInProgress: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorBrightWhite)),

		ItemTitleCompleted: lipgloss.NewStyle().
			Strikethrough(true).
			Foreground(lipgloss.Color(ColorDimGray)),

		ItemTitleRemoved: lipgloss.NewStyle().
			Strikethrough(true).
			Foreground(lipgloss.Color(ColorVeryDim)),

		// Mode and tab selectors
		ActiveMode: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorWhite)).
			Background(lipgloss.Color(ColorPurple)).
			Padding(0, 2).
			MarginRight(1),

		InactiveMode: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Padding(0, 2).
			MarginRight(1),

		ActiveTab: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorWhite)).
			Background(lipgloss.Color(ColorPurple)).
			Padding(0, 2),

		InactiveTab: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Padding(0, 2),

		// UI elements
		Hint: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorLightGray)).
			Italic(true),

		StatusMsg: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorGreen)),

		Loader: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorGreen)).
			MarginLeft(2),

		LoadMore: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorGreen)).
			Italic(true),

		BatchIndicator: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorOrange)).
			Bold(true),

		// Help and messages
		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")),

		Separator: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")),

		Log: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true),

		// Detail view
		Card: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorPurple)).
			Padding(1, 2),

		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorBlue)).
			MarginBottom(1),

		Label: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorGray)).
			Width(15),

		Value: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorWhite)),

		Section: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginTop(1).
			MarginBottom(1),

		Description: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorLightGray)).
			Italic(true).
			MarginTop(1).
			MarginBottom(1),

		// Edit view
		EditLabel: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorGreen)).
			Bold(true).
			Width(15),

		EditHelp: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorGray)).
			Italic(true),

		EditSection: lipgloss.NewStyle().
			MarginTop(1).
			MarginBottom(1),

		// Error and warnings
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorRed)).
			Bold(true).
			MarginBottom(1),

		Warning: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorRed)).
			Bold(true).
			MarginBottom(1),

		Detail: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorWhite)).
			PaddingLeft(2),

		// Help view
		Key: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorGreen)).
			Bold(true).
			Width(20),

		Desc: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorWhite)),

		SectionHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorPurple)).
			MarginTop(1).
			MarginBottom(1),
	}
}

// GetItemTitleStyle returns the appropriate title style based on category and selection
func (s Styles) GetItemTitleStyle(category string, isSelected bool, hasChildren bool) lipgloss.Style {
	if isSelected {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorWhite)).
			Background(lipgloss.Color(ColorPurple)).
			Bold(true)
	}

	var style lipgloss.Style
	switch category {
	case "Proposed":
		style = s.ItemTitleProposed
	case "InProgress":
		style = s.ItemTitleInProgress
	case "Completed":
		style = s.ItemTitleCompleted
	case "Removed":
		style = s.ItemTitleRemoved
	default:
		style = s.ItemTitleProposed
	}

	// Make title bold if this is a parent (has children)
	if hasChildren {
		style = style.Bold(true)
	}

	return style
}

// GetStateStyle returns the appropriate state style based on category and selection
func (s Styles) GetStateStyle(category string, isSelected bool) lipgloss.Style {
	if isSelected {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorWhite)).
			Background(lipgloss.Color(ColorPurple)).
			Bold(true)
	}

	switch category {
	case "Proposed":
		return s.ProposedState
	case "InProgress":
		return s.InProgressState
	case "Completed":
		return s.CompletedState
	case "Removed":
		return s.RemovedState
	default:
		return s.ProposedState
	}
}

// RenderTitleBar renders the title bar with the given title text and width
func (s Styles) RenderTitleBar(title string, width int) string {
	titleBarStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(ColorPurple)).
		Foreground(lipgloss.Color(ColorWhite)).
		Bold(true).
		Width(width).
		Padding(0, 1)

	return titleBarStyle.Render(title) + "\n\n"
}
