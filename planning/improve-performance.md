## Performance Concerns ⚠️

**Issue**: The custom marshalers use a double-marshal approach that has performance overhead:

```go
// Marshal struct to JSON
aux, err := json.Marshal((*Alias)(i))
// Unmarshal JSON back to map
var m map[string]interface{}
json.Unmarshal(aux, &m)
// Merge Extra fields
for k, v := range i.Extra {
    m[k] = v
}
// Marshal final map to JSON
return json.Marshal(m)
```

**Impact**:
- Each struct with Extra fields requires 2 marshal operations + 1 unmarshal operation
- For deeply nested documents, this compounds significantly
- Profiling would help quantify the actual performance impact

**Recommendation**:
- Consider documenting the performance tradeoff in godoc comments
- For v1.6.0+, explore more efficient implementations (e.g., manual field serialization)