package conformance

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Source is one vendored version: where its fixtures came from and how many of
// each kind sources.txt says should be present.
type Source struct {
	// Version is the OAS version the fixtures target, as "3.0" or "3.2".
	Version string
	// Ref is the upstream branch the fixtures were taken from. Recorded so a
	// refresh knows what to resolve; the commit is what it actually fetches.
	Ref string
	// Commit is the exact upstream revision vendored, which is what makes the
	// suite reproducible.
	Commit string
	// Subpath is the directory holding pass/ and fail/ upstream. It differs
	// between the version branches and main, which keeps 3.0 under _archive_.
	Subpath string
	// Pass and Fail are the fixture counts recorded at vendor time. A tree that
	// disagrees has lost or gained files since.
	Pass int
	Fail int
}

// Dir returns the absolute path to the vendored suite, resolved from this
// file's own location so a test's working directory does not matter.
func Dir() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "testdata", "conformance")
}

// FixtureDir returns the directory holding one version's fixtures of one kind,
// where kind is "pass" or "fail".
func FixtureDir(version, kind string) string {
	return filepath.Join(Dir(), version, kind)
}

// LoadSources parses testdata/conformance/sources.txt.
//
// The format is deliberately plain: comment lines, then one whitespace-separated
// record per version. The refresh script is shell and has to read the same file,
// and macOS ships bash 3.2, so anything needing a JSON parser would have meant
// depending on jq being installed.
func LoadSources() ([]Source, error) {
	data, err := os.ReadFile(filepath.Join(Dir(), "sources.txt"))
	if err != nil {
		return nil, err
	}

	var sources []Source
	for raw := range strings.Lines(string(data)) {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 6 {
			return nil, &parseError{line: line, reason: "want 6 fields: version ref commit subpath pass fail"}
		}
		pass, err := strconv.Atoi(fields[4])
		if err != nil {
			return nil, &parseError{line: line, reason: "pass count is not a number"}
		}
		fail, err := strconv.Atoi(fields[5])
		if err != nil {
			return nil, &parseError{line: line, reason: "fail count is not a number"}
		}
		sources = append(sources, Source{
			Version: fields[0],
			Ref:     fields[1],
			Commit:  fields[2],
			Subpath: fields[3],
			Pass:    pass,
			Fail:    fail,
		})
	}
	return sources, nil
}

// Fixtures lists the fixture file names for one version and kind, sorted, so a
// caller iterating them reports in a stable order.
func Fixtures(version, kind string) ([]string, error) {
	entries, err := os.ReadDir(FixtureDir(version, kind))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

type parseError struct {
	line   string
	reason string
}

func (e *parseError) Error() string {
	return "conformance: cannot parse sources.txt line " + strconv.Quote(e.line) + ": " + e.reason
}
