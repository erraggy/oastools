// Package conformance reads the OpenAPI Specification conformance suite
// vendored under testdata/conformance.
//
// The suite is OAI's own, published per version at tests/schema/{pass,fail} on
// the version branches, plus the six archived 3.0 documents on main. It is
// vendored at an exact commit rather than fetched, because the schemas version
// independently of the specification and OAI publishes no endpoint naming the
// current one. testdata/conformance/sources.txt records what is pinned, and
// scripts/conformance-vendor.sh is what materializes it.
//
// What the suite is an oracle for matters when reading a result. The fixtures
// assert validity against the published JSON Schema, not against the prose, and
// the schema's own README says it covers only the mandatory aspects of the OAS.
// So the suite is sound in the rejection direction and incomplete in the
// acceptance one: a fail fixture that oastools accepts is a gap, while a pass
// fixture that oastools rejects is a triage item that may turn out to be
// correct. tests/schema/pass/operation-object-example.yaml on 3.2 is the known
// case, being schema-valid and specification-invalid.
package conformance
