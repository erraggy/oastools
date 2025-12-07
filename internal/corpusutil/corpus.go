package corpusutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// SpecInfo contains metadata about a corpus specification.
type SpecInfo struct {
	Name             string // Human-readable name (e.g., "Stripe")
	Filename         string // Local filename in testdata/corpus/
	URL              string // Remote source URL
	OASVersion       string // Expected OAS version (e.g., "3.0.0", "2.0")
	Format           string // "json" or "yaml"
	ExpectedValid    bool   // Whether strict validation should pass
	ExpectedErrors   int    // Approximate error count if invalid (for tolerance)
	IsLarge          bool   // True if file size >5MB
	SizeBytes        int64  // Approximate file size in bytes
	HasParsingIssues bool   // True if spec uses features our parser doesn't handle
}

// GetLocalPath returns the absolute path to the cached spec file.
func (s SpecInfo) GetLocalPath() string {
	return filepath.Join(CorpusDir(), s.Filename)
}

// IsAvailable checks if the spec is cached locally.
func (s SpecInfo) IsAvailable() bool {
	_, err := os.Stat(s.GetLocalPath())
	return err == nil
}

// Corpus defines all 10 public specifications for integration testing.
// Specifications are ordered by size (smallest first) for faster test feedback.
var Corpus = []SpecInfo{
	{
		Name:           "Petstore",
		Filename:       "petstore-swagger.json",
		URL:            "https://petstore.swagger.io/v2/swagger.json",
		OASVersion:     "2.0",
		Format:         "json",
		ExpectedValid:  true,
		ExpectedErrors: 0,
		IsLarge:        false,
		SizeBytes:      20_000,
	},
	{
		Name:             "DigitalOcean",
		Filename:         "digitalocean-public.v2.yaml",
		URL:              "https://raw.githubusercontent.com/digitalocean/openapi/main/specification/DigitalOcean-public.v2.yaml",
		OASVersion:       "3.0.0",
		Format:           "yaml",
		ExpectedValid:    true,
		ExpectedErrors:   0,
		IsLarge:          false,
		SizeBytes:        200_000,
		HasParsingIssues: true, // Uses $ref in info.description which our parser can't handle
	},
	{
		Name:           "Asana",
		Filename:       "asana-oas.yaml",
		URL:            "https://raw.githubusercontent.com/Asana/openapi/master/defs/asana_oas.yaml",
		OASVersion:     "3.0.0",
		Format:         "yaml",
		ExpectedValid:  false,
		ExpectedErrors: 302,
		IsLarge:        false,
		SizeBytes:      405_000,
	},
	{
		Name:           "GoogleMaps",
		Filename:       "google-maps-platform.json",
		URL:            "https://raw.githubusercontent.com/googlemaps/openapi-specification/main/dist/google-maps-platform-openapi3.json",
		OASVersion:     "3.0.3",
		Format:         "json",
		ExpectedValid:  false,
		ExpectedErrors: 228,
		IsLarge:        false,
		SizeBytes:      500_000,
	},
	{
		Name:           "USNWS",
		Filename:       "nws-openapi.json",
		URL:            "https://api.weather.gov/openapi.json",
		OASVersion:     "3.0.3",
		Format:         "json",
		ExpectedValid:  false,
		ExpectedErrors: 156,
		IsLarge:        false,
		SizeBytes:      800_000,
	},
	{
		Name:           "Plaid",
		Filename:       "plaid-2020-09-14.yml",
		URL:            "https://raw.githubusercontent.com/plaid/plaid-openapi/master/2020-09-14.yml",
		OASVersion:     "3.0.0",
		Format:         "yaml",
		ExpectedValid:  false,
		ExpectedErrors: 101,
		IsLarge:        false,
		SizeBytes:      1_200_000,
	},
	{
		Name:           "Discord",
		Filename:       "discord-openapi.json",
		URL:            "https://raw.githubusercontent.com/discord/discord-api-spec/main/specs/openapi.json",
		OASVersion:     "3.1.0",
		Format:         "json",
		ExpectedValid:  true,
		ExpectedErrors: 0,
		IsLarge:        false,
		SizeBytes:      3_000_000,
	},
	{
		Name:           "GitHub",
		Filename:       "github-api.json",
		URL:            "https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.json",
		OASVersion:     "3.0.3",
		Format:         "json",
		ExpectedValid:  false,
		ExpectedErrors: 8000,
		IsLarge:        false,
		SizeBytes:      5_000_000,
	},
	{
		Name:           "Stripe",
		Filename:       "stripe-spec3.json",
		URL:            "https://raw.githubusercontent.com/stripe/openapi/master/openapi/spec3.json",
		OASVersion:     "3.0.0",
		Format:         "json",
		ExpectedValid:  true,
		ExpectedErrors: 0,
		IsLarge:        true,
		SizeBytes:      14_000_000,
	},
	{
		Name:           "MicrosoftGraph",
		Filename:       "msgraph-openapi.yaml",
		URL:            "https://raw.githubusercontent.com/microsoftgraph/msgraph-metadata/master/openapi/v1.0/openapi.yaml",
		OASVersion:     "3.0.4",
		Format:         "yaml",
		ExpectedValid:  false,
		ExpectedErrors: 30000,
		IsLarge:        true,
		SizeBytes:      15_000_000,
	},
}

// CorpusDir returns the absolute path to the corpus directory.
func CorpusDir() string {
	// Get the directory of this source file
	_, thisFile, _, ok := runtime.Caller(0)
	if ok {
		// Go up from internal/corpusutil to project root
		projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
		return filepath.Join(projectRoot, "testdata", "corpus")
	}
	// Fallback to relative path
	return filepath.Join("testdata", "corpus")
}

// GetSpecs returns specs filtered by the includeLarge flag.
func GetSpecs(includeLarge bool) []SpecInfo {
	result := make([]SpecInfo, 0, len(Corpus))
	for _, spec := range Corpus {
		if !includeLarge && spec.IsLarge {
			continue
		}
		result = append(result, spec)
	}
	return result
}

// GetValidSpecs returns only specs expected to pass validation.
func GetValidSpecs(includeLarge bool) []SpecInfo {
	result := make([]SpecInfo, 0)
	for _, spec := range GetSpecs(includeLarge) {
		if spec.ExpectedValid {
			result = append(result, spec)
		}
	}
	return result
}

// GetInvalidSpecs returns only specs expected to fail validation.
func GetInvalidSpecs(includeLarge bool) []SpecInfo {
	result := make([]SpecInfo, 0)
	for _, spec := range GetSpecs(includeLarge) {
		if !spec.ExpectedValid {
			result = append(result, spec)
		}
	}
	return result
}

// GetOAS2Specs returns OAS 2.0 (Swagger) specifications.
func GetOAS2Specs() []SpecInfo {
	result := make([]SpecInfo, 0)
	for _, spec := range Corpus {
		if spec.OASVersion == "2.0" {
			result = append(result, spec)
		}
	}
	return result
}

// GetOAS3Specs returns OAS 3.x specifications.
func GetOAS3Specs(includeLarge bool) []SpecInfo {
	result := make([]SpecInfo, 0)
	for _, spec := range GetSpecs(includeLarge) {
		if spec.OASVersion != "2.0" {
			result = append(result, spec)
		}
	}
	return result
}

// GetLargeSpecs returns only large (>5MB) specifications.
func GetLargeSpecs() []SpecInfo {
	result := make([]SpecInfo, 0)
	for _, spec := range Corpus {
		if spec.IsLarge {
			result = append(result, spec)
		}
	}
	return result
}

// GetByName returns a spec by name, or nil if not found.
func GetByName(name string) *SpecInfo {
	for i := range Corpus {
		if Corpus[i].Name == name {
			return &Corpus[i]
		}
	}
	return nil
}

// SkipIfNotCached skips the test if the corpus file is not available locally.
func SkipIfNotCached(t testing.TB, spec SpecInfo) {
	t.Helper()
	if !spec.IsAvailable() {
		t.Skipf("Corpus file %s not cached locally; run 'make corpus-download' to fetch", spec.Filename)
	}
}

// SkipLargeInShortMode skips large specs when running with -short flag.
func SkipLargeInShortMode(t testing.TB, spec SpecInfo) {
	t.Helper()
	if testing.Short() && spec.IsLarge {
		t.Skipf("Skipping large spec %s in short mode", spec.Name)
	}
}

// SkipIfEnvSet skips the test if the specified environment variable is set to "1".
func SkipIfEnvSet(t testing.TB, envVar string) {
	t.Helper()
	if os.Getenv(envVar) == "1" {
		t.Skipf("Skipping test due to %s=1", envVar)
	}
}

// SkipIfHasParsingIssues skips specs that have known parsing issues.
func SkipIfHasParsingIssues(t testing.TB, spec SpecInfo) {
	t.Helper()
	if spec.HasParsingIssues {
		t.Skipf("Skipping %s: has known parsing issues", spec.Name)
	}
}

// GetParseableSpecs returns only specs without known parsing issues.
func GetParseableSpecs(includeLarge bool) []SpecInfo {
	result := make([]SpecInfo, 0)
	for _, spec := range GetSpecs(includeLarge) {
		if !spec.HasParsingIssues {
			result = append(result, spec)
		}
	}
	return result
}
