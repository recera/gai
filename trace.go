package gai

// TraceFunc is a function type for receiving trace information about LLM calls
type TraceFunc func(TraceInfo)