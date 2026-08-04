package conformance

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fullSHA matches an unabbreviated commit. A pin has to be exact: an
// abbreviation can become ambiguous as the upstream repository grows, and a
// branch name would not be a pin at all.
var fullSHA = regexp.MustCompile(`^[0-9a-f]{40}$`)

func TestSourcesParse(t *testing.T) {
	sources, err := LoadSources()
	require.NoError(t, err)

	versions := make([]string, 0, len(sources))
	for _, s := range sources {
		versions = append(versions, s.Version)
	}
	assert.Equal(t, []string{"3.0", "3.1", "3.2", "3.3"}, versions,
		"the vendored versions are 3.0 through 3.3; 2.0 publishes no fixtures upstream and none are authored here")

	for _, s := range sources {
		t.Run(s.Version, func(t *testing.T) {
			assert.Regexp(t, fullSHA, s.Commit, "the pin must be a full commit")
			assert.NotEmpty(t, s.Ref)
			assert.NotEmpty(t, s.Subpath)
			assert.Positive(t, s.Pass, "every vendored version has pass fixtures")
			assert.GreaterOrEqual(t, s.Fail, 0)
		})
	}
}

// TestVendoredTreeMatchesSources is the drift guard. The counts in sources.txt
// were what upstream published at the pinned commit, so a tree that disagrees
// has lost or gained files since it was vendored, which no other test would
// notice: a deleted fixture makes a suite smaller, not redder.
func TestVendoredTreeMatchesSources(t *testing.T) {
	sources, err := LoadSources()
	require.NoError(t, err)

	for _, s := range sources {
		t.Run(s.Version, func(t *testing.T) {
			for kind, want := range map[string]int{"pass": s.Pass, "fail": s.Fail} {
				got, err := Fixtures(s.Version, kind)
				require.NoError(t, err)
				assert.Len(t, got, want,
					"%s/%s: sources.txt records %d fixtures; run 'make conformance-vendor' to restore the tree",
					s.Version, kind, want)
			}
		})
	}
}

// TestVendoredTreeHoldsOnlyFixtures keeps the vendor to documents. Upstream
// keeps a JavaScript harness beside its fixtures, and 3.3 keeps
// minimal-objects.yaml, a table of object stubs rather than an OpenAPI
// document, so a walk that took everything in the directory would feed all
// three to a validator.
func TestVendoredTreeHoldsOnlyFixtures(t *testing.T) {
	sources, err := LoadSources()
	require.NoError(t, err)

	for _, s := range sources {
		for _, kind := range []string{"pass", "fail"} {
			dir := FixtureDir(s.Version, kind)
			entries, err := os.ReadDir(dir)
			if os.IsNotExist(err) {
				continue
			}
			require.NoError(t, err)

			for _, entry := range entries {
				name := filepath.Join(s.Version, kind, entry.Name())
				assert.False(t, entry.IsDir(), "%s: the suite is flat, so a directory here is unexpected", name)
				assert.True(t, strings.HasSuffix(entry.Name(), ".yaml"),
					"%s: only .yaml documents are vendored", name)

				info, err := entry.Info()
				require.NoError(t, err)
				assert.Positive(t, info.Size(), "%s: an empty fixture asserts nothing", name)
			}
		}
	}
}

// TestKnownFixturesArePresent names one fixture per version that the project
// already reasons about, so a refresh that silently dropped the interesting
// cases is caught by more than a count.
func TestKnownFixturesArePresent(t *testing.T) {
	for _, tc := range []struct {
		version string
		kind    string
		name    string
		why     string
	}{
		{"3.0", "pass", "petstore.yaml", "the archived 3.0 documents are the only fixtures that version publishes"},
		{"3.1", "fail", "parameter-object-header-allowReserved.yaml", "the documented v1 divergence, which a bool cannot express"},
		{"3.2", "pass", "operation-object-example.yaml", "schema-valid and specification-invalid, so it stays rejected"},
		{"3.2", "fail", "server_enum_empty.yaml", "a server variable enum that must not be empty"},
		{"3.3", "fail", "server_enum_empty.yaml", "3.3 restates the constraints 3.2 states, so the suite carries them forward"},
	} {
		t.Run(tc.version+"/"+tc.kind+"/"+tc.name, func(t *testing.T) {
			names, err := Fixtures(tc.version, tc.kind)
			require.NoError(t, err)
			assert.Contains(t, names, tc.name, tc.why)
		})
	}
}

// TestVersionsAreNotCopiesOfEachOther pins fixtures that change sides between
// versions, which no count can detect.
//
// 3.2 widened where `allowReserved` may appear, so two documents that are
// negative fixtures at 3.1 are positive ones at 3.2 and 3.3. A refresh that
// vendored one branch into all three version directories would produce the
// recorded counts and still be wrong; this is what would catch it.
func TestVersionsAreNotCopiesOfEachOther(t *testing.T) {
	for _, name := range []string{
		"parameter-object-path-allowReserved.yaml",
		"parameter-object-cookie-form-allowReserved.yaml",
	} {
		t.Run(name, func(t *testing.T) {
			for _, tc := range []struct {
				version string
				kind    string
			}{
				{"3.1", "fail"},
				{"3.2", "pass"},
				{"3.3", "pass"},
			} {
				names, err := Fixtures(tc.version, tc.kind)
				require.NoError(t, err)
				assert.Contains(t, names, name,
					"%s should be a %s fixture at %s, since 3.2 widened where allowReserved may appear",
					name, tc.kind, tc.version)
			}
		})
	}
}
