package parser

// OASVersion represents each canonical version of the OpenAPI Specification that may be found at:
// https://github.com/OAI/OpenAPI-Specification/releases
type OASVersion int

const (
	// Unknown represents an unknown or invalid OAS version
	Unknown OASVersion = iota
	// OASVersion20 OpenAPI Specification Version 2.0 (Swagger)
	OASVersion20
	// OASVersion300 OpenAPI Specification Version 3.0.0
	OASVersion300
	// OASVersion301  OpenAPI Specification Version 3.0.1
	OASVersion301
	// OASVersion302  OpenAPI Specification Version 3.0.2
	OASVersion302
	// OASVersion303  OpenAPI Specification Version 3.0.3
	OASVersion303
	// OASVersion304  OpenAPI Specification Version 3.0.4
	OASVersion304
	// OASVersion310  OpenAPI Specification Version 3.1.0
	OASVersion310
	// OASVersion311  OpenAPI Specification Version 3.1.1
	OASVersion311
	// OASVersion312  OpenAPI Specification Version 3.1.2
	OASVersion312
	// OASVersion320  OpenAPI Specification Version 3.2.0
	OASVersion320
)

var (
	versionToString = map[OASVersion]string{
		OASVersion20:  "2.0",
		OASVersion300: "3.0.0",
		OASVersion301: "3.0.1",
		OASVersion302: "3.0.2",
		OASVersion303: "3.0.3",
		OASVersion304: "3.0.4",
		OASVersion310: "3.1.0",
		OASVersion311: "3.1.1",
		OASVersion312: "3.1.2",
		OASVersion320: "3.2.0",
	}

	stringToVersion = func() map[string]OASVersion {
		m := make(map[string]OASVersion, 10)
		for k, v := range versionToString {
			m[v] = k
		}
		return m
	}()
)

func (v OASVersion) String() string {
	if s, ok := versionToString[v]; ok {
		return s
	}
	return "unknown"
}

// IsValid returns true if this is a valid version
func (v OASVersion) IsValid() bool {
	_, ok := versionToString[v]
	return ok
}

// ParseVersion will attempt to parse the string s into an OASVersion, and returns false if not valid
func ParseVersion(s string) (OASVersion, bool) {
	v, ok := stringToVersion[s]
	return v, ok
}
