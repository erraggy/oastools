---
name: generate-code
description: Generate Go client, server, or type code from an OpenAPI spec
---

# Generate Code from an OpenAPI Specification

## Step 1: Understand the spec

Call `parse` to get an overview of the API:

```json
{"spec": {"file": "<path>"}}
```

Report the API title, version, number of operations and schemas. This helps set expectations for what will be generated.

## Step 2: Determine what to generate

Ask the user what they need. The `generate` tool supports three output modes:

| Option | What it generates |
|--------|------------------|
| `types` | Go type definitions for all schemas |
| `client` | HTTP client with methods for each operation (includes types) |
| `server` | Server interface and handler stubs (includes types) |

Also determine:
- `output_dir` (required) -- Where to write the generated files
- `package_name` (optional, default: `api`) -- Go package name for the generated code

## Step 3: Generate the code

Call the `generate` tool:

```json
{
  "spec": {"file": "<path>"},
  "client": true,
  "output_dir": "./generated",
  "package_name": "petstore"
}
```

## Step 4: Review the output

The tool returns a manifest of generated files with names and sizes. Review:

- **File list** -- Confirm all expected files were created
- **Generated types count** -- Matches the schema count from Step 1
- **Generated operations count** -- Matches the operation count from Step 1
- **Warnings** -- Any issues encountered during generation (unsupported features, naming conflicts)
- **Critical errors** -- If `success` is false, explain what went wrong

Read the generated files and provide a summary of the key types and functions.

## Step 5: Suggest integration steps

Based on what was generated, suggest next steps:

**For types only:**
- Import the package and use the types in existing code
- Note any custom types that may need manual adjustment (e.g., `interface{}` fields)

**For client:**
- Show how to initialize the client with a base URL
- Demonstrate calling one or two key endpoints
- Note authentication setup if the spec has security schemes

**For server:**
- Show how to implement the generated interface
- Demonstrate registering handlers with a router
- Note any middleware needed (auth, CORS, etc.)
