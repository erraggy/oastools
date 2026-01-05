// Package naming provides shared case conversion utilities for oastools packages.
//
// This internal package contains common string transformation functions used
// by multiple oastools packages including builder and joiner. Functions include
// ToPascalCase, ToCamelCase, ToSnakeCase, ToKebabCase, and ToTitleCase.
//
// These functions are used for:
//   - Builder package: Schema and operation naming from titles
//   - Joiner package: Template functions for operation-aware schema renaming
//
// As an internal package, these functions are not part of the public API
// and may change without notice.
package naming
