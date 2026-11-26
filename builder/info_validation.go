package builder

import (
	"github.com/erraggy/oastools/internal/inforules"
)

// infoHints names the setter for a required field, appended to its message.
var infoHints = map[string]string{
	"info.title":   "; call SetTitle",
	"info.version": " (the version of your API, not of the OpenAPI Specification); call SetVersion",
}

// infoErrors returns every problem with the info object for the builder's
// version. It runs at build time, since SetInfo keeps the caller's pointer.
// The rules are shared with the validator via internal/inforules.
func (b *Builder) infoErrors() []*BuilderError {
	if b.info == nil {
		return []*BuilderError{NewMissingInfoError()}
	}

	problems := inforules.Check(b.info, b.version)
	if len(problems) == 0 {
		return nil
	}
	errs := make([]*BuilderError, 0, len(problems))
	for _, p := range problems {
		errs = append(errs, NewInfoError(p.Path, p.Message+infoHints[p.Path]))
	}
	return errs
}
