# Evaluation & Datasets

Use the Recorder middleware to log interactions to NDJSON, then build datasets.

## Record

```go
rec, _ := eval.NewRecorder("eval_logs.ndjson")
prov := rec.Wrap(baseProvider)
// Use prov via client wiring
```

## Build dataset

```go
_ = eval.BuildDataset("eval_logs.ndjson", "dataset.json", func(m map[string]any) *eval.Entry {
  return &eval.Entry{
    Provider: m["provider"].(string),
    Model:    m["model"].(string),
    Messages: nil, // or transform from m["messages"] if desired
    Response: m["response"].(string),
    Expected: m["expected_text"], // or expected_json
    Metadata: nil,
  }
})
```

Upload the dataset to your evaluation platform (e.g., Braintrust/Arize) and attach scorers.

