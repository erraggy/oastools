// Package corpusutil provides utilities for loading and managing the
// integration test corpus of real-world public OpenAPI specifications.
//
// The corpus includes 10 carefully selected specifications spanning:
//   - OAS versions: 2.0, 3.0.0, 3.0.3, 3.0.4, 3.1.0
//   - Formats: JSON and YAML
//   - Sizes: From 20KB (Petstore) to 34MB (Microsoft Graph)
//   - Domains: FinTech, Developer Tools, Enterprise, etc.
//
// # Usage
//
// Tests should use the SkipIfNotCached helper to gracefully skip when
// corpus files are not available:
//
//	func TestCorpus_Parse(t *testing.T) {
//	    for _, spec := range corpusutil.GetSpecs(false) {
//	        t.Run(spec.Name, func(t *testing.T) {
//	            corpusutil.SkipIfNotCached(t, spec)
//	            // ... test implementation
//	        })
//	    }
//	}
//
// # Downloading the Corpus
//
// Run `make corpus-download` to fetch all specifications to testdata/corpus/.
// These files are not committed to the repository.
package corpusutil
