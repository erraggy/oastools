# OpenAPI Specification conformance suite (vendored)

OAI's own conformance fixtures, vendored at a pinned commit. Unlike
`testdata/corpus`, which is downloaded and gitignored, these files are committed:
measuring conformance should need no network, and should not change because an
upstream branch moved.

- **Source**: <https://github.com/OAI/OpenAPI-Specification>
- **License**: Apache-2.0, the upstream project's own
- **Pins**: [sources.txt](sources.txt), one record per version
- **Refresh**: `make conformance-vendor` re-materializes the tree at those pins;
  `make conformance-update` moves the pins to each branch's current head

```
3.0/pass/    3.1/pass/  3.1/fail/    3.2/pass/  3.2/fail/    3.3/pass/  3.3/fail/
```

## What the suite is an oracle for

The fixtures assert validity **against the published JSON Schema**, not against
the prose. The upstream directory is `tests/schema`, and the schema's own README
says it validates only the mandatory aspects of the OAS. A document can therefore
be a legitimate `pass` fixture while violating a prose MUST that no schema can
express.

So the suite is sound in the rejection direction and incomplete in the acceptance
one:

| Result | Reading |
|---|---|
| a `fail` fixture that oastools accepts | a gap in oastools |
| a `pass` fixture that oastools rejects | a triage item, which may be correct |

`3.2/pass/operation-object-example.yaml` is the known case of the second kind. It
is schema-valid and specification-invalid, violating both path-template parameter
correspondence and the requirement that a security requirement name a declared
scheme, so oastools is right to reject it.

The number to watch is not "every pass fixture passes" but "no *unexplained*
rejection".

## Where the fixtures come from

3.1, 3.2 and 3.3 publish `tests/schema/{pass,fail}` on their version branches.
`main` carries no such tree; its only fixtures are the six archived 3.0 documents
under `_archive_/schemas/v3.0/pass`, which are vendored here because they exist.

2.0 publishes no fixtures at all, and none are authored here: a hand-written
suite would measure this project's reading of the specification rather than OAI's.

Only `.yaml` documents are vendored. Upstream keeps its own JavaScript harness
beside the fixtures, and 3.3 keeps `minimal-objects.yaml`, a table of object
stubs rather than an OpenAPI document.

## Known upstream defect

`tests/schema/failparameter-object-path-allowReserved.yaml` on `v3.1-dev` is a
negative fixture whose path is missing a separator, so it sits beside the `fail`
directory rather than inside it and upstream's own harness never runs it. It is
not a copy of `fail/parameter-object-path-allowReserved.yaml`: the skipped file
sets `required: true`, so `allowReserved` is the only violation it carries, while
the one that does run omits `required` on an `in: path` parameter and can be
rejected for that instead.

It is not vendored here, since vendoring a file upstream does not test would
misrepresent the suite.
