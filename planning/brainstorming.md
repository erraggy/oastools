The OASTools software is designed for validating, parsing, converting, diffing, and joining OpenAPI specifications (OAS) spanning versions 2.0 through 3.2.0 [1-4].

Based on the information available, studying ways to improve and extend OASTools can focus on enhancing existing capabilities (performance, API use) and incorporating support for real-world scenarios outlined as current limitations or complex features within the OpenAPI Specification (OAS).

***

## I. Improvements (Performance and API Usability)

OASTools, which is built using the Go programming language [5, 6], already emphasizes performance and offers distinct API paths to maximize efficiency, but there is always scope for further refinement.

### 1. Performance

The sources indicate that **significant performance optimizations** have already been a focus, particularly through the introduction of the "ParseOnce pattern" [7, 8].

| Operation | Performance Highlight | Context |
| :--- | :--- | :--- |
| **Joining** | **154x faster** with `JoinParsed` vs `Join` | Uses the `ParseOnce` pattern (parse once, join many) [9, 10]. |
| **Diffing** | **58x faster** with `DiffParsed` vs `Diff` | Uses the `ParseOnce` pattern (parse once, diff many) [9, 10]. |
| **Validation** | **30x faster** with `ValidateParsed` vs `Validate` | Uses the `ParseOnce` pattern (parse once, validate many) [9, 10]. |
| **Conversion** | **9x faster** with `ConvertParsed` vs `Convert` | Uses the `ParseOnce` pattern (parse once, convert many) [9, 10]. |
| **General** | **25-32% faster** JSON marshaling and **29-37% fewer** memory allocations (as of v1.9.1) [9, 10]. |

To further pursue performance improvements, the focus could be on:

*   **Maximizing ParseOnce Adoption:** Encouraging or automating the use of the **Advanced API (Reusable Instances)**, which achieves these speed gains by leveraging the `ParseOnce` pattern for processing multiple files with the same configuration [11, 12].
*   **CLI Efficiency:** Although the CLI allows loading specs from files or URLs [13, 14], internal architecture could be investigated to ensure sequential CLI operations (if applicable) benefit from caching or repeated use of parsed results akin to the library's advanced API pattern.

### 2. Easier to Utilize API

The library already provides two API styles: a **Simple API** for quick, one-off operations (e.g., `parser.Parse`, `validator.Validate`), and an **Advanced API** designed for efficiency with reusable instances and explicit configuration (e.g., `parser.New()`, `v.ValidateParsed(result1)`) [11, 12, 15, 16].

Future API improvements could involve:

*   **Configuration Abstraction:** Streamlining the configuration of reusable instances (such as setting modes for the differ, `d.Mode = differ.ModeBreaking`) [11, 12] or setting parser options (`p.ResolveRefs = false`) [11, 12] to allow easier setup while retaining the performance benefits of the Advanced API.
*   **Centralized Error/Issue Handling:** The library includes internal shared utilities for `severity` and `issues` [17, 18]. Further standardization or documentation of how to consume the structured, severity-tracked issues (Info, Warning, Critical) returned by conversion and differing operations could enhance usability [7, 8].

***

## II. Extensions (Real-World Use Cases and Limitations)

Extensions can target known limitations in the current implementation and support complex features or ambiguities highlighted within the OAS specifications themselves.

### 1. Addressing Current Limitations on External References

The sources note clear limitations regarding external references (`$ref` values) that represent key areas for extension to support complex, distributed specifications [19, 20]:

*   **HTTP(S) References Not Supported:** Currently, OASTools only supports local file references for `$ref` values, excluding remote HTTP(S) references [19, 20]. Implementing support for remote reference resolution would be a critical extension for real-world APIs hosted across domains [19, 20].
*   **Incorrect URL-Loaded Resolution:** When loading a specification from a URL, a known limitation causes relative `$ref` paths to resolve against the *current local directory*, not relative to the source URL [19, 20]. Fixing this resolution logic is necessary for correctly handling specs that are composed of multiple remote files.

### 2. Supporting Complex OAS Features and Ambiguities

The OpenAPI Specification documents outline scenarios that are complex or have "implementation-defined" or "undefined" behavior [21]. Robust extensions could provide defined, interoperable handling for these cases:

#### A. Multi-Document Resolution and Ambiguity

In multi-document OpenAPI Descriptions (OADs), several features rely on **implicit connections** (not URI-based references) whose resolution can be implementation-defined [22, 23].

*   **Component and Tag Resolution:** For resolving component names (like schemas, parameters, or security schemes) or tag names in a referenced document, the behavior is implementation-defined, though tools are *recommended* to resolve from the **entry document** [24, 25]. OASTools could formalize this recommendation or provide explicit configuration for resolution scope.
*   **Ambiguous Path Templating:** The specification warns that templated paths with the same hierarchy but different templated names (e.g., `/pets/{petId}` and `/pets/{name}`) are considered identical and invalid [26-28]. Tooling to explicitly detect and flag ambiguous path matching beyond simple equivalence could be valuable.

#### B. Advanced Serialization and Parameter Handling

The OAS includes multiple styles for serializing complex parameters (arrays/objects) for query strings, some of which are non-RFC6570 compliant (`spaceDelimited`, `pipeDelimited`, `deepObject`) [29-31].

*   **Non-Standard Query Formats:** Extending `parser`, `validator`, and `converter` packages to provide comprehensive support and strict validation for these non-RFC6570 query serialization styles could address real-world needs where complex query structures are used [29-31].
*   **New Parameter Location:** Supporting the **`querystring`** parameter location introduced in OAS 3.2.0, which allows treating the entire URL query string as a single value using the `content` field [32]. This is distinct from the older `query` location [32].

#### C. Runtime Behavior Analysis

The specification utilizes **runtime expressions** in advanced features like **Link Objects** and **Callback Objects** (OAS 3.0+) [33-38].

*   **Runtime Expression Validation:** Extending the validation or parsing capabilities to analyze the syntax and semantic validity of these complex runtime expressions (e.g., `$request.body#/user/uuid` or `$response.header.Server`) [38-40] could help OAD authors ensure correctness before deployment.

***
*In essence, improving OASTools could focus on making the already fast Advanced API easier to use and making the CLI more efficient for bulk operations, while extending OASTools requires tackling the complexity of remote reference resolution and rigorously defining behavior for the numerous implementation-defined ambiguities found within modern OpenAPI Specification documents.*

## References

1.  **[ABNF]**: `https://www.rfc-editor.org/rfc/rfc5234` [1-3]
2.  **[CommonMark]**: `https://spec.commonmark.org/` [1-3]
3.  **[CommonMark-0.27]**: `https://spec.commonmark.org/0.27/` [1-3]
4.  **[DOM]**: `https://dom.spec.whatwg.org/` [3]
5.  **[HTML]**: `https://html.spec.whatwg.org/multipage/` [4]
6.  **[HTML401]**: `https://www.w3.org/TR/html401/` [4, 5]
7.  **[IANA-HTTP-AUTHSCHEMES]**: `https://www.iana.org/assignments/http-authschemes/` [2, 3]
8.  **[IANA-HTTP-STATUS-CODES]**: `https://www.iana.org/assignments/http-status-codes/` [1-3, 5]
9.  **[JSON-Reference]**: `https://datatracker.ietf.org/doc/html/draft-pbryan-zyp-json-ref-03` [1, 5]
10. **[JSON-Schema-04]**: `https://datatracker.ietf.org/doc/html/draft-zyp-json-schema-04` [5]
11. **[JSON-Schema-05]**: `https://datatracker.ietf.org/doc/html/draft-wright-json-schema-00` [1]
12. **[JSON-Schema-2020-12]**: `https://datatracker.ietf.org/doc/html/draft-bhutton-json-schema-01` [2]
13. **[JSON-Schema-Validation-04]**: `https://datatracker.ietf.org/doc/html/draft-fge-json-schema-validation-00` [5]
14. **[JSON-Schema-Validation-05]**: `https://datatracker.ietf.org/doc/html/draft-wright-json-schema-validation-00` [1]
15. **[JSON-Schema-Validation-2020-12]**: `https://datatracker.ietf.org/doc/html/draft-bhutton-json-schema-validation-01` [2]
16. **[OpenAPI-Learn]**: `https://learn.openapis.org/` [4]
17. **[OpenAPI-Registry]**: `https://spec.openapis.org/registry/index.html` [3]
18. **[OpenID-Connect-Core]**: `https://openid.net/specs/openid-connect-core-1_0.html` [3]
19. **[OpenID-Connect-Discovery]**: `https://openid.net/specs/openid-connect-discovery-1_0.html` [3]
20. **[RFC1738]**: `https://www.rfc-editor.org/rfc/rfc1738` [4]
21. **[RFC1866]**: `https://www.rfc-editor.org/rfc/rfc1866` [1, 2, 4]
22. **[RFC2045]**: `https://www.rfc-editor.org/rfc/rfc2045` [2]
23. **[RFC2046]**: `https://www.rfc-editor.org/rfc/rfc2046` [3]
24. **[RFC2119]**: `https://www.rfc-editor.org/rfc/rfc2119` [1-3, 5]
25. **[RFC2396]**: `https://www.rfc-editor.org/rfc/rfc2396` [4]
26. **[RFC2557]**: `https://www.rfc-editor.org/rfc/rfc2557` [4]
27. **[RFC3339]**: `https://www.rfc-editor.org/rfc/rfc3339` [1, 5]
28. **[RFC3986]**: `https://www.rfc-editor.org/rfc/rfc3986` [1-3]
29. **[RFC3987]**: `https://www.rfc-editor.org/rfc/rfc3987` [3]
30. **[RFC4648]**: `https://www.rfc-editor.org/rfc/rfc4648` [2, 3]
31. **[RFC6265]**: `https://httpwg.org/specs/rfc6265.html` [3]
32. **[RFC6570]**: `https://www.rfc-editor.org/rfc/rfc6570` [1-3]
33. **[RFC6749]**: `https://www.rfc-editor.org/rfc/rfc6749` [1-3]
34. **[RFC6838]**: `https://www.rfc-editor.org/rfc/rfc6838` [1, 2, 4, 5]
35. **[RFC6901]**: `https://www.rfc-editor.org/rfc/rfc6901` [1-3, 5]
36. **[RFC7159]**: `https://www.rfc-editor.org/rfc/rfc7159` [1, 2, 5]
37. **[RFC7230]**: `https://httpwg.org/specs/rfc7230.html` [1, 2]
38. **[RFC7231]**: `https://httpwg.org/specs/rfc7231.html` [1, 2, 5]
39. **[RFC7235]**: `https://httpwg.org/specs/rfc7235.html` [1, 2]
40. **[RFC7464]**: `https://www.rfc-editor.org/rfc/rfc7464` [4]
41. **[RFC7578]**: `https://www.rfc-editor.org/rfc/rfc7578` [2, 3]
42. **[RFC8091]**: `https://www.rfc-editor.org/rfc/rfc8091` [4]
43. **[RFC8174]**: `https://www.rfc-editor.org/rfc/rfc8174` [1-3]
44. **[RFC8259]**: `https://www.rfc-editor.org/rfc/rfc8259` [3]
45. **[RFC8414]**: `https://www.rfc-editor.org/rfc/rfc8414` [3]
46. **[RFC8628]**: `https://www.rfc-editor.org/rfc/rfc8628` [3]
47. **[RFC9110]**: `https://httpwg.org/specs/rfc9110.html` [3]
48. **[RFC9264]**: `https://www.rfc-editor.org/rfc/rfc9264` [3]
49. **[RFC9512]**: `https://www.rfc-editor.org/rfc/rfc9512` [6]
50. **[RFC9535]**: `https://www.rfc-editor.org/rfc/rfc9535` [4]
51. **[SNIFF]**: `https://mimesniff.spec.whatwg.org/` [6]
52. **[SPDX-Licenses]**: `https://spdx.org/licenses/` [2, 6]
53. **[WHATWG-URL]**: `https://url.spec.whatwg.org/` [6]
54. **[YAML]**: `http://yaml.org/spec/1.2/spec.html` [1, 5-7]
55. **[xml-names11]**: `https://www.w3.org/TR/xml-names11/` [6]