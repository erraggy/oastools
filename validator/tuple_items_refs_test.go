// tuple_items_refs_test.go covers reference validation against the OAS 2.0 tuple
// form of `items`.
package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const danglingTupleRefSpec = `
swagger: "2.0"
info:
  title: petstore
  version: "1.0.0"
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: successful operation
          schema:
            $ref: "#/definitions/PetTuple"
definitions:
  PetTuple:
    type: array
    items:
      - type: string
      - $ref: "#/definitions/PetDetails"
`

const resolvedTupleRefSpec = danglingTupleRefSpec + `
  PetDetails:
    type: object
    properties:
      name:
        type: string
`

// TestDanglingRefInTupleItemsIsReported covers #502: a $ref held in an element
// of a tuple-form `items` was never visited, so a document pointing at a
// definition it does not have passed validation.
func TestDanglingRefInTupleItemsIsReported(t *testing.T) {
	result := validateSpec(t, danglingTupleRefSpec)

	assert.False(t, result.Valid, "a document with a dangling $ref should not validate")
	assert.True(t, resultHasMessage(result, "#/definitions/PetDetails"),
		"the unresolvable $ref inside a tuple element was not reported; errors: %v", result.Errors)
}

// TestResolvedRefInTupleItemsIsAccepted is the other half: the same position
// with a target that exists must stay valid.
func TestResolvedRefInTupleItemsIsAccepted(t *testing.T) {
	result := validateSpec(t, resolvedTupleRefSpec)

	assert.True(t, result.Valid, "a resolvable $ref in a tuple element was rejected: %v", result.Errors)
}
