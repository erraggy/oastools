package commands

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCLIFlagsDocumented verifies that every registered CLI flag appears in
// docs/cli-reference.md, and every documented flag corresponds to a registered flag.
func TestCLIFlagsDocumented(t *testing.T) {
	// Resolve docs path from this test file's location.
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller(0) failed to retrieve file path")
	docPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "docs", "cli-reference.md")

	docBytes, err := os.ReadFile(docPath)
	require.NoError(t, err, "reading cli-reference.md")
	docContent := string(docBytes)

	// Build the map of command -> FlagSet by calling every Setup*Flags function.
	commands := map[string]*flag.FlagSet{
		"validate":         mustFS(SetupValidateFlags()),
		"parse":            mustFS(SetupParseFlags()),
		"fix":              mustFS(SetupFixFlags()),
		"convert":          mustFS(SetupConvertFlags()),
		"diff":             mustFS(SetupDiffFlags()),
		"join":             mustFS(SetupJoinFlags()),
		"generate":         mustFS(SetupGenerateFlags()),
		"overlay apply":    mustFS(SetupOverlayApplyFlags()),
		"overlay validate": mustFS(SetupOverlayValidateFlags()),
		"walk operations":  mustFS(SetupWalkOperationsFlags()),
		"walk schemas":     mustFS(SetupWalkSchemasFlags()),
		"walk parameters":  mustFS(SetupWalkParametersFlags()),
		"walk responses":   mustFS(SetupWalkResponsesFlags()),
		"walk security":    mustFS(SetupWalkSecurityFlags()),
		"walk paths":       mustFS(SetupWalkPathsFlags()),
	}

	// Parse documented flags from markdown tables per command section.
	knownCmds := make(map[string]bool, len(commands))
	for name := range commands {
		knownCmds[name] = true
	}
	docFlags := parseDocFlags(docContent, knownCmds)

	for cmdName, fs := range commands {
		t.Run(cmdName, func(t *testing.T) {
			documented := docFlags[cmdName]
			require.NotNil(t, documented, "no flag table found in cli-reference.md for command %q", cmdName)

			// Collect registered long-form flag names (skip single-char aliases and help).
			registeredSet := make(map[string]bool)
			fs.VisitAll(func(f *flag.Flag) {
				if len(f.Name) == 1 || f.Name == "help" {
					return
				}
				registeredSet[f.Name] = true
			})

			// Check: every registered flag should be documented.
			for name := range registeredSet {
				assert.True(t, documented[name], "flag --%s is registered for %q but not documented in cli-reference.md", name, cmdName)
			}

			// Check: every documented flag should be registered.
			for name := range documented {
				if name == "help" {
					continue
				}
				assert.True(t, registeredSet[name], "flag --%s is documented for %q in cli-reference.md but not registered", name, cmdName)
			}
		})
	}
}

// mustFS extracts the *flag.FlagSet from a Setup*Flags return pair.
func mustFS[T any](fs *flag.FlagSet, _ T) *flag.FlagSet {
	return fs
}

// parseDocFlags parses cli-reference.md and returns a map of command name -> set of documented flag names.
// It looks for markdown sections (## command / ### subcommand) and flag table rows (| `--flagname` |).
// knownCmds is the authoritative set of command names to match against section headers.
func parseDocFlags(content string, knownCmds map[string]bool) map[string]map[string]bool {
	result := make(map[string]map[string]bool)

	// Split into lines for section tracking.
	lines := strings.Split(content, "\n")

	// allFlagsRe matches all --flag-name occurrences in a table row.
	// This handles rows like:
	// | `--flag-name` | description |
	// | `-s, --flag-name` | description |
	// | `--prune-all, --prune` | description |
	// | `-o, --flag-name string` | description |
	allFlagsRe := regexp.MustCompile(`--([a-zA-Z][a-zA-Z0-9-]*)`)

	var currentCmd string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track current section from ## and ### headers.
		// - ## headers are top-level commands: set currentCmd if recognized, reset otherwise
		// - ### headers are subcommands (overlay apply, walk operations, etc.): set if recognized, keep otherwise
		// - #### headers are sub-subsections (Flags, Examples): always ignore
		if strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### ") {
			var headerText string
			switch {
			case strings.HasPrefix(trimmed, "### "):
				headerText = strings.TrimPrefix(trimmed, "### ")
			case strings.HasPrefix(trimmed, "## "):
				headerText = strings.TrimPrefix(trimmed, "## ")
			}
			headerText = strings.TrimSpace(headerText)
			headerLower := strings.ToLower(headerText)

			if knownCmds[headerLower] {
				currentCmd = headerLower
				if result[currentCmd] == nil {
					result[currentCmd] = make(map[string]bool)
				}
			} else if strings.HasPrefix(trimmed, "## ") {
				// Reset currentCmd on unrecognized ## headers to avoid
				// attributing flags from unrelated sections to the previous command.
				currentCmd = ""
			}
			// For unrecognized ### headers (e.g., "### Flags", "### Examples"),
			// keep currentCmd as-is — they are subsections of the current command.
		}

		// Extract flags from the first cell of table rows.
		if currentCmd != "" && strings.HasPrefix(trimmed, "|") {
			// Extract the first table cell (between the first and second |).
			parts := strings.SplitN(trimmed, "|", 3)
			if len(parts) >= 3 {
				firstCell := parts[1]
				for _, matches := range allFlagsRe.FindAllStringSubmatch(firstCell, -1) {
					if len(matches) > 1 {
						result[currentCmd][matches[1]] = true
					}
				}
			}
		}
	}

	return result
}
