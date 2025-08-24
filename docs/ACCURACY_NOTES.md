# Documentation Accuracy Notes

## Corrections Made

After reviewing the codebase against the documentation, the following corrections have been made to ensure accuracy:

### 1. Import Paths ✅ FIXED
- **Changed from**: `github.com/yourusername/gai`
- **Changed to**: `github.com/recera/gai`
- Applied across all documentation files

### 2. Error Function Names ✅ FIXED
The following error checking functions were corrected to match the actual implementation:
- `IsContextLengthExceeded()` → `IsContextSizeExceeded()`
- `IsInvalidRequest()` → `IsBadRequest()`
- `IsUnauthorized()` → `IsAuth()`
- `IsProviderUnavailable()` → `IsOverloaded()`

### 3. Anthropic API Version ✅ FIXED
- Updated default version from `2024-02-15` to `2023-06-01` to match actual code

### 4. Removed Non-Existent Features ✅ FIXED
- Removed references to `WithBetaFeatures()` in Anthropic provider (doesn't exist)
- Removed `WithAzureDeployment()` and `WithAPIVersion()` for OpenAI (use `WithBaseURL()` instead)
- Updated Azure OpenAI configuration example to use correct approach

## Verified Accurate Features

The following features were verified to be accurately documented:

### Core Package ✅
- Message types and roles (System, User, Assistant, Tool)
- Part types (Text, ImageURL, Audio, Video, File)
- BlobRef system with three kinds (URL, Bytes, ProviderFile)
- Event types and streaming interfaces
- ToolChoice constants including ToolAuto

### Provider Options ✅
- OpenAI: `WithAPIKey()`, `WithModel()`, `WithOrganization()`, `WithProject()`, `WithBaseURL()`
- Anthropic: `WithAPIKey()`, `WithModel()`, `WithVersion()`, `WithBaseURL()`
- Common: `WithTimeout()`, `WithMaxRetries()`, `WithRetryDelay()`, `WithHTTPClient()`

### Error Handling Functions ✅
All of these exist and are documented correctly:
- `IsTransient()` - For retryable errors
- `IsRateLimited()` - For rate limiting
- `IsAuth()` - For authentication/authorization errors
- `IsBadRequest()` - For invalid requests
- `IsNotFound()` - For missing resources
- `IsSafetyBlocked()` - For content filtering
- `IsTimeout()` - For timeout errors
- `IsNetwork()` - For network errors
- `IsOverloaded()` - For service overload
- `IsContextSizeExceeded()` - For context length issues
- `GetRetryAfter()` - To get retry delay

### Streaming ✅
- SSE (Server-Sent Events) support
- NDJSON support
- Channel-based event streaming
- Proper Close() methods

### Tool Calling ✅
- Tools package with generic Tool[I,O] type
- Schema generation from Go types
- Multi-step execution support
- Parallel tool execution

## Documentation Status

The documentation is now accurate with respect to:
- ✅ Import paths and module names
- ✅ Available provider options and configuration methods
- ✅ Error handling functions and patterns
- ✅ Core types and interfaces
- ✅ Streaming capabilities
- ✅ Tool calling system
- ✅ API versions and defaults

## Notes for Future Updates

When updating documentation, verify:
1. Function and method names exist in the codebase
2. Import paths match go.mod
3. Default values match constants in code
4. Error function names match core/errors.go
5. Provider-specific options match actual Option functions