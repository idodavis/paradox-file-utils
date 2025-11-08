# Synchronizer Enhancement Plan

## Overview

Transform the vanilla synchronizer from a CLI-only tool into a comprehensive GUI application with advanced merge capabilities, diff viewing, robust testing, and patch comparison features.

## Architecture Changes

### 1. Project Restructure

- **New structure:**
  - `cmd/vanilla-synchronizer/` - CLI entry point (preserved for backward compatibility)
  - `cmd/gui/` - Wails GUI application entry point
  - `internal/core/` - Core merger logic (refactored from current main)
  - `internal/diff/` - Diff generation and comparison utilities
  - `internal/config/` - Configuration management
  - `internal/gui/` - GUI-specific logic and state management
  - `parser/` - Keep existing parser package
  - `testdata/` - Test files for parser robustness testing

### 2. Core Refactoring

- Extract merger logic from `vanilla_synchronizer.go` into reusable `internal/core/merger.go`
- Create `internal/core/types.go` for shared data structures (Config, MergeOptions, etc.)
- Implement `internal/core/merger.go` with:
  - Generic merge function supporting any two file sets (A + B)
  - Precedence rules (A wins, B wins, custom list)
  - Entry-level precedence override support
  - Merge result tracking for diff generation

## Implementation Phases

### Phase 1: Core Refactoring & Diff Feature

**Files to create/modify:**

- `internal/core/merger.go` - Extracted merger logic with enhanced options
- `internal/core/types.go` - Shared types and structures
- `internal/diff/diff.go` - Diff generation using `github.com/sergi/go-diff`
- `internal/diff/renderer.go` - Diff rendering (unified, side-by-side formats)
- Modify `vanilla_synchronizer.go` to use new core package

**Features:**

- Refactor current merge logic into reusable functions
- Add merge result tracking (what changed, added, removed)
- Generate unified diff output
- Save diff files alongside merged output
- Support diff viewing in terminal (optional)

### Phase 2: Enhanced Merge Options

**Files to create/modify:**

- `internal/core/merger.go` - Add merge strategies
- `internal/core/types.go` - Add MergeOptions struct
- `internal/config/config.go` - Enhanced config with merge options

**Features:**

- Support merging any two file sets (not just vanilla + mod)
- Precedence modes: A-wins, B-wins, custom-entry-list
- Per-entry precedence override list
- File set specification (directory A, directory B with file matching)

### Phase 3: GUI Implementation (go-wails)

**Files to create:**

- `cmd/gui/main.go` - Wails app entry point
- `internal/gui/app.go` - Backend application logic
- `internal/gui/frontend/` - Frontend assets (HTML/CSS/JS)
- `wails.json` - Wails configuration
- `app.go` - Wails app struct and methods

**GUI Features:**

- File/directory browser for selecting A and B file sets
- Configuration panel for merge options
- Precedence rule selection (radio buttons/dropdown)
- Custom precedence entry list (text area or table)
- Run merge button with progress indicator
- Diff viewer panel (integrated or separate window)
- Results panel showing merge status and file list
- Save/load configuration profiles

**Dependencies to add:**

- `github.com/wailsapp/wails/v2`
- Frontend framework (Vue/React/Vanilla JS - user choice)

### Phase 4: Parser Testing & Robustness

**Files to create:**

- `parser/parser_test.go` - Comprehensive parser tests
- `testdata/events/` - Various event file samples
- `testdata/landed_titles/` - Landed titles samples
- `testdata/decisions/` - Decision files
- `testdata/scripted_effects/` - Scripted effects
- `testdata/edge_cases/` - Edge cases and malformed files

**Testing strategy:**

- Unit tests for parser with various file types
- Edge case testing (nested objects, complex expressions, etc.)
- Error handling tests
- Performance tests with large files
- Integration tests with real CK3 files

### Phase 5: Patch Diff Feature (Long-term)

**Files to create:**

- `internal/patch/patch.go` - Patch comparison logic
- `internal/patch/storage.go` - Version storage (optional)
- GUI panel for patch comparison

**Features:**

- Compare current game directory with previous patch version
- Detect all changed files
- Generate diffs for each changed file
- Display summary of changes
- Export patch diff report

## Technical Decisions

1. **Diff Library:** Use `github.com/sergi/go-diff/diffmatchpatch` for generating diffs
2. **GUI Framework:** go-wails v2 with Vue 3 (or user preference)
3. **Config Format:** Keep YAML for CLI, add JSON support for GUI (easier to serialize)
4. **Backward Compatibility:** Maintain CLI tool alongside GUI
5. **File Matching:** Use relative path matching between file sets A and B

## Dependencies to Add

```go
require (
    github.com/wailsapp/wails/v2 v2.x.x
    github.com/sergi/go-diff v1.x.x
    // ... existing dependencies
)
```

## Migration Path

1. Phase 1-2 can be done incrementally without breaking existing CLI
2. GUI (Phase 3) runs alongside CLI initially
3. Eventually GUI becomes primary interface, CLI remains for automation
4. Testing (Phase 4) runs continuously during development
5. Patch feature (Phase 5) added as enhancement after core features stable

## Key Files Reference

**Current:**

- `vanilla-synchronizer/vanilla_synchronizer.go` - Current implementation
- `parser/paradox_participle.go` - Parser implementation

**New Core:**

- `internal/core/merger.go` - Core merge logic
- `internal/core/types.go` - Shared types
- `internal/diff/diff.go` - Diff generation

**New GUI:**

- `cmd/gui/main.go` - GUI entry point
- `internal/gui/app.go` - Backend logic
- `internal/gui/frontend/` - Frontend code

**Testing:**

- `parser/parser_test.go` - Parser tests
- `testdata/` - Test files directory