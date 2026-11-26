package builder

import (
	"github.com/erraggy/oastools/internal/inforules"
)

// infoHints adds the builder call that sets a required field to the shared
// rule's message, since a builder user reaches the field through a setter.
var infoHints = map[string]string{
	"info.title":   "; call SetTitle",
	"info.version": " (the version of your API, not of the OpenAPI Specification); call SetVersion",
}

// infoErrors returns every problem that makes the info object invalid for the
// builder's OAS version. It runs at build time and stores nothing, since
// SetInfo keeps the caller's pointer and the object can change until then.
//
// The rules themselves live in internal/inforules, which the validator also
// uses, so the builder never accepts an info object the validator rejects.
// They are listed in the package documentation.
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
