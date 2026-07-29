// Package naming provides shared naming utilities for oastools packages: case
// conversion, and the charset rule OAS 3.x imposes on component names.
//
// This internal package contains common string transformation functions used
// by multiple oastools packages including builder and joiner. Functions include
// ToPascalCase, ToCamelCase, ToSnakeCase, ToKebabCase, and ToTitleCase.
//
// These functions are used for:
//   - Builder package: Schema and operation naming from titles
//   - Joiner package: Template functions for operation-aware schema renaming
//   - Fixer package: Operation ID naming
//
// # Component Names
//
// [ComponentNamePattern] and [IsValidComponentName] state, once, the charset
// OAS 3.x requires of every key of the Components Object. The validator asks
// whether a document's names are legal and the fixer asks which characters it
// may keep when building a replacement; both consume the same definition so
// neither can drift from the spec or from the other.
//
// As an internal package, these functions are not part of the public API
// and may change without notice.
package naming
