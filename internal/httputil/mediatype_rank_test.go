package httputil

import "testing"

func TestMediaTypeRank(t *testing.T) {
	for _, tc := range []struct {
		mediaType string
		want      int
	}{
		{"application/json", MediaTypeRankJSON},
		{"APPLICATION/JSON", MediaTypeRankJSON},
		// Parameters do not change what the body is.
		{"application/json; charset=utf-8", MediaTypeRankJSON},
		{"application/problem+json", MediaTypeRankJSONSuffix},
		{"application/geo+json", MediaTypeRankJSONSuffix},
		{"application/xml", MediaTypeRankOther},
		{"text/plain", MediaTypeRankOther},
		// A media type that does not parse is ranked last, not rejected: the
		// caller is choosing between what a document actually offers. The
		// suffix is not consulted for one of these, or a name ending in +json
		// that is not a media type at all would outrank application/xml.
		{"not a media type", MediaTypeRankOther},
		{"not a media type+json", MediaTypeRankOther},
		{"///+json", MediaTypeRankOther},
		{"", MediaTypeRankOther},
		// A bare token parses, both for mime.ParseMediaType and for this
		// package's own IsValidMediaType, so it is a media type as far as
		// anything here is concerned and its suffix counts. Ranking it lower
		// would put a second notion of validity in a package that has one.
		{"garbage+json", MediaTypeRankJSONSuffix},
	} {
		if got := MediaTypeRank(tc.mediaType); got != tc.want {
			t.Errorf("MediaTypeRank(%q) = %d, want %d", tc.mediaType, got, tc.want)
		}
	}
}

func TestPreferredMediaType(t *testing.T) {
	for _, tc := range []struct {
		a, b, want string
	}{
		{"application/json", "application/xml", "application/json"},
		{"application/xml", "application/json", "application/json"},
		{"application/json", "application/problem+json", "application/json"},
		{"application/geo+json", "application/xml", "application/geo+json"},
		// Same rank, so the name decides and the answer does not depend on
		// which side it arrived on.
		{"application/geo+json", "application/ld+json", "application/geo+json"},
		{"application/ld+json", "application/geo+json", "application/geo+json"},
		{"text/plain", "application/xml", "application/xml"},
	} {
		if got := PreferredMediaType(tc.a, tc.b); got != tc.want {
			t.Errorf("PreferredMediaType(%q, %q) = %q, want %q", tc.a, tc.b, got, tc.want)
		}
	}
}

// TestPreferredMediaTypeIsSymmetric is the property the caller depends on: the
// answer cannot depend on the order two candidates are compared in, or a map
// range would still decide it.
func TestPreferredMediaTypeIsSymmetric(t *testing.T) {
	names := []string{
		"application/json", "application/problem+json", "application/geo+json",
		"application/ld+json", "application/xml", "text/plain", "application/atom+xml", "",
	}
	for _, a := range names {
		for _, b := range names {
			if PreferredMediaType(a, b) != PreferredMediaType(b, a) {
				t.Errorf("PreferredMediaType is not symmetric for %q and %q", a, b)
			}
		}
	}
}
