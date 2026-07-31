// Package driftguard holds the tests that keep a parser struct field from being
// added in one place and forgotten in the others.
//
// A field on a parser type has to be handled in several independent places: the
// hand-built slow MarshalJSON path, the structural hasher, each type's equality
// function, and the joiner's schema comparison. Nothing connected them, so a
// field could be added, tested and shipped while silently vanishing from JSON
// documents that carried an extension, or while leaving two different schemas
// looking identical. That is issue #397, and #414 found nine more fields in the
// same state afterwards.
//
// The guards live here, in one internal package with no exported symbols, rather
// than spread across the packages they exercise. Keeping them together means the
// reflection helper they share is not a maintained API for anyone else, and the
// list of deliberate exclusions sits next to the checks that honor it.
//
// Everything is in _test.go files; this file exists only to give the package a
// clause and a home for this explanation.
package driftguard
