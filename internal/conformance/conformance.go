package conformance

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
)

// Kind distinguishes the two halves of the suite. A bare string would sit next
// to the version at every call site, where the two are indistinguishable to the
// compiler and a swap reports no fixtures rather than an error.
type Kind string

const (
	// KindPass holds documents the published JSON Schema accepts.
	KindPass Kind = "pass"
	// KindFail holds documents it rejects.
	KindFail Kind = "fail"
)

// Kinds is every kind, for callers walking the whole suite.
var Kinds = []Kind{KindPass, KindFail}

// Source is one vendored version: where its fixtures came from, how many of
// each kind sources.txt records, and a digest over their names and contents.
type Source struct {
	// Version is the OAS version the fixtures target, as "3.0" or "3.2".
	Version string
	// Ref is the upstream branch the fixtures were taken from. Recorded so a
	// refresh knows what to resolve; the commit is what it actually fetches.
	Ref string
	// Commit is the exact upstream revision vendored, always unabbreviated: an
	// abbreviation can become ambiguous as the upstream repository grows.
	Commit string
	// Subpath is the directory holding pass/ and fail/ upstream. It differs
	// between the version branches and main, which keeps 3.0 under _archive_.
	Subpath string
	// pass and fail are the fixture counts recorded at vendor time. Unexported
	// so the count and its kind cannot drift apart at a call site; read them
	// through Count.
	pass int
	fail int
	// Digest fingerprints every fixture's name and content for this version.
	// Counts alone cannot see a fixture edited in place, nor one version's
	// fixtures vendored into another's directory: 3.2 and 3.3 publish the same
	// 37 pass fixture names and all 37 differ in content.
	Digest string
}

// Count returns the number of fixtures of one kind that sources.txt records.
func (s Source) Count(kind Kind) int {
	if kind == KindFail {
		return s.fail
	}
	return s.pass
}

// FixtureDir returns the directory holding this version's fixtures of one kind.
func (s Source) FixtureDir(kind Kind) string {
	return filepath.Join(Dir(), s.Version, string(kind))
}

// Fixtures lists this version's fixture file names of one kind, sorted.
//
// A version that publishes none of a kind has no directory, which yields no
// names and no error: 3.0 has no fail fixtures. Callers must therefore treat a
// count of zero as meaningful rather than as an empty success, which is what
// Count is for.
func (s Source) Fixtures(kind Kind) ([]string, error) {
	entries, err := os.ReadDir(s.FixtureDir(kind))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Type().IsRegular() && strings.HasSuffix(entry.Name(), ".yaml") {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

// ComputeDigest fingerprints this version's vendored fixtures, by the same
// construction scripts/conformance-vendor.sh uses: one line per fixture holding
// its kind, name and content hash, sorted bytewise, hashed.
func (s Source) ComputeDigest() (string, error) {
	var lines []string
	for _, kind := range Kinds {
		names, err := s.Fixtures(kind)
		if err != nil {
			return "", err
		}
		for _, name := range names {
			content, err := os.ReadFile(filepath.Join(s.FixtureDir(kind), name))
			if err != nil {
				return "", err
			}
			sum := sha256.Sum256(content)
			lines = append(lines, string(kind)+"/"+name+" "+hex.EncodeToString(sum[:]))
		}
	}
	slices.Sort(lines)

	h := sha256.New()
	for _, line := range lines {
		h.Write([]byte(line + "\n"))
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Dir returns the absolute path to the vendored suite, resolved from this
// file's own location so a caller's working directory does not matter.
//
// It panics rather than returning a path it could not resolve. runtime.Caller
// does not fail for depth 0 in any ordinary build, and an empty root would be
// joined away silently, leaving every lookup relative and every version looking
// like a version that publishes nothing.
func Dir() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("conformance: cannot resolve this file's location")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "testdata", "conformance")
}

// LoadSources parses testdata/conformance/sources.txt.
func LoadSources() ([]Source, error) {
	data, err := os.ReadFile(filepath.Join(Dir(), "sources.txt"))
	if err != nil {
		return nil, err
	}
	return parseSources(data)
}

// parseSources is LoadSources without the file, so the error paths below are
// reachable from a test.
//
// The format is deliberately plain: comment lines, then one whitespace-separated
// record per version. The refresh script is shell and has to read the same file,
// and macOS ships bash 3.2, so anything needing a JSON parser would have meant
// depending on jq being installed.
func parseSources(data []byte) ([]Source, error) {
	var sources []Source
	for raw := range strings.Lines(string(data)) {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 7 {
			return nil, &parseError{line: line, reason: "want 7 fields: version ref commit subpath pass fail digest"}
		}
		pass, err := parseCount(fields[4])
		if err != nil {
			return nil, &parseError{line: line, reason: "pass count: " + err.Error()}
		}
		fail, err := parseCount(fields[5])
		if err != nil {
			return nil, &parseError{line: line, reason: "fail count: " + err.Error()}
		}
		if !isHex(fields[2], 40) {
			return nil, &parseError{line: line, reason: "commit is not a full 40-character revision"}
		}
		if !isHex(fields[6], 64) {
			return nil, &parseError{line: line, reason: "digest is not a 64-character sha256"}
		}
		sources = append(sources, Source{
			Version: fields[0],
			Ref:     fields[1],
			Commit:  fields[2],
			Subpath: fields[3],
			pass:    pass,
			fail:    fail,
			Digest:  fields[6],
		})
	}
	if len(sources) == 0 {
		return nil, &parseError{line: "", reason: "no records"}
	}
	return sources, nil
}

func parseCount(field string) (int, error) {
	n, err := strconv.Atoi(field)
	if err != nil {
		return 0, errNotANumber
	}
	if n < 0 {
		return 0, errNegative
	}
	return n, nil
}

func isHex(s string, want int) bool {
	if len(s) != want {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

type constError string

func (e constError) Error() string { return string(e) }

const (
	errNotANumber constError = "not a number"
	errNegative   constError = "negative"
)

type parseError struct {
	line   string
	reason string
}

func (e *parseError) Error() string {
	if e.line == "" {
		return "conformance: cannot parse sources.txt: " + e.reason
	}
	return "conformance: cannot parse sources.txt line " + strconv.Quote(e.line) + ": " + e.reason
}
