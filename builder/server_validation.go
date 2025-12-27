package builder

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/erraggy/oastools/httpvalidator"
)

// validationMiddleware creates the validation middleware.
func validationMiddleware(v *httpvalidator.Validator, cfg ValidationConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.IncludeRequestValidation {
				next.ServeHTTP(w, r)
				return
			}

			result, err := v.ValidateRequest(r)
			if err != nil {
				writeValidationError(w, http.StatusInternalServerError, err.Error())
				return
			}

			hasErrors := len(result.Errors) > 0
			hasWarnings := len(result.Warnings) > 0 && cfg.StrictMode

			if hasErrors || hasWarnings {
				if cfg.OnValidationError != nil {
					cfg.OnValidationError(w, r, result)
					return
				}
				writeValidationResult(w, result)
				return
			}

			// Store validated params in context for handler access
			ctx := contextWithValidationResult(r.Context(), result)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// validationResultKey is the context key for validation results.
type validationResultKey struct{}

// contextWithValidationResult adds the validation result to the context.
func contextWithValidationResult(ctx context.Context, result *httpvalidator.RequestValidationResult) context.Context {
	return context.WithValue(ctx, validationResultKey{}, result)
}

// validationResultFromContext retrieves the validation result from the context.
func validationResultFromContext(ctx context.Context) *httpvalidator.RequestValidationResult {
	if result, ok := ctx.Value(validationResultKey{}).(*httpvalidator.RequestValidationResult); ok {
		return result
	}
	return nil
}

// writeValidationError writes a simple validation error response.
func writeValidationError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

// writeValidationResult writes a detailed validation result response.
func writeValidationResult(w http.ResponseWriter, result *httpvalidator.RequestValidationResult) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	errors := make([]map[string]string, 0, len(result.Errors))
	for _, e := range result.Errors {
		errors = append(errors, map[string]string{
			"path":    e.Path,
			"message": e.Message,
		})
	}

	response := map[string]any{
		"error":  "validation failed",
		"errors": errors,
	}

	if len(result.Warnings) > 0 {
		warnings := make([]map[string]string, 0, len(result.Warnings))
		for _, w := range result.Warnings {
			warnings = append(warnings, map[string]string{
				"path":    w.Path,
				"message": w.Message,
			})
		}
		response["warnings"] = warnings
	}

	_ = json.NewEncoder(w).Encode(response)
}
