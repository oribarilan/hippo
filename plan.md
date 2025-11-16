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

### ✅ #2. Break down large view rendering functions (HIGH PRIORITY)

**Problem:** `views.go` (1,134 lines) had several massive rendering functions:
- `renderListView()` (236 lines: 172-408)
- `renderCreateView()` (119 lines: 738-998)
- `renderHelpView()` (76 lines: 597-674)
- Plus modal views (state picker, filter, find, error, delete confirm)

**Solution Implemented:**
- ✅ `view_list.go` (~270 lines) - List view with tabs, modes, and tree rendering
- ✅ `view_create.go` (~280 lines) - Create view with inline item insertion
- ✅ `view_detail.go` (~95 lines) - Detail view and edit view rendering
- ✅ `view_help.go` (~90 lines) - Keybindings help screen
- ✅ `view_modals.go` (~300 lines) - All modal views (state picker, filter, find, error, delete confirm)
- ✅ `views.go` (~200 lines) - View() dispatcher and shared helpers (title bar, footer, loading screen, openInBrowser)

**Result:** Reduced views.go from 1,134 lines to ~200 lines. Split into 6 focused files, each with a clear single responsibility.

---

### ✅ #5. Simplify model state management (MEDIUM PRIORITY)

**Problem:** The `model` struct had 40+ fields mixing different concerns:
- List management (sprintLists, backlogLists)
- UI state (cursor, scrollOffset, width, height)
- Temporary state (filteredTasks, filterActive)
- Edit mode fields
- Create mode fields
- Delete mode fields

**Solution Implemented:**
- ✅ Created `UIState` struct - cursor, scrollOffset, width, height, viewportReady
- ✅ Created `EditState` struct - titleInput, descriptionInput, fieldCursor, fieldCount
- ✅ Created `CreateState` struct - input, insertPos, after, parentID, depth, isLast, createdItemID
- ✅ Created `DeleteState` struct - itemID, itemTitle
- ✅ Created `BatchState` struct - selectedItems, operationCount
- ✅ Created `FilterState` struct - filteredTasks, active, filterInput, findInput
- ✅ Updated all 10 Go files with ~280+ field references to use new sub-struct organization

**Result:** Model struct now has clear separation of concerns with grouped state. Much easier to understand what state belongs where and pass relevant state to functions.

---

## Pending Refactorings

---

### 🔥 #3. Consolidate message handlers (MEDIUM PRIORITY) - NEXT UP

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

### 🟡 #4. Extract tree rendering logic (LOW PRIORITY)

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

**Note:** This is a polish refactor - tree rendering currently works well, so this is lower priority.

---

## Quick Wins (Low Effort, High Value)

1. **Extract constants** - Move constants from types.go to `constants.go` (version, defaultLoadLimit, etc.)
2. **Move helper functions** - Move `openInBrowser()` from views.go to `utils.go`
3. **Consolidate style definitions** - Move inline style creation from view files into `styles.go`

---

## Progress Summary

### Completed (3/5 major refactorings):
- ✅ #1: Split main.go into focused modules
- ✅ #2: Break down large view rendering functions
- ✅ #5: Simplify model state management

### Next Up:
- 🔥 #3: Consolidate message handlers (handlers.go split)
- 🟡 #4: Extract tree rendering logic (optional polish)

---

## Recommended Order

1. ✅ **#1** (split main.go) - Completed ✓
2. ✅ **#2** (views refactor) - Completed ✓ - Biggest file size reduction, improves maintainability
3. ✅ **#5** (model refactor) - Completed ✓ - Makes everything else clearer
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
