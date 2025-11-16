# Issues To Address
The following issues have been identified and need to be addressed.

## 1. Validator fails to validate `$ref` paths properly point to schemas that are resolvable
If for instance an OAS3.2.0 document defines an Operation with a `requestBody.content.application/json.schema.$ref` is set to `"#/definitions/foo.Bar"` (a swagger 2.0 reference), the validation doens't report any errors or warnings.

## 2. Converter fails to update `$ref` paths when the schemas need to be moved
If for instance you are converting a swagger 2.0 document to an OAS3.2.0 document, the schemas will need to be moved from their source location at the root level `definitions` into their destination location in root level: `components.schemas`, but even though the schemas are all properly moved, the string values of all `$ref` pointers to them are left unchanged.

## 3 Unit test coverage failed to identify issues 1 & 2 above
Clearly there is a gap in unit test coverage since these 2 critical issues were never identified.
