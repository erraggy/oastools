package doctest

import (
	"go/ast"
	goparser "go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDocCodeExampleAPISync verifies that Go code examples in documentation
// reference symbols that actually exist in the oastools public packages.
//
// This catches:
//   - References to renamed or removed functions (e.g., WithDocument → WithParsed)
//   - References to nonexistent types or constants (e.g., builder.ResponseConfig)
//   - References to internal packages in user-facing examples (e.g., severity.SeverityCritical)
func TestDocCodeExampleAPISync(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller(0) failed")
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	// Public oastools packages to verify symbol references against.
	publicPkgNames := []string{
		"builder", "converter", "differ", "fixer", "generator",
		"httpvalidator", "joiner", "oaserrors", "overlay",
		"parser", "validator", "walker",
	}

	// Build symbol table: package name → set of exported symbol names.
	symbols := make(map[string]map[string]bool, len(publicPkgNames))
	for _, pkg := range publicPkgNames {
		dir := filepath.Join(repoRoot, pkg)
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		symbols[pkg] = extractExportedSymbols(t, dir)
	}

	// Internal package names that should not be referenced in doc code examples.
	// Value is the suggested public package to use instead (empty if no direct equivalent).
	internalPkgs := map[string]string{
		"severity":   "differ",
		"httputil":   "",
		"jsonpath":   "",
		"maputil":    "",
		"naming":     "",
		"pathutil":   "",
		"schemautil": "",
		"stringutil": "",
		"testutil":   "",
	}

	// Build regex for matching qualified references: knownPkg.ExportedSymbol.
	allPkgNames := make([]string, 0, len(publicPkgNames)+len(internalPkgs))
	allPkgNames = append(allPkgNames, publicPkgNames...)
	for pkg := range internalPkgs {
		allPkgNames = append(allPkgNames, pkg)
	}
	sort.Strings(allPkgNames)
	refRe := regexp.MustCompile(`\b(` + strings.Join(allPkgNames, "|") + `)\.([A-Z][a-zA-Z0-9]*)`)

	// Known exceptions: qualified references in doc code examples that are
	// intentional even though they don't match an exported symbol.
	// Key: relative file path → set of "pkg.Symbol" strings.
	exceptions := map[string]map[string]bool{
		// The httpvalidator deep_dive (and its generated docs/packages/ copy)
		// intentionally shows how it re-exports internal severity constants:
		// SeverityError = severity.SeverityError.
		"httpvalidator/deep_dive.md": {
			"severity.Severity":         true,
			"severity.SeverityError":    true,
			"severity.SeverityWarning":  true,
			"severity.SeverityInfo":     true,
			"severity.SeverityCritical": true,
		},
		"docs/packages/httpvalidator.md": {
			"severity.Severity":         true,
			"severity.SeverityError":    true,
			"severity.SeverityWarning":  true,
			"severity.SeverityInfo":     true,
			"severity.SeverityCritical": true,
		},
		// The whitepaper shows the ValidationError struct definition which
		// includes the internal severity.Severity type.
		"docs/whitepaper.md": {
			"severity.Severity": true,
		},
		// Petstore examples use a local variable named "validator" (generated
		// SecurityValidator type), not the oastools validator package.
		"examples/petstore/stdlib/README.md": {
			"validator.ConfigureScheme":   true,
			"validator.ValidateOperation": true,
		},
		"examples/petstore/chi/README.md": {
			"validator.ConfigureScheme":   true,
			"validator.ValidateOperation": true,
		},
		"docs/examples/petstore/stdlib.md": {
			"validator.ConfigureScheme":   true,
			"validator.ValidateOperation": true,
		},
		"docs/examples/petstore/chi.md": {
			"validator.ConfigureScheme":   true,
			"validator.ValidateOperation": true,
		},
	}

	// Find and scan all documentation markdown files.
	mdFiles := findDocMarkdownFiles(t, repoRoot)
	require.NotEmpty(t, mdFiles, "no markdown files found to scan")

	for _, mdFile := range mdFiles {
		relPath, _ := filepath.Rel(repoRoot, mdFile)
		t.Run(relPath, func(t *testing.T) {
			content, err := os.ReadFile(mdFile)
			require.NoError(t, err)

			blocks := extractGoCodeBlocks(string(content))
			if len(blocks) == 0 {
				return
			}

			fileExc := exceptions[relPath]
			foundRefs := make(map[string]bool) // for staleness checks

			for _, block := range blocks {
				lines := strings.Split(block.code, "\n")
				for lineIdx, line := range lines {
					for _, match := range refRe.FindAllStringSubmatch(line, -1) {
						pkg, sym := match[1], match[2]
						qualName := pkg + "." + sym
						mdLine := block.startLine + lineIdx
						foundRefs[qualName] = true

						if fileExc[qualName] {
							continue
						}

						// Flag internal package references.
						if alt, isInternal := internalPkgs[pkg]; isInternal {
							if alt != "" {
								t.Errorf("%s:%d: references internal package %s.%s (use %s.%s instead)",
									relPath, mdLine, pkg, sym, alt, sym)
							} else {
								t.Errorf("%s:%d: references internal package %s.%s",
									relPath, mdLine, pkg, sym)
							}
							continue
						}

						// Verify the symbol exists in the public package.
						pkgSymbols := symbols[pkg]
						if pkgSymbols == nil {
							continue
						}
						assert.True(t, pkgSymbols[sym],
							"%s:%d: references %s but no such exported symbol exists in the %s package",
							relPath, mdLine, qualName, pkg)
					}
				}
			}

			// Check for stale exceptions.
			for exc := range fileExc {
				assert.True(t, foundRefs[exc],
					"%s: exception %q is stale — reference no longer appears in code examples",
					relPath, exc)
			}
		})
	}
}

// goCodeBlock represents a Go code example extracted from a markdown file.
type goCodeBlock struct {
	code      string
	startLine int // 1-indexed line number in the markdown file
}

var (
	goFenceOpenRe    = regexp.MustCompile("^`{3,}go(?:\\s.*)?$")
	codeFenceCloseRe = regexp.MustCompile("^`{3,}\\s*$")
)

// extractGoCodeBlocks parses markdown content and returns all fenced Go code
// blocks with their starting line numbers.
func extractGoCodeBlocks(content string) []goCodeBlock {
	lines := strings.Split(content, "\n")
	var blocks []goCodeBlock
	var current []string
	startLine := 0
	inBlock := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case !inBlock && goFenceOpenRe.MatchString(trimmed):
			inBlock = true
			startLine = i + 2 // 1-indexed, next line is first code line
			current = current[:0]
		case inBlock && codeFenceCloseRe.MatchString(trimmed):
			inBlock = false
			blocks = append(blocks, goCodeBlock{
				code:      strings.Join(current, "\n"),
				startLine: startLine,
			})
		case inBlock:
			current = append(current, line)
		}
	}
	return blocks
}

// extractExportedSymbols uses go/ast to find all exported names (functions,
// methods, types, constants, variables) in the given package directory,
// excluding test files. Methods are included because doc comments and code
// examples use the godoc-style package.Method syntax (e.g., validator.Validate).
func extractExportedSymbols(t *testing.T, dir string) map[string]bool {
	t.Helper()

	fset := token.NewFileSet()
	pkgs, err := goparser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	require.NoError(t, err, "parsing package dir %s", dir)

	syms := make(map[string]bool)
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				switch d := decl.(type) {
				case *ast.FuncDecl:
					if d.Name.IsExported() {
						syms[d.Name.Name] = true
					}
				case *ast.GenDecl:
					for _, spec := range d.Specs {
						switch s := spec.(type) {
						case *ast.TypeSpec:
							if s.Name.IsExported() {
								syms[s.Name.Name] = true
							}
						case *ast.ValueSpec:
							for _, name := range s.Names {
								if name.IsExported() {
									syms[name.Name] = true
								}
							}
						}
					}
				}
			}
		}
	}
	return syms
}

// findDocMarkdownFiles returns all user-facing documentation markdown files.
// It scans: README.md, docs/ (excluding plans/), package deep_dive.md files,
// and examples/ READMEs. Generated files (docs/examples/, docs/packages/) are
// included if present locally but may not exist in CI (they're gitignored).
func findDocMarkdownFiles(t *testing.T, repoRoot string) []string {
	t.Helper()

	var files []string

	// Root README.
	readme := filepath.Join(repoRoot, "README.md")
	if _, err := os.Stat(readme); err == nil {
		files = append(files, readme)
	}

	// Non-API docs to skip (contributor guides, licenses — not user-facing code examples).
	skipFiles := map[string]bool{
		"CONTRIBUTORS.md": true,
		"LICENSE.md":      true,
	}

	// docs/ directory (skip plans/ — design docs, not user-facing).
	walkMarkdownDir(filepath.Join(repoRoot, "docs"), &files, func(name string) bool {
		return name == "plans"
	}, skipFiles)

	// Package deep_dive.md files.
	entries, err := os.ReadDir(repoRoot)
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			ddPath := filepath.Join(repoRoot, e.Name(), "deep_dive.md")
			if _, err := os.Stat(ddPath); err == nil {
				files = append(files, ddPath)
			}
		}
	}

	// examples/ directory.
	walkMarkdownDir(filepath.Join(repoRoot, "examples"), &files, nil, nil)

	sort.Strings(files)
	return files
}

// walkMarkdownDir recursively walks a directory, appending .md files to out.
// skipDir is called with each directory's base name; returning true skips it.
// skipFile, if non-nil, skips files whose base name is in the set.
func walkMarkdownDir(root string, out *[]string, skipDir func(string) bool, skipFile map[string]bool) {
	if _, err := os.Stat(root); err != nil {
		return
	}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && skipDir != nil && skipDir(d.Name()) {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".md") && !skipFile[d.Name()] {
			*out = append(*out, path)
		}
		return nil
	})
}
