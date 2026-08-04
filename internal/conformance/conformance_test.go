package conformance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadSources fails the test rather than returning, and asserts the suite is
// non-empty: every guard below iterates the records, so an empty slice would
// turn them all into assertions that never run.
func loadSources(t *testing.T) []Source {
	t.Helper()
	sources, err := LoadSources()
	require.NoError(t, err)
	require.Len(t, sources, 4, "3.0 through 3.3 are vendored")
	return sources
}

func TestSourcesParse(t *testing.T) {
	sources := loadSources(t)

	versions := make([]string, 0, len(sources))
	for _, s := range sources {
		versions = append(versions, s.Version)
	}
	assert.Equal(t, []string{"3.0", "3.1", "3.2", "3.3"}, versions,
		"2.0 publishes no fixtures upstream and none are authored here")
}

// TestParseSourcesRejects covers the malformed records parseSources refuses.
// sources.txt is rewritten by a shell script doing string concatenation, so a
// malformed rewrite is a plausible failure and this message is the only thing a
// maintainer would see.
func TestParseSourcesRejects(t *testing.T) {
	const (
		commit = "5423da4b0c16a563a7226663018fcd9294be279d"
		digest = "d1fe82bdd6909d31a6c5f501559ab5a88b9dc9b590adf63fe4054e55bc028fa8"
	)
	good := "3.0 main " + commit + " _archive_/schemas/v3.0 6 0 " + digest

	for _, tc := range []struct {
		name  string
		input string
		want  string
	}{
		{"too few fields", "3.0 main " + commit + " sub 6 0", "want 7 fields"},
		{"too many fields", good + " extra", "want 7 fields"},
		{"pass is not a number", "3.0 main " + commit + " sub x 0 " + digest, "pass count: not a number"},
		{"fail is not a number", "3.0 main " + commit + " sub 6 3.5 " + digest, "fail count: not a number"},
		{"pass is negative", "3.0 main " + commit + " sub -1 0 " + digest, "pass count: negative"},
		{"commit is abbreviated", "3.0 main 5423da4 sub 6 0 " + digest, "not a full 40-character revision"},
		{"commit is not hex", "3.0 main " + commit[:39] + "z sub 6 0 " + digest, "not a full 40-character revision"},
		{"digest is truncated", "3.0 main " + commit + " sub 6 0 " + digest[:63], "not a 64-character sha256"},
		{"no records at all", "# only a comment\n\n", "no records"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseSources([]byte(tc.input + "\n"))
			assert.Nil(t, got)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
			assert.Contains(t, err.Error(), "conformance: cannot parse sources.txt")
		})
	}

	t.Run("the good record round-trips", func(t *testing.T) {
		got, err := parseSources([]byte("# header\n\n" + good + "\n"))
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "3.0", got[0].Version)
		assert.Equal(t, 6, got[0].Count(KindPass))
		assert.Equal(t, 0, got[0].Count(KindFail))
		assert.Equal(t, digest, got[0].Digest)
	})
}

// TestVendoredTreeMatchesSources is the count half of the drift guard. A
// deleted fixture makes a suite smaller rather than redder, so nothing else
// would notice it.
func TestVendoredTreeMatchesSources(t *testing.T) {
	for _, s := range loadSources(t) {
		t.Run(s.Version, func(t *testing.T) {
			for _, kind := range Kinds {
				got, err := s.Fixtures(kind)
				require.NoError(t, err)

				if s.Count(kind) == 0 {
					// Absence is asserted rather than inferred. Fixtures reports
					// no names for a directory that is missing and for one that
					// is empty, so without this the subtest would pass either way.
					assert.Empty(t, got)
					assert.NoDirExists(t, s.FixtureDir(kind),
						"%s/%s: sources.txt records none, so upstream publishes no such directory",
						s.Version, kind)
					continue
				}
				assert.Len(t, got, s.Count(kind),
					"%s/%s: sources.txt records %d fixtures; run 'make conformance-vendor' to restore the tree",
					s.Version, kind, s.Count(kind))
			}
		})
	}
}

// TestVendoredTreeMatchesDigests is the content half, and it is what makes the
// guard exact. Names and counts cannot distinguish 3.2 from 3.3: both publish
// the same 37 pass fixture names, and all 37 differ in content, so vendoring
// one into the other's directory changes nothing a count can see. It also
// catches a fixture edited in place, which is how a failing fixture would stop
// failing.
func TestVendoredTreeMatchesDigests(t *testing.T) {
	seen := make(map[string]string, 4)
	for _, s := range loadSources(t) {
		t.Run(s.Version, func(t *testing.T) {
			got, err := s.ComputeDigest()
			require.NoError(t, err)
			assert.Equal(t, s.Digest, got,
				"%s: the vendored fixtures are not the ones sources.txt records", s.Version)
		})
		if other, dup := seen[s.Digest]; dup {
			t.Errorf("%s and %s have the same digest, so one was vendored into the other", other, s.Version)
		}
		seen[s.Digest] = s.Version
	}
}

// TestVendoredTreeHoldsOnlyFixtures keeps the tree to documents, so a stray
// file cannot be counted as a fixture or fed to a validator by #436's harness.
func TestVendoredTreeHoldsOnlyFixtures(t *testing.T) {
	for _, s := range loadSources(t) {
		for _, kind := range Kinds {
			dir := s.FixtureDir(kind)
			entries, err := os.ReadDir(dir)
			if s.Count(kind) == 0 {
				// Covered by TestVendoredTreeMatchesSources, which asserts the
				// directory is absent rather than skipping it.
				continue
			}
			require.NoError(t, err, "%s/%s: sources.txt records fixtures here", s.Version, kind)

			for _, entry := range entries {
				name := filepath.Join(s.Version, string(kind), entry.Name())
				// Regular rather than "not a directory": DirEntry reports the
				// link itself, so a symlink is neither, and its size is the
				// length of the target path rather than of any document.
				assert.True(t, entry.Type().IsRegular(), "%s: only regular files are vendored", name)
				assert.Equal(t, ".yaml", filepath.Ext(entry.Name()), "%s: only .yaml documents are vendored", name)

				info, err := entry.Info()
				require.NoError(t, err)
				assert.Positive(t, info.Size(), "%s: an empty fixture asserts nothing", name)
			}
		}
	}
}

// TestVendoredTreeHasNothingElse checks the complement. Every other guard walks
// the versions sources.txt names, so a stale directory from a renamed version,
// or a stray file at the root, is invisible to all of them.
func TestVendoredTreeHasNothingElse(t *testing.T) {
	want := map[string]bool{"README.md": true, "sources.txt": true}
	for _, s := range loadSources(t) {
		want[s.Version] = true
	}

	entries, err := os.ReadDir(Dir())
	require.NoError(t, err)
	for _, entry := range entries {
		assert.True(t, want[entry.Name()],
			"%s is not recorded in sources.txt, so nothing else here checks it", entry.Name())
	}

	for _, s := range loadSources(t) {
		sub, err := os.ReadDir(filepath.Join(Dir(), s.Version))
		require.NoError(t, err)
		for _, entry := range sub {
			assert.Contains(t, []string{"pass", "fail"}, entry.Name(),
				"%s/%s: a version holds only pass and fail", s.Version, entry.Name())
		}
	}
}

// TestKnownFixturesArePresent names fixtures this project already reasons
// about, so a refresh that kept the counts while dropping the interesting cases
// is caught by more than an arithmetic check.
func TestKnownFixturesArePresent(t *testing.T) {
	byVersion := make(map[string]Source, 4)
	for _, s := range loadSources(t) {
		byVersion[s.Version] = s
	}

	for _, tc := range []struct {
		version string
		kind    Kind
		name    string
		why     string
	}{
		{"3.0", KindPass, "petstore.yaml", "the archived documents are the only fixtures 3.0 publishes"},
		{"3.1", KindFail, "parameter-object-header-allowReserved.yaml", "the documented v1 divergence, which a bool cannot express"},
		{"3.2", KindPass, "operation-object-example.yaml", "schema-valid and specification-invalid, so it stays rejected"},
		{"3.2", KindFail, "server_enum_empty.yaml", "a server variable enum that must not be empty"},
		{"3.3", KindFail, "server_enum_empty.yaml", "3.3 restates the constraints 3.2 states, so the suite carries them forward"},
	} {
		t.Run(tc.version+"/"+string(tc.kind)+"/"+tc.name, func(t *testing.T) {
			names, err := byVersion[tc.version].Fixtures(tc.kind)
			require.NoError(t, err)
			assert.Contains(t, names, tc.name, tc.why)
		})
	}
}

// TestVersionsDifferWhereTheSpecificationDid pins fixtures that change sides
// between versions, which pins a boundary no digest can explain.
//
// A digest proves the versions hold different bytes; it cannot say the
// difference is the right one. 3.2 widened where `allowReserved` may appear, so
// two documents that are negative fixtures at 3.1 are positive ones at 3.2 and
// 3.3, and a refresh that pointed 3.1 at a later branch would satisfy every
// other guard here.
func TestVersionsDifferWhereTheSpecificationDid(t *testing.T) {
	byVersion := make(map[string]Source, 4)
	for _, s := range loadSources(t) {
		byVersion[s.Version] = s
	}

	for _, name := range []string{
		"parameter-object-path-allowReserved.yaml",
		"parameter-object-cookie-form-allowReserved.yaml",
	} {
		t.Run(name, func(t *testing.T) {
			for _, tc := range []struct {
				version string
				kind    Kind
			}{
				{"3.1", KindFail},
				{"3.2", KindPass},
				{"3.3", KindPass},
			} {
				names, err := byVersion[tc.version].Fixtures(tc.kind)
				require.NoError(t, err)
				assert.Contains(t, names, name,
					"%s should be a %s fixture at %s, since 3.2 widened where allowReserved may appear",
					name, tc.kind, tc.version)
			}
		})
	}

	// 3.2 and 3.3 are the pair a name-level check cannot separate, so the one
	// asymmetry between them is worth pinning by name as well as by digest.
	for _, name := range []string{
		"discriminator-object-unexpected-property.yaml",
		"xml-object-unexpected-property.yaml",
	} {
		t.Run("3.3 only: "+name, func(t *testing.T) {
			at33, err := byVersion["3.3"].Fixtures(KindFail)
			require.NoError(t, err)
			at32, err := byVersion["3.2"].Fixtures(KindFail)
			require.NoError(t, err)

			assert.Contains(t, at33, name, "3.3 adds this negative fixture")
			assert.NotContains(t, at32, name, "3.2 does not have it, which is what separates the two")
		})
	}
}
