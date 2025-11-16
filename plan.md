# Hippo Refactoring Plan

Based on analysis of the codebase structure and AGENTS.md principles, here are the top refactoring opportunities prioritized by impact.

## Completed

### ✅ #1. Split `main.go` into focused modules (HIGH PRIORITY)

**Problem:** `main.go` (319 lines) contained 5 different responsibilities:
- Type definitions (viewState, appMode, Sprint, model struct)
- Initialization logic (initialModel, Init)
- Update routing logic (Update function)
- Utility functions (min, max)
- Entry point (main)

**Solution Implemented:**
- ✅ `types.go` - All type definitions (viewState, appMode, tabs, Sprint, model)
- ✅ `model_init.go` - Model initialization and Init() method
- ✅ `update.go` - Update method and message routing
- ✅ `utils.go` - Helper functions (min, max)
- ✅ `main.go` - Only main() entry point (21 lines)

**Result:** Reduced main.go from 320 lines to 21 lines, following single responsibility principle.

---

## Pending Refactorings

### 🔥 #2. Break down large view rendering functions (HIGH PRIORITY)

**Problem:** `views.go` (1134 lines) has several massive rendering functions:
- `renderListView()` (236 lines: 172-408)
- `renderCreateView()` (119 lines: 738-998)
- `renderHelpView()` (76 lines: 597-674)

**Recommendation:** Extract into smaller, focused files:
- `view_list.go` - List view rendering with helper functions for tabs, modes, items
- `view_create.go` - Create view rendering
- `view_help.go` - Help view rendering
- `view_detail.go` - Detail view rendering
- `view_modals.go` - State picker, filter, find, error, delete confirm views
- Keep only the main `View()` dispatcher in `views.go`

**Benefits:**
- Easier to find and modify specific view logic
- Smaller files are easier to test and reason about
- Reduces cognitive load when working on UI

---

### 🔥 #3. Consolidate message handlers (MEDIUM PRIORITY)

**Problem:** `handlers.go` (1189 lines) is the largest file and mixes concerns:
- Key input handlers (lines 10-360)
- Message handlers (lines 776-1190)
- Both view-specific and global handlers

**Recommendation:** Split into:
- `handlers_keys.go` - All keyboard input handlers
- `handlers_messages.go` - All tea.Msg handlers
- `handlers_global.go` - Global hotkeys

**Benefits:**
- Clear separation between keyboard input and message handling
- Easier to find specific handler logic
- Better organization for future enhancements

---

### 🟡 #4. Extract tree rendering logic (MEDIUM PRIORITY)

**Problem:** Tree prefix logic is scattered across multiple view renderers with duplicated styling code (e.g., in `renderListView`, `renderFilterView`, `renderCreateView`).

**Recommendation:**
- Create `tree_renderer.go` with functions like:
  - `renderTreeItem(item TreeItem, isSelected, isBatchSelected bool, cursor string) string`
  - `renderLoadMoreItem(cursor int, isSelected, isLoading bool) string`
  - Consolidate all tree-related rendering logic

**Benefits:**
- DRY (Don't Repeat Yourself) - single source of truth for tree rendering
- Consistent styling across all views
- Easier to modify tree rendering behavior

---

### 🟡 #5. Simplify model state management (MEDIUM PRIORITY)

**Problem:** The `model` struct (lines 62-128 in types.go) has 40+ fields mixing different concerns:
- List management (sprintLists, backlogLists)
- UI state (cursor, scrollOffset, width, height)
- Temporary state (filteredTasks, filterActive)
- Edit mode fields
- Create mode fields
- Delete mode fields

**Recommendation:** Group related fields into sub-structs:
```go
type model struct {
    // Core data
    lists    *ListManager
    client   *AzureDevOpsClient
    
    // UI state
    ui       UIState
    
    // Mode-specific state
    edit     EditState
    create   CreateState
    delete   DeleteState
    
    // Styles
    styles   Styles
}
```

**Benefits:**
- Clearer organization of state
- Easier to pass relevant state to functions
- Better encapsulation of mode-specific data

---

## Quick Wins (Low Effort, High Value)

1. **Extract constants** - Lines from types.go should be in `constants.go`
2. **Move helper functions** - `openInBrowser()` (lines 1117-1134 in views.go) should be in `browser.go` or extend `utils.go`
3. **Consolidate style definitions** - Already have `styles.go`, but style creation is scattered in view files

---

## Recommended Order

1. ✅ **#1** (split main.go) - Completed ✓
2. **#2** (views refactor) - Biggest file size reduction, improves maintainability
3. **#5** (model refactor) - Makes everything else clearer
4. **#3** (handlers) - Completes the core refactoring
5. **#4** (tree rendering) - Polish and DRY

---

## Principles to Follow

From AGENTS.md:
- **Keep files simple with clear single responsibility**
- Each file should have ONE clear purpose
- Split large files into smaller, focused modules
- Extract related functionality into separate files
- Name files clearly to reflect their single responsibility
- Avoid creating monolithic files that do multiple things
