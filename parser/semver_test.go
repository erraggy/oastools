package parser

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantMajor  int
		wantMinor  int
		wantPatch  int
		wantPre    string
		shouldFail bool
	}{
		{
			name:      "simple 2.0",
			input:     "2.0",
			wantMajor: 2,
			wantMinor: 0,
			wantPatch: 0,
		},
		{
			name:      "standard 3.0.0",
			input:     "3.0.0",
			wantMajor: 3,
			wantMinor: 0,
			wantPatch: 0,
		},
		{
			name:      "patch version 3.0.1",
			input:     "3.0.1",
			wantMajor: 3,
			wantMinor: 0,
			wantPatch: 1,
		},
		{
			name:      "minor version 3.1.0",
			input:     "3.1.0",
			wantMajor: 3,
			wantMinor: 1,
			wantPatch: 0,
		},
		{
			name:      "with prerelease 3.0.0-rc1",
			input:     "3.0.0-rc1",
			wantMajor: 3,
			wantMinor: 0,
			wantPatch: 0,
			wantPre:   "rc1",
		},
		{
			name:      "with prerelease 3.1.0-beta.2",
			input:     "3.1.0-beta.2",
			wantMajor: 3,
			wantMinor: 1,
			wantPatch: 0,
			wantPre:   "beta.2",
		},
		{
			name:       "invalid empty",
			input:      "",
			shouldFail: true,
		},
		{
			name:       "invalid single number",
			input:      "3",
			shouldFail: true,
		},
		{
			name:       "invalid too many parts",
			input:      "3.0.0.1",
			shouldFail: true,
		},
		{
			name:       "invalid non-numeric",
			input:      "three.zero.zero",
			shouldFail: true,
		},
		{
			name:       "invalid negative",
			input:      "3.-1.0",
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ver, err := parseVersion(tt.input)
			if tt.shouldFail {
				if err == nil {
					t.Errorf("parseVersion(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseVersion(%q) unexpected error: %v", tt.input, err)
			}

			if ver.major != tt.wantMajor {
				t.Errorf("major = %d, want %d", ver.major, tt.wantMajor)
			}
			if ver.minor != tt.wantMinor {
				t.Errorf("minor = %d, want %d", ver.minor, tt.wantMinor)
			}
			if ver.patch != tt.wantPatch {
				t.Errorf("patch = %d, want %d", ver.patch, tt.wantPatch)
			}
			if ver.prerelease != tt.wantPre {
				t.Errorf("prerelease = %q, want %q", ver.prerelease, tt.wantPre)
			}
		})
	}
}

func TestVersionSegments(t *testing.T) {
	ver, _ := parseVersion("3.1.2")
	segments := ver.segments()

	if len(segments) != 3 {
		t.Fatalf("segments length = %d, want 3", len(segments))
	}
	if segments[0] != 3 || segments[1] != 1 || segments[2] != 2 {
		t.Errorf("segments = %v, want [3 1 2]", segments)
	}
}

func TestVersionLessThan(t *testing.T) {
	tests := []struct {
		name string
		v1   string
		v2   string
		want bool
	}{
		{"major less", "2.0.0", "3.0.0", true},
		{"major greater", "3.0.0", "2.0.0", false},
		{"minor less", "3.0.0", "3.1.0", true},
		{"minor greater", "3.1.0", "3.0.0", false},
		{"patch less", "3.0.0", "3.0.1", true},
		{"patch greater", "3.0.1", "3.0.0", false},
		{"equal", "3.0.0", "3.0.0", false},
		{"prerelease less than release", "3.0.0-rc1", "3.0.0", true},
		{"release not less than prerelease", "3.0.0", "3.0.0-rc1", false},
		{"prerelease comparison", "3.0.0-alpha", "3.0.0-beta", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v1, err := parseVersion(tt.v1)
			if err != nil {
				t.Fatalf("parseVersion(%q) error: %v", tt.v1, err)
			}
			v2, err := parseVersion(tt.v2)
			if err != nil {
				t.Fatalf("parseVersion(%q) error: %v", tt.v2, err)
			}

			got := v1.lessThan(v2)
			if got != tt.want {
				t.Errorf("%s.lessThan(%s) = %v, want %v", tt.v1, tt.v2, got, tt.want)
			}
		})
	}
}

func TestVersionGreaterThanOrEqual(t *testing.T) {
	tests := []struct {
		name string
		v1   string
		v2   string
		want bool
	}{
		{"greater major", "3.0.0", "2.0.0", true},
		{"greater minor", "3.1.0", "3.0.0", true},
		{"greater patch", "3.0.1", "3.0.0", true},
		{"equal", "3.0.0", "3.0.0", true},
		{"less", "2.0.0", "3.0.0", false},
		{"release >= prerelease", "3.0.0", "3.0.0-rc1", true},
		{"prerelease < release", "3.0.0-rc1", "3.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v1, err := parseVersion(tt.v1)
			if err != nil {
				t.Fatalf("parseVersion(%q) error: %v", tt.v1, err)
			}
			v2, err := parseVersion(tt.v2)
			if err != nil {
				t.Fatalf("parseVersion(%q) error: %v", tt.v2, err)
			}

			got := v1.greaterThanOrEqual(v2)
			if got != tt.want {
				t.Errorf("%s.greaterThanOrEqual(%s) = %v, want %v", tt.v1, tt.v2, got, tt.want)
			}
		})
	}
}
