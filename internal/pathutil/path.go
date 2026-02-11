package pathutil

import "regexp"

// PathParamRegex matches path template parameters like {paramName}.
// It captures the parameter name inside the braces.
var PathParamRegex = regexp.MustCompile(`\{([^}]+)\}`)
