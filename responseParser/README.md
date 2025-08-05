# Response Parser: Robust JSON Parser for LLM Responses

The `responseParser` package is designed to reliably parse JSON from Large Language Model (LLM) responses, handling the common edge cases and imperfections that frequently occur in LLM-generated JSON. This implementation provides maximum reliability when working with structured data from LLMs.

## Features

- **Preprocessing**: Removes markdown formatting, code blocks, and extracts JSON from mixed text
- **Robust Parsing**: Handles common LLM JSON errors like:
  - Single quotes instead of double quotes
  - Comments within JSON
  - Trailing commas
  - Unquoted keys
  - Unbalanced braces and brackets
  - JavaScript arrow syntax (`=>`)
  - Mixed-case booleans (`TRUE`, `False`)
- **Type Coercion**: Intelligently converts between compatible types (string→int, string→float, etc.)
- **Case Insensitive**: Supports field name matching regardless of case
- **Snake Case Support**: Maps between snake_case and camelCase field names
- **Time Parsing**: Automatically handles various time/date formats

## Architecture

The package consists of three main components:

1. **Cleanup** (`cleanup/`): Preprocessing LLM responses to handle markdown and extract JSON
2. **Parser** (`parser/`): Converting JSON-like content to canonical JSON
3. **Coercer** (`coercer/`): Mapping between JSON values and Go struct fields

Each component can be used independently or as part of the main parsing pipeline.

## Usage

The responseParser is integrated into the main gai package and is used automatically when calling `GetResponseObject`:

```go
type Person struct {
    Name  string   `json:"name"`
    Age   int      `json:"age"`
    Skills []string `json:"skills"`
}

// The responseParser handles all the complexity automatically
var person Person
err := client.GetResponseObject(ctx, parts, &person)
```

## Configuration Options

The package provides flexible configuration through `ParseOptions`:

```go
// Default options - maximum robustness
ParseInto(llmResponse, &result, DefaultOptions())

// Strict parsing - only valid JSON
ParseInto(llmResponse, &result, StrictOptions())

// Custom options
ParseInto(llmResponse, &result, ParseOptions{
    Relaxed:       true,   // Allow non-standard JSON syntax
    Extract:       true,   // Extract JSON from text/markdown
    Autocomplete:  true,   // Fix unbalanced delimiters
    FixFormat:     true,   // Fix common LLM formatting issues
    AllowCoercion: true,   // Allow type coercion between compatible types
})
```

## Why responseParser?

Standard JSON parsers are designed for machine-generated, perfectly formatted JSON. However, LLMs often produce JSON with various imperfections:

1. **Markdown Formatting**: LLMs frequently wrap JSON in markdown code blocks
2. **JavaScript Habits**: Single quotes, trailing commas, comments, etc.
3. **Incomplete Output**: Truncated or unbalanced JSON structures
4. **Type Inconsistencies**: Strings instead of numbers, mixed case booleans, etc.
5. **Mixed Content**: JSON embedded within explanatory text

The responseParser addresses all these issues with a multi-layered approach to ensure you can reliably extract structured data from LLM responses.