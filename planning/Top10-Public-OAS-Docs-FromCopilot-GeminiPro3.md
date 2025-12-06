Here are the top 10 real-world OpenAPI Specification (OAS) documents, selected based on popularity, size, and diversity.

### 1. Stripe API
*   **Source URL**: [https://raw.githubusercontent.com/stripe/openapi/refs/heads/master/openapi/spec3.json](https://raw.githubusercontent.com/stripe/openapi/refs/heads/master/openapi/spec3.json)
*   **OAS Version**: 3.0.x
*   **Number of**:
    *   **Paths**: ~400+
    *   **Operations**: ~600+
    *   **Defined Schemas**: ~500+
*   **Document size**: ~3.5 MB
*   **Format**: JSON
*   **Description**: Extremely comprehensive financial services API. Known for its massive size and complex schema inheritance (polymorphism). It is a "gold standard" stress test for parsers due to its file size and extensive use of `anyOf`/`oneOf`.

### 2. GitHub REST API
*   **Source URL**: [https://raw.githubusercontent.com/github/rest-api-description/refs/heads/main/descriptions/api.github.com/api.github.com.yaml](https://raw.githubusercontent.com/github/rest-api-description/refs/heads/main/descriptions/api.github.com/api.github.com.yaml)
*   **OAS Version**: 3.0.x (and 3.1 features in newer versions)
*   **Number of**:
    *   **Paths**: ~500+
    *   **Operations**: ~900+
    *   **Defined Schemas**: ~600+
*   **Document size**: ~2 MB
*   **Format**: YAML
*   **Description**: Represents one of the most popular developer platforms. It is highly modular and uses extensive `$ref` linking. A great example of a large-scale API that has transitioned from legacy custom formats to standard OAS 3.0.

### 3. Kubernetes (K8s)
*   **Source URL**: [https://raw.githubusercontent.com/kubernetes/kubernetes/refs/heads/master/api/openapi-spec/swagger.json](https://raw.githubusercontent.com/kubernetes/kubernetes/refs/heads/master/api/openapi-spec/swagger.json)
*   **OAS Version**: 2.0 (Swagger)
*   **Number of**:
    *   **Paths**: ~1,000+
    *   **Operations**: ~1,500+
    *   **Defined Schemas**: ~600+
*   **Document size**: ~3+ MB
*   **Format**: JSON
*   **Description**: The heavyweight champion of Swagger 2.0 specs. It is famously deeply nested and verbose, often used to benchmark performance of code generators and validators because of its sheer volume of definitions.

### 4. Twilio
*   **Source URL**: [https://raw.githubusercontent.com/twilio/twilio-oai/refs/heads/main/spec/json/twilio_api_v2010.json](https://raw.githubusercontent.com/twilio/twilio-oai/refs/heads/main/spec/json/twilio_api_v2010.json)
*   **OAS Version**: 3.0.x
*   **Number of**:
    *   **Paths**: ~100+
    *   **Operations**: ~300+
    *   **Defined Schemas**: ~200+
*   **Document size**: ~500 KB - 1 MB (varies by product file)
*   **Format**: JSON
*   **Description**: Twilio splits their huge API into multiple specs. The "Api v2010" spec is a classic example of a mature telecommunications API using a flattened structure for complex resources like calls and messages.

### 5. Slack Web API
*   **Source URL**: [https://raw.githubusercontent.com/slackapi/slack-api-specs/master/web-api/slack_web_openapi_v2.json](https://raw.githubusercontent.com/slackapi/slack-api-specs/master/web-api/slack_web_openapi_v2.json)
*   **OAS Version**: 3.0.x
*   **Number of**:
    *   **Paths**: ~230+
    *   **Operations**: ~230+
    *   **Defined Schemas**: ~100+
*   **Document size**: ~600 KB
*   **Format**: YAML
*   **Description**: A community-maintained but widely used spec. It is unique because it models an RPC-style API (mostly POST requests to unique endpoints) within the REST-centric OpenAPI format.

### 6. Plaid
*   **Source URL**: [https://raw.githubusercontent.com/plaid/plaid-openapi/refs/heads/master/2020-09-14.yml](https://raw.githubusercontent.com/plaid/plaid-openapi/refs/heads/master/2020-09-14.yml)
*   **OAS Version**: 3.0.x
*   **Number of**:
    *   **Paths**: ~50+
    *   **Operations**: ~100+
    *   **Defined Schemas**: ~150+
*   **Document size**: ~400 KB
*   **Format**: JSON
*   **Description**: A modern Fintech API. It is very clean and strictly typed, making it an excellent test case for schema validation and strict type generation logic.

### 7. AWS S3 (Community Example)
*   **Source URL**: [https://github.com/aws-samples/aws-s3-openapi/blob/main/aws-s3-openapi.yaml](https://github.com/aws-samples/aws-s3-openapi/blob/main/aws-s3-openapi.yaml)
    * _NOTE: The URL above returns a 404, but I found a docs page on AWS that includes the full spec:_
      * https://docs.aws.amazon.com/apigateway/latest/developerguide/api-as-s3-proxy-export-swagger-with-extensions.html
*   **OAS Version**: 3.0.x
*   **Number of**:
    *   **Paths**: ~60+
    *   **Operations**: ~150+
    *   **Defined Schemas**: ~100+
*   **Document size**: ~120 KB
*   **Format**: YAML
*   **Description**: While AWS uses Smithy internally, this OAS representation of S3 is crucial for testing integration with object storage services. It includes extensive XML schema mapping, which is rare in modern JSON-first APIs.

### 8. DigitalOcean
*   **Source URL**: https://raw.githubusercontent.com/digitalocean/openapi/refs/heads/main/specification/DigitalOcean-public.v2.yaml
*   **OAS Version**: 3.0.x
*   **Number of**:
    *   **Paths**: ~150+
    *   **Operations**: ~250+
    *   **Defined Schemas**: ~150+
*   **Document size**: ~1 MB
*   **Format**: YAML
*   **Description**: A clean, resource-oriented cloud provider API. It is well-structured and uses standard HTTP status codes and methods effectively, making it a "textbook" example of RESTful OAS 3 design.

### 9. Uber API
*   **Source URL**: [https://github.com/uber/api-example/blob/master/swagger/uber.json](https://github.com/uber/api-example/blob/master/swagger/uber.json)
    - _Note: The link above returns a 404 and I could not find the spec online_
*   **OAS Version**: 2.0 (Swagger)
*   **Number of**:
    *   **Paths**: ~5
    *   **Operations**: ~5
    *   **Defined Schemas**: ~20
*   **Document size**: ~40 KB
*   **Format**: JSON
*   **Description**: A concise Swagger 2.0 example often used for tutorials. Unlike the massive Kubernetes spec, this is a small, digestible 2.0 file perfect for testing legacy support without parsing megabytes of JSON.

### 10. Swagger Petstore (Reference)
*   **Source URL**: [https://raw.githubusercontent.com/swagger-api/swagger-petstore/refs/heads/master/src/main/resources/openapi.yaml](https://raw.githubusercontent.com/swagger-api/swagger-petstore/refs/heads/master/src/main/resources/openapi.yaml)
*   **OAS Version**: 3.0.4
*   **Number of**:
    *   **Paths**: 13
    *   **Operations**: 19
    *   **Defined Schemas**: 6
*   **Document size**: ~25 KB
*   **Format**: YAML
*   **Description**: The official reference implementation for OpenAPI 3. It contains examples of authentication (OAuth2, API Key), file uploads, and common CRUD operations. It is the baseline for functional correctness.