# Building Production-Ready AI Applications

This comprehensive guide covers best practices, patterns, and techniques for building robust, scalable AI applications using GAI in production environments.

## Table of Contents
- [Production Architecture](#production-architecture)
- [Configuration Management](#configuration-management)
- [Error Handling and Resilience](#error-handling-and-resilience)
- [Performance Optimization](#performance-optimization)
- [Security Best Practices](#security-best-practices)
- [Monitoring and Observability](#monitoring-and-observability)
- [Testing Strategies](#testing-strategies)
- [Deployment Patterns](#deployment-patterns)
- [Cost Optimization](#cost-optimization)
- [Scaling Considerations](#scaling-considerations)

## Production Architecture

### Layered Architecture Pattern

Structure your AI application in clear layers for maintainability and testability:

```go
// Application layer - business logic
type AIService struct {
    providers   map[string]core.Provider
    tools       map[string]tools.Handle
    config      *Config
    metrics     MetricsCollector
    logger      Logger
}

// Infrastructure layer - provider management
type ProviderManager struct {
    primary     core.Provider
    fallbacks   []core.Provider
    router      *ProviderRouter
    circuitBreaker *CircuitBreaker
}

// Domain layer - business entities
type ConversationManager struct {
    store       ConversationStore
    validator   MessageValidator
    enricher    ContextEnricher
}
```

### Provider Abstraction with Fallbacks

Implement robust provider management with automatic failover:

```go
type RobustProviderManager struct {
    providers     map[string]core.Provider
    primaryName   string
    fallbackOrder []string
    healthChecker *HealthChecker
    metrics       *ProviderMetrics
    circuitBreakers map[string]*CircuitBreaker
}

func NewRobustProviderManager(config ProviderConfig) *RobustProviderManager {
    pm := &RobustProviderManager{
        providers:       make(map[string]core.Provider),
        circuitBreakers: make(map[string]*CircuitBreaker),
        healthChecker:   NewHealthChecker(),
        metrics:        NewProviderMetrics(),
    }
    
    // Initialize providers with middleware
    pm.initializeProviders(config)
    
    return pm
}

func (pm *RobustProviderManager) initializeProviders(config ProviderConfig) {
    // Primary provider (e.g., OpenAI)
    pm.providers["openai"] = middleware.Chain(
        middleware.WithRetry(middleware.RetryOpts{
            MaxAttempts:  3,
            InitialDelay: time.Second,
            MaxDelay:     30 * time.Second,
        }),
        middleware.WithRateLimit(middleware.RateLimitOpts{
            RequestsPerSecond: 10,
            BurstSize:        20,
        }),
        middleware.WithObservability(middleware.ObservabilityOpts{
            MetricsCollector: pm.metrics,
            TracingEnabled:   true,
        }),
    )(openai.New(
        openai.WithAPIKey(config.OpenAI.APIKey),
        openai.WithModel(config.OpenAI.Model),
        openai.WithTimeout(30 * time.Second),
    ))
    
    // Fallback providers
    pm.providers["anthropic"] = middleware.Chain(
        middleware.WithRetry(defaultRetryOpts()),
        middleware.WithObservability(observabilityOpts(pm.metrics)),
    )(anthropic.New(
        anthropic.WithAPIKey(config.Anthropic.APIKey),
    ))
    
    pm.providers["groq"] = groq.New(
        groq.WithAPIKey(config.Groq.APIKey),
        groq.WithModel(groq.Llama318BInstant), // Fast fallback
    )
    
    // Circuit breakers for each provider
    for name := range pm.providers {
        pm.circuitBreakers[name] = NewCircuitBreaker(CircuitBreakerOpts{
            FailureThreshold: 5,
            ResetTimeout:     30 * time.Second,
        })
    }
}

func (pm *RobustProviderManager) GenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {
    // Try primary provider first
    if result, err := pm.tryProvider(ctx, pm.primaryName, req); err == nil {
        return result, nil
    }
    
    // Try fallbacks in order
    for _, providerName := range pm.fallbackOrder {
        if result, err := pm.tryProvider(ctx, providerName, req); err == nil {
            pm.metrics.RecordFallbackUsed(providerName)
            return result, nil
        }
    }
    
    return nil, fmt.Errorf("all providers failed")
}

func (pm *RobustProviderManager) tryProvider(ctx context.Context, name string, req core.Request) (*core.TextResult, error) {
    provider := pm.providers[name]
    cb := pm.circuitBreakers[name]
    
    return cb.Execute(func() (*core.TextResult, error) {
        return provider.GenerateText(ctx, req)
    })
}
```

### Service Layer Implementation

Create a robust service layer that handles business logic:

```go
type AIService struct {
    providerManager *RobustProviderManager
    toolRegistry    *ToolRegistry
    config          *ServiceConfig
    logger          Logger
    metrics         MetricsCollector
    validator       RequestValidator
}

func (s *AIService) ProcessRequest(ctx context.Context, userRequest UserRequest) (*ServiceResponse, error) {
    // Add request tracing
    ctx = s.addTracing(ctx, userRequest.ID)
    
    // Validate request
    if err := s.validator.Validate(userRequest); err != nil {
        s.metrics.RecordValidationError(err)
        return nil, fmt.Errorf("validation failed: %w", err)
    }
    
    // Convert to core request
    coreReq, err := s.convertRequest(userRequest)
    if err != nil {
        return nil, fmt.Errorf("request conversion failed: %w", err)
    }
    
    // Add tools if requested
    if userRequest.EnableTools {
        coreReq.Tools = s.toolRegistry.GetToolsForUser(userRequest.UserID)
        coreReq.StopWhen = core.CombineConditions(
            core.MaxSteps(s.config.MaxSteps),
            core.NoMoreTools(),
            s.createCostLimitCondition(userRequest.UserID),
        )
    }
    
    // Execute with timeout
    execCtx, cancel := context.WithTimeout(ctx, s.config.RequestTimeout)
    defer cancel()
    
    result, err := s.providerManager.GenerateText(execCtx, *coreReq)
    if err != nil {
        s.metrics.RecordRequestError(err)
        return nil, s.handleError(err)
    }
    
    // Process and validate response
    response, err := s.processResponse(result, userRequest)
    if err != nil {
        return nil, fmt.Errorf("response processing failed: %w", err)
    }
    
    s.metrics.RecordSuccessfulRequest(result.Usage.TotalTokens)
    return response, nil
}

func (s *AIService) createCostLimitCondition(userID string) core.StopCondition {
    return func(stepNum int, step core.Step) bool {
        cost := s.estimateStepCost(step)
        userBudget := s.getUserBudget(userID)
        
        if cost > userBudget {
            s.logger.Warn("Stopping execution due to cost limit",
                "user_id", userID,
                "cost", cost,
                "budget", userBudget)
            return true
        }
        
        return false
    }
}
```

## Configuration Management

### Environment-Based Configuration

Implement robust configuration management:

```go
type Config struct {
    Environment string `env:"ENVIRONMENT" default:"development"`
    
    // Provider configurations
    OpenAI    OpenAIConfig    `json:"openai"`
    Anthropic AnthropicConfig `json:"anthropic"`
    Groq      GroqConfig      `json:"groq"`
    
    // Service configuration
    Server    ServerConfig    `json:"server"`
    Database  DatabaseConfig  `json:"database"`
    Redis     RedisConfig     `json:"redis"`
    
    // Feature flags
    Features  FeatureConfig   `json:"features"`
    
    // Observability
    Metrics   MetricsConfig   `json:"metrics"`
    Logging   LoggingConfig   `json:"logging"`
    Tracing   TracingConfig   `json:"tracing"`
}

type OpenAIConfig struct {
    APIKey           string        `env:"OPENAI_API_KEY" required:"true"`
    Model            string        `env:"OPENAI_MODEL" default:"gpt-4o-mini"`
    BaseURL          string        `env:"OPENAI_BASE_URL"`
    Organization     string        `env:"OPENAI_ORGANIZATION"`
    MaxRetries       int           `env:"OPENAI_MAX_RETRIES" default:"3"`
    Timeout          time.Duration `env:"OPENAI_TIMEOUT" default:"60s"`
    RateLimitRPS     float64       `env:"OPENAI_RATE_LIMIT_RPS" default:"10"`
}

type FeatureConfig struct {
    EnableTools      bool `env:"FEATURE_ENABLE_TOOLS" default:"true"`
    EnableStreaming  bool `env:"FEATURE_ENABLE_STREAMING" default:"true"`
    EnableMultiStep  bool `env:"FEATURE_ENABLE_MULTI_STEP" default:"true"`
    MaxStepsPerUser  int  `env:"FEATURE_MAX_STEPS_PER_USER" default:"20"`
}

// Load configuration from environment and files
func LoadConfig() (*Config, error) {
    config := &Config{}
    
    // Load from environment variables
    if err := env.Parse(config); err != nil {
        return nil, fmt.Errorf("failed to parse environment: %w", err)
    }
    
    // Load from config file if exists
    if configPath := os.Getenv("CONFIG_PATH"); configPath != "" {
        if err := loadConfigFile(config, configPath); err != nil {
            return nil, fmt.Errorf("failed to load config file: %w", err)
        }
    }
    
    // Validate configuration
    if err := config.Validate(); err != nil {
        return nil, fmt.Errorf("config validation failed: %w", err)
    }
    
    return config, nil
}

func (c *Config) Validate() error {
    // Validate provider configurations
    if c.OpenAI.APIKey == "" && c.Anthropic.APIKey == "" {
        return fmt.Errorf("at least one provider API key must be configured")
    }
    
    // Validate rate limits
    if c.OpenAI.RateLimitRPS <= 0 {
        return fmt.Errorf("rate limit must be positive")
    }
    
    // Validate timeouts
    if c.OpenAI.Timeout <= 0 {
        return fmt.Errorf("timeout must be positive")
    }
    
    return nil
}
```

### Configuration Hot Reloading

Implement configuration hot reloading for production environments:

```go
type ConfigManager struct {
    config       *Config
    subscribers  []chan *Config
    mutex        sync.RWMutex
    watcher      *fsnotify.Watcher
    logger       Logger
}

func NewConfigManager(initialConfig *Config) (*ConfigManager, error) {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return nil, fmt.Errorf("failed to create file watcher: %w", err)
    }
    
    cm := &ConfigManager{
        config:  initialConfig,
        watcher: watcher,
        logger:  log.New(os.Stdout, "[ConfigManager] ", log.LstdFlags),
    }
    
    go cm.watchConfigFile()
    
    return cm, nil
}

func (cm *ConfigManager) GetConfig() *Config {
    cm.mutex.RLock()
    defer cm.mutex.RUnlock()
    return cm.config
}

func (cm *ConfigManager) Subscribe() <-chan *Config {
    cm.mutex.Lock()
    defer cm.mutex.Unlock()
    
    ch := make(chan *Config, 1)
    cm.subscribers = append(cm.subscribers, ch)
    
    // Send current config immediately
    ch <- cm.config
    
    return ch
}

func (cm *ConfigManager) watchConfigFile() {
    configPath := os.Getenv("CONFIG_PATH")
    if configPath == "" {
        return // No config file to watch
    }
    
    if err := cm.watcher.Add(configPath); err != nil {
        cm.logger.Printf("Failed to watch config file: %v", err)
        return
    }
    
    for {
        select {
        case event := <-cm.watcher.Events:
            if event.Op&fsnotify.Write == fsnotify.Write {
                cm.logger.Printf("Config file modified: %s", event.Name)
                cm.reloadConfig(configPath)
            }
            
        case err := <-cm.watcher.Errors:
            cm.logger.Printf("Config watcher error: %v", err)
        }
    }
}

func (cm *ConfigManager) reloadConfig(configPath string) {
    newConfig := &Config{}
    
    // Parse environment variables
    if err := env.Parse(newConfig); err != nil {
        cm.logger.Printf("Failed to parse environment during reload: %v", err)
        return
    }
    
    // Load from file
    if err := loadConfigFile(newConfig, configPath); err != nil {
        cm.logger.Printf("Failed to reload config file: %v", err)
        return
    }
    
    // Validate new configuration
    if err := newConfig.Validate(); err != nil {
        cm.logger.Printf("New config validation failed: %v", err)
        return
    }
    
    // Update configuration
    cm.mutex.Lock()
    oldConfig := cm.config
    cm.config = newConfig
    
    // Notify subscribers
    for _, ch := range cm.subscribers {
        select {
        case ch <- newConfig:
        default:
            // Channel full, skip
        }
    }
    cm.mutex.Unlock()
    
    cm.logger.Printf("Configuration reloaded successfully")
    
    // Log configuration changes
    cm.logConfigChanges(oldConfig, newConfig)
}
```

## Error Handling and Resilience

### Comprehensive Error Handling

Implement structured error handling throughout your application:

```go
// Custom error types for different scenarios
type ServiceError struct {
    Code       string         `json:"code"`
    Message    string         `json:"message"`
    UserFacing bool          `json:"user_facing"`
    Retryable  bool          `json:"retryable"`
    Context    map[string]any `json:"context,omitempty"`
    Cause      error         `json:"-"`
}

func (e *ServiceError) Error() string {
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *ServiceError) Unwrap() error {
    return e.Cause
}

// Error handling middleware
type ErrorHandler struct {
    logger Logger
    metrics MetricsCollector
}

func (eh *ErrorHandler) HandleError(ctx context.Context, err error) *ServiceError {
    // Extract user ID from context for logging
    userID := getUserID(ctx)
    
    // Log the error with context
    eh.logger.Error("Service error occurred",
        "error", err,
        "user_id", userID,
        "trace_id", getTraceID(ctx))
    
    // Record metrics
    eh.metrics.RecordError(err)
    
    // Convert to service error based on error type
    switch {
    case errors.Is(err, context.DeadlineExceeded):
        return &ServiceError{
            Code:       "TIMEOUT",
            Message:    "Request timed out",
            UserFacing: true,
            Retryable:  true,
        }
        
    case isRateLimitError(err):
        return &ServiceError{
            Code:       "RATE_LIMIT",
            Message:    "Rate limit exceeded, please try again later",
            UserFacing: true,
            Retryable:  true,
            Context:    map[string]any{"retry_after": getRateLimitRetryAfter(err)},
        }
        
    case isQuotaError(err):
        return &ServiceError{
            Code:       "QUOTA_EXCEEDED",
            Message:    "Usage quota exceeded",
            UserFacing: true,
            Retryable:  false,
        }
        
    case isValidationError(err):
        return &ServiceError{
            Code:       "VALIDATION_ERROR",
            Message:    "Invalid request parameters",
            UserFacing: true,
            Retryable:  false,
            Context:    extractValidationDetails(err),
        }
        
    default:
        // Don't expose internal errors to users
        return &ServiceError{
            Code:       "INTERNAL_ERROR",
            Message:    "An internal error occurred",
            UserFacing: true,
            Retryable:  true,
            Cause:      err,
        }
    }
}
```

### Circuit Breaker Implementation

Implement circuit breaker pattern for provider reliability:

```go
type CircuitBreakerState int

const (
    StateClosed CircuitBreakerState = iota
    StateOpen
    StateHalfOpen
)

type CircuitBreaker struct {
    failureThreshold int
    resetTimeout     time.Duration
    onStateChange    func(CircuitBreakerState)
    
    mutex           sync.RWMutex
    state           CircuitBreakerState
    failures        int
    lastFailureTime time.Time
    requests        int
    successes       int
}

type CircuitBreakerOpts struct {
    FailureThreshold int
    ResetTimeout     time.Duration
    OnStateChange    func(CircuitBreakerState)
}

func NewCircuitBreaker(opts CircuitBreakerOpts) *CircuitBreaker {
    return &CircuitBreaker{
        failureThreshold: opts.FailureThreshold,
        resetTimeout:     opts.ResetTimeout,
        onStateChange:    opts.OnStateChange,
        state:           StateClosed,
    }
}

func (cb *CircuitBreaker) Execute(fn func() (*core.TextResult, error)) (*core.TextResult, error) {
    if !cb.allowRequest() {
        return nil, fmt.Errorf("circuit breaker is open")
    }
    
    result, err := fn()
    cb.recordResult(err == nil)
    
    return result, err
}

func (cb *CircuitBreaker) allowRequest() bool {
    cb.mutex.RLock()
    defer cb.mutex.RUnlock()
    
    switch cb.state {
    case StateClosed:
        return true
    case StateOpen:
        return time.Since(cb.lastFailureTime) >= cb.resetTimeout
    case StateHalfOpen:
        return cb.requests < 5 // Allow limited requests to test
    }
    
    return false
}

func (cb *CircuitBreaker) recordResult(success bool) {
    cb.mutex.Lock()
    defer cb.mutex.Unlock()
    
    cb.requests++
    
    if success {
        cb.successes++
        
        if cb.state == StateHalfOpen && cb.successes >= 3 {
            // Enough successes in half-open state, close the circuit
            cb.changeState(StateClosed)
            cb.reset()
        }
    } else {
        cb.failures++
        cb.lastFailureTime = time.Now()
        
        if cb.state == StateClosed && cb.failures >= cb.failureThreshold {
            cb.changeState(StateOpen)
        } else if cb.state == StateHalfOpen {
            cb.changeState(StateOpen)
        }
    }
}

func (cb *CircuitBreaker) changeState(newState CircuitBreakerState) {
    if cb.state != newState {
        cb.state = newState
        if cb.onStateChange != nil {
            cb.onStateChange(newState)
        }
    }
}
```

### Graceful Degradation

Implement graceful degradation strategies:

```go
type DegradationManager struct {
    config     *Config
    healthCheck *HealthChecker
    features   map[string]bool
    mutex      sync.RWMutex
}

func (dm *DegradationManager) ProcessRequest(ctx context.Context, req *ServiceRequest) (*ServiceResponse, error) {
    // Check system health and adjust request accordingly
    healthScore := dm.healthCheck.GetCurrentScore()
    
    switch {
    case healthScore >= 0.9:
        // System healthy - full features
        return dm.processFullFeatureRequest(ctx, req)
        
    case healthScore >= 0.7:
        // Moderate load - reduce complex features
        req = dm.simplifyRequest(req)
        return dm.processSimplifiedRequest(ctx, req)
        
    case healthScore >= 0.5:
        // High load - essential features only
        req = dm.essentialFeaturesOnly(req)
        return dm.processEssentialRequest(ctx, req)
        
    default:
        // Critical load - emergency mode
        return dm.processEmergencyRequest(ctx, req)
    }
}

func (dm *DegradationManager) simplifyRequest(req *ServiceRequest) *ServiceRequest {
    // Reduce tool availability
    req.EnableTools = false
    
    // Reduce max tokens
    if req.MaxTokens > 500 {
        req.MaxTokens = 500
    }
    
    // Use faster, cheaper models
    req.PreferredProvider = "groq" // Fast LPU inference
    req.Model = "llama-3.1-8b-instant"
    
    return req
}

func (dm *DegradationManager) essentialFeaturesOnly(req *ServiceRequest) *ServiceRequest {
    // Disable all advanced features
    req.EnableTools = false
    req.EnableStreaming = false
    req.MaxTokens = 200
    
    // Use the fastest model
    req.PreferredProvider = "groq"
    req.Model = "llama-3.1-8b-instant"
    
    return req
}

func (dm *DegradationManager) processEmergencyRequest(ctx context.Context, req *ServiceRequest) (*ServiceResponse, error) {
    // Return cached response if available
    if cached := dm.getCachedResponse(req); cached != nil {
        cached.Source = "cache"
        cached.Warning = "System is under high load, returning cached response"
        return cached, nil
    }
    
    // Return predefined response for common queries
    if predefined := dm.getPredefinedResponse(req); predefined != nil {
        predefined.Source = "predefined"
        predefined.Warning = "System is under high load, returning predefined response"
        return predefined, nil
    }
    
    // Last resort: error with helpful message
    return nil, &ServiceError{
        Code:       "SERVICE_DEGRADED",
        Message:    "Service is temporarily degraded, please try again later",
        UserFacing: true,
        Retryable:  true,
        Context: map[string]any{
            "retry_after": 30,
            "health_score": dm.healthCheck.GetCurrentScore(),
        },
    }
}
```

## Performance Optimization

### Connection Pooling and Reuse

Optimize HTTP connections for better performance:

```go
type OptimizedHTTPClient struct {
    client *http.Client
    pool   sync.Pool
}

func NewOptimizedHTTPClient() *OptimizedHTTPClient {
    transport := &http.Transport{
        // Connection pooling
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 20,
        IdleConnTimeout:     90 * time.Second,
        
        // Timeouts
        TLSHandshakeTimeout:   10 * time.Second,
        ResponseHeaderTimeout: 30 * time.Second,
        ExpectContinueTimeout: 1 * time.Second,
        
        // Keep-alive
        DisableKeepAlives: false,
        
        // Compression
        DisableCompression: false,
    }
    
    client := &http.Client{
        Transport: transport,
        Timeout:   60 * time.Second,
    }
    
    return &OptimizedHTTPClient{
        client: client,
        pool: sync.Pool{
            New: func() any {
                return make([]byte, 0, 4096) // 4KB buffer
            },
        },
    }
}

func (ohc *OptimizedHTTPClient) Do(req *http.Request) (*http.Response, error) {
    // Add standard headers for better caching/compression
    if req.Header.Get("Accept-Encoding") == "" {
        req.Header.Set("Accept-Encoding", "gzip, deflate")
    }
    
    if req.Header.Get("Connection") == "" {
        req.Header.Set("Connection", "keep-alive")
    }
    
    return ohc.client.Do(req)
}
```

### Caching Strategies

Implement intelligent caching for AI responses:

```go
type ResponseCache struct {
    redis      redis.Client
    local      *lru.Cache
    hasher     hash.Hash
    config     CacheConfig
    metrics    CacheMetrics
}

type CacheConfig struct {
    TTL              time.Duration `json:"ttl"`
    MaxSize          int          `json:"max_size"`
    EnableRedis      bool         `json:"enable_redis"`
    EnableLocal      bool         `json:"enable_local"`
    CompressionLevel int          `json:"compression_level"`
}

func (rc *ResponseCache) Get(ctx context.Context, req *core.Request) (*core.TextResult, bool) {
    cacheKey := rc.generateCacheKey(req)
    
    // Try local cache first (fastest)
    if rc.config.EnableLocal {
        if result, found := rc.local.Get(cacheKey); found {
            rc.metrics.RecordHit("local")
            return result.(*core.TextResult), true
        }
    }
    
    // Try Redis cache (shared across instances)
    if rc.config.EnableRedis {
        if result, err := rc.getFromRedis(ctx, cacheKey); err == nil {
            rc.metrics.RecordHit("redis")
            
            // Also cache locally for next time
            if rc.config.EnableLocal {
                rc.local.Add(cacheKey, result)
            }
            
            return result, true
        }
    }
    
    rc.metrics.RecordMiss()
    return nil, false
}

func (rc *ResponseCache) Set(ctx context.Context, req *core.Request, result *core.TextResult) error {
    cacheKey := rc.generateCacheKey(req)
    
    // Cache locally
    if rc.config.EnableLocal {
        rc.local.Add(cacheKey, result)
    }
    
    // Cache in Redis
    if rc.config.EnableRedis {
        return rc.setInRedis(ctx, cacheKey, result)
    }
    
    return nil
}

func (rc *ResponseCache) generateCacheKey(req *core.Request) string {
    rc.hasher.Reset()
    
    // Include relevant request parameters in cache key
    keyData := struct {
        Model       string                 `json:"model"`
        Messages    []core.Message         `json:"messages"`
        Temperature float32                `json:"temperature"`
        MaxTokens   int                   `json:"max_tokens"`
        Tools       []string              `json:"tools"`
        Options     map[string]any        `json:"options"`
    }{
        Model:       req.Model,
        Messages:    req.Messages,
        Temperature: req.Temperature,
        MaxTokens:   req.MaxTokens,
        Options:     req.ProviderOptions,
    }
    
    // Add tool names only (not implementations)
    for _, tool := range req.Tools {
        keyData.Tools = append(keyData.Tools, tool.Name())
    }
    
    jsonData, _ := json.Marshal(keyData)
    rc.hasher.Write(jsonData)
    
    return fmt.Sprintf("%x", rc.hasher.Sum(nil))
}

func (rc *ResponseCache) shouldCache(req *core.Request, result *core.TextResult) bool {
    // Don't cache requests with tools (dynamic responses)
    if len(req.Tools) > 0 {
        return false
    }
    
    // Don't cache streaming requests
    if req.Stream {
        return false
    }
    
    // Don't cache high temperature requests (random responses)
    if req.Temperature > 0.8 {
        return false
    }
    
    // Don't cache very long responses
    if len(result.Text) > 10000 {
        return false
    }
    
    return true
}
```

### Request Batching

Implement intelligent request batching for efficiency:

```go
type RequestBatcher struct {
    batchSize     int
    batchTimeout  time.Duration
    processor     func([]*BatchedRequest) ([]*core.TextResult, error)
    pending       []*BatchedRequest
    timer         *time.Timer
    mutex         sync.Mutex
    resultChans   map[string]chan *BatchResult
}

type BatchedRequest struct {
    ID       string
    Request  *core.Request
    ResultCh chan *BatchResult
}

type BatchResult struct {
    Result *core.TextResult
    Error  error
}

func NewRequestBatcher(batchSize int, timeout time.Duration) *RequestBatcher {
    rb := &RequestBatcher{
        batchSize:    batchSize,
        batchTimeout: timeout,
        pending:     make([]*BatchedRequest, 0, batchSize),
        resultChans:  make(map[string]chan *BatchResult),
    }
    
    return rb
}

func (rb *RequestBatcher) SubmitRequest(ctx context.Context, req *core.Request) (*core.TextResult, error) {
    // Create batched request
    batchReq := &BatchedRequest{
        ID:       generateRequestID(),
        Request:  req,
        ResultCh: make(chan *BatchResult, 1),
    }
    
    // Add to batch
    rb.addToBatch(batchReq)
    
    // Wait for result
    select {
    case result := <-batchReq.ResultCh:
        return result.Result, result.Error
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}

func (rb *RequestBatcher) addToBatch(req *BatchedRequest) {
    rb.mutex.Lock()
    defer rb.mutex.Unlock()
    
    rb.pending = append(rb.pending, req)
    
    // Process batch if full
    if len(rb.pending) >= rb.batchSize {
        rb.processBatch()
        return
    }
    
    // Set timer for timeout if this is the first request
    if len(rb.pending) == 1 {
        rb.timer = time.AfterFunc(rb.batchTimeout, rb.processBatch)
    }
}

func (rb *RequestBatcher) processBatch() {
    rb.mutex.Lock()
    
    if len(rb.pending) == 0 {
        rb.mutex.Unlock()
        return
    }
    
    batch := rb.pending
    rb.pending = make([]*BatchedRequest, 0, rb.batchSize)
    
    if rb.timer != nil {
        rb.timer.Stop()
        rb.timer = nil
    }
    
    rb.mutex.Unlock()
    
    // Process batch asynchronously
    go func() {
        results, err := rb.processor(batch)
        
        // Send results to waiting requests
        for i, req := range batch {
            var result *BatchResult
            
            if err != nil {
                result = &BatchResult{Error: err}
            } else if i < len(results) {
                result = &BatchResult{Result: results[i]}
            } else {
                result = &BatchResult{Error: fmt.Errorf("missing result for request %s", req.ID)}
            }
            
            select {
            case req.ResultCh <- result:
            default:
                // Channel might be closed if request was cancelled
            }
        }
    }()
}
```

## Security Best Practices

### Input Validation and Sanitization

Implement comprehensive input validation:

```go
type RequestValidator struct {
    maxMessageLength int
    maxMessagesCount int
    allowedRoles     map[core.Role]bool
    contentFilters   []ContentFilter
    rateLimiter     *RateLimiter
}

type ContentFilter interface {
    Filter(text string) (filtered string, violations []string, err error)
}

func (rv *RequestValidator) ValidateRequest(ctx context.Context, req *ServiceRequest) error {
    userID := getUserID(ctx)
    
    // Rate limiting per user
    if !rv.rateLimiter.Allow(userID) {
        return &ValidationError{
            Code:    "RATE_LIMIT_EXCEEDED",
            Message: "Too many requests",
        }
    }
    
    // Validate message count
    if len(req.Messages) > rv.maxMessagesCount {
        return &ValidationError{
            Code:    "TOO_MANY_MESSAGES",
            Message: fmt.Sprintf("Maximum %d messages allowed", rv.maxMessagesCount),
        }
    }
    
    // Validate each message
    for i, msg := range req.Messages {
        if err := rv.validateMessage(msg); err != nil {
            return fmt.Errorf("message %d: %w", i, err)
        }
    }
    
    // Validate generation parameters
    if err := rv.validateGenerationParams(req); err != nil {
        return fmt.Errorf("generation params: %w", err)
    }
    
    return nil
}

func (rv *RequestValidator) validateMessage(msg core.Message) error {
    // Check role validity
    if !rv.allowedRoles[msg.Role] {
        return &ValidationError{
            Code:    "INVALID_ROLE",
            Message: fmt.Sprintf("Role %d not allowed", msg.Role),
        }
    }
    
    // Validate parts
    for i, part := range msg.Parts {
        if err := rv.validatePart(part); err != nil {
            return fmt.Errorf("part %d: %w", i, err)
        }
    }
    
    return nil
}

func (rv *RequestValidator) validatePart(part core.Part) error {
    switch p := part.(type) {
    case core.Text:
        return rv.validateTextPart(p)
    case core.ImageURL:
        return rv.validateImagePart(p)
    case core.Audio:
        return rv.validateAudioPart(p)
    default:
        return &ValidationError{
            Code:    "UNSUPPORTED_PART_TYPE",
            Message: "Unsupported content type",
        }
    }
}

func (rv *RequestValidator) validateTextPart(text core.Text) error {
    // Length check
    if len(text.Text) > rv.maxMessageLength {
        return &ValidationError{
            Code:    "TEXT_TOO_LONG",
            Message: fmt.Sprintf("Text length exceeds %d characters", rv.maxMessageLength),
        }
    }
    
    // Content filtering
    for _, filter := range rv.contentFilters {
        filtered, violations, err := filter.Filter(text.Text)
        if err != nil {
            return fmt.Errorf("content filtering error: %w", err)
        }
        
        if len(violations) > 0 {
            return &ValidationError{
                Code:    "CONTENT_POLICY_VIOLATION",
                Message: "Content violates policy",
                Context: map[string]any{"violations": violations},
            }
        }
    }
    
    return nil
}

// PII Detection Filter
type PIIFilter struct {
    patterns map[string]*regexp.Regexp
}

func NewPIIFilter() *PIIFilter {
    return &PIIFilter{
        patterns: map[string]*regexp.Regexp{
            "email":      regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
            "phone":      regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`),
            "ssn":        regexp.MustCompile(`\b\d{3}-?\d{2}-?\d{4}\b`),
            "credit_card": regexp.MustCompile(`\b(?:\d{4}[-\s]?){3}\d{4}\b`),
        },
    }
}

func (pf *PIIFilter) Filter(text string) (string, []string, error) {
    filtered := text
    var violations []string
    
    for piiType, pattern := range pf.patterns {
        if pattern.MatchString(text) {
            violations = append(violations, piiType)
            // Replace with placeholders
            filtered = pattern.ReplaceAllString(filtered, "["+strings.ToUpper(piiType)+"_REDACTED]")
        }
    }
    
    return filtered, violations, nil
}
```

### API Key Management

Implement secure API key management:

```go
type SecureKeyManager struct {
    vault      KeyVault
    cache      *keyCache
    rotator    *KeyRotator
    metrics    KeyMetrics
    logger     Logger
}

type KeyVault interface {
    GetKey(ctx context.Context, provider string, userID string) (string, error)
    SetKey(ctx context.Context, provider string, userID string, key string) error
    RotateKey(ctx context.Context, provider string, userID string) (string, error)
}

func (km *SecureKeyManager) GetProviderKey(ctx context.Context, provider, userID string) (string, error) {
    // Try cache first
    if cached := km.cache.Get(provider, userID); cached != nil {
        if !km.isKeyExpired(cached) {
            return cached.Key, nil
        }
        km.cache.Remove(provider, userID)
    }
    
    // Fetch from vault
    key, err := km.vault.GetKey(ctx, provider, userID)
    if err != nil {
        km.metrics.RecordKeyFetchError(provider, err)
        return "", fmt.Errorf("failed to fetch key: %w", err)
    }
    
    // Cache the key
    km.cache.Set(provider, userID, &CachedKey{
        Key:       key,
        ExpiresAt: time.Now().Add(30 * time.Minute),
    })
    
    return key, nil
}

func (km *SecureKeyManager) ValidateKey(ctx context.Context, provider, key string) error {
    // Test key with simple request
    testProvider := km.createTestProvider(provider, key)
    
    testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()
    
    _, err := testProvider.GenerateText(testCtx, core.Request{
        Messages: []core.Message{
            {
                Role: core.User,
                Parts: []core.Part{
                    core.Text{Text: "test"},
                },
            },
        },
        MaxTokens: 5,
    })
    
    return err
}

// Key rotation for security
type KeyRotator struct {
    schedule   time.Duration
    keyManager *SecureKeyManager
    logger     Logger
}

func (kr *KeyRotator) StartRotation(ctx context.Context) {
    ticker := time.NewTicker(kr.schedule)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            kr.rotateExpiredKeys(ctx)
        case <-ctx.Done():
            return
        }
    }
}

func (kr *KeyRotator) rotateExpiredKeys(ctx context.Context) {
    // Get all keys that need rotation
    expiredKeys := kr.keyManager.getExpiredKeys()
    
    for _, keyInfo := range expiredKeys {
        if err := kr.rotateKey(ctx, keyInfo); err != nil {
            kr.logger.Error("Failed to rotate key",
                "provider", keyInfo.Provider,
                "user_id", keyInfo.UserID,
                "error", err)
        } else {
            kr.logger.Info("Key rotated successfully",
                "provider", keyInfo.Provider,
                "user_id", keyInfo.UserID)
        }
    }
}
```

## Monitoring and Observability

### Metrics Collection

Implement comprehensive metrics collection:

```go
type MetricsCollector struct {
    registry prometheus.Registerer
    
    // Request metrics
    requestDuration    *prometheus.HistogramVec
    requestTotal       *prometheus.CounterVec
    requestTokens      *prometheus.HistogramVec
    
    // Provider metrics
    providerRequests   *prometheus.CounterVec
    providerErrors     *prometheus.CounterVec
    providerLatency    *prometheus.HistogramVec
    
    // Tool metrics
    toolExecutions     *prometheus.CounterVec
    toolDuration       *prometheus.HistogramVec
    toolErrors         *prometheus.CounterVec
    
    // System metrics
    activeConnections  prometheus.Gauge
    memoryUsage       prometheus.Gauge
    goroutines        prometheus.Gauge
}

func NewMetricsCollector() *MetricsCollector {
    mc := &MetricsCollector{
        registry: prometheus.NewRegistry(),
    }
    
    mc.initializeMetrics()
    return mc
}

func (mc *MetricsCollector) initializeMetrics() {
    mc.requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "gai_request_duration_seconds",
            Help:    "Time spent processing requests",
            Buckets: prometheus.DefBuckets,
        },
        []string{"provider", "model", "status"},
    )
    
    mc.requestTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "gai_requests_total",
            Help: "Total number of requests processed",
        },
        []string{"provider", "model", "status"},
    )
    
    mc.requestTokens = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "gai_request_tokens",
            Help:    "Number of tokens in requests",
            Buckets: []float64{10, 50, 100, 500, 1000, 5000, 10000, 50000},
        },
        []string{"provider", "model", "type"}, // type: input, output, total
    )
    
    mc.toolExecutions = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "gai_tool_executions_total",
            Help: "Total number of tool executions",
        },
        []string{"tool", "status"},
    )
    
    mc.activeConnections = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "gai_active_connections",
            Help: "Number of active connections",
        },
    )
    
    // Register all metrics
    mc.registry.MustRegister(
        mc.requestDuration,
        mc.requestTotal,
        mc.requestTokens,
        mc.toolExecutions,
        mc.activeConnections,
    )
}

func (mc *MetricsCollector) RecordRequest(provider, model string, duration time.Duration, tokens int, err error) {
    status := "success"
    if err != nil {
        status = "error"
    }
    
    mc.requestDuration.WithLabelValues(provider, model, status).Observe(duration.Seconds())
    mc.requestTotal.WithLabelValues(provider, model, status).Inc()
    mc.requestTokens.WithLabelValues(provider, model, "total").Observe(float64(tokens))
}

func (mc *MetricsCollector) RecordToolExecution(toolName string, duration time.Duration, err error) {
    status := "success"
    if err != nil {
        status = "error"
    }
    
    mc.toolExecutions.WithLabelValues(toolName, status).Inc()
}

// Middleware integration
func MetricsMiddleware(collector *MetricsCollector) middleware.Middleware {
    return func(provider core.Provider) core.Provider {
        return &metricsProvider{
            inner:     provider,
            collector: collector,
        }
    }
}

type metricsProvider struct {
    inner     core.Provider
    collector *MetricsCollector
}

func (mp *metricsProvider) GenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {
    start := time.Now()
    
    result, err := mp.inner.GenerateText(ctx, req)
    
    duration := time.Since(start)
    tokens := 0
    if result != nil {
        tokens = result.Usage.TotalTokens
    }
    
    mp.collector.RecordRequest(
        getProviderName(mp.inner),
        req.Model,
        duration,
        tokens,
        err,
    )
    
    return result, err
}
```

### Distributed Tracing

Implement distributed tracing with OpenTelemetry:

```go
type TracingManager struct {
    tracer trace.Tracer
    config TracingConfig
}

type TracingConfig struct {
    ServiceName     string `json:"service_name"`
    ServiceVersion  string `json:"service_version"`
    JaegerEndpoint  string `json:"jaeger_endpoint"`
    SamplingRatio   float64 `json:"sampling_ratio"`
    EnabledProviders []string `json:"enabled_providers"`
}

func NewTracingManager(config TracingConfig) (*TracingManager, error) {
    // Create tracer provider
    tp, err := tracerProvider(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create tracer provider: %w", err)
    }
    
    otel.SetTracerProvider(tp)
    otel.SetTextMapPropagator(propagation.TraceContext{})
    
    tracer := tp.Tracer(config.ServiceName)
    
    return &TracingManager{
        tracer: tracer,
        config: config,
    }, nil
}

func (tm *TracingManager) TraceRequest(ctx context.Context, operationName string, req *core.Request, fn func(context.Context) (*core.TextResult, error)) (*core.TextResult, error) {
    ctx, span := tm.tracer.Start(ctx, operationName)
    defer span.End()
    
    // Add request attributes
    span.SetAttributes(
        attribute.String("ai.provider", getProviderFromContext(ctx)),
        attribute.String("ai.model", req.Model),
        attribute.Int("ai.max_tokens", req.MaxTokens),
        attribute.Float64("ai.temperature", float64(req.Temperature)),
        attribute.Int("ai.message_count", len(req.Messages)),
        attribute.Int("ai.tool_count", len(req.Tools)),
    )
    
    // Add message content (be careful with sensitive data)
    if tm.config.IncludeMessageContent {
        for i, msg := range req.Messages {
            if text := extractTextFromMessage(msg); text != "" {
                span.SetAttributes(
                    attribute.String(fmt.Sprintf("ai.message.%d.role", i), msg.Role.String()),
                    attribute.String(fmt.Sprintf("ai.message.%d.content", i), truncate(text, 200)),
                )
            }
        }
    }
    
    result, err := fn(ctx)
    
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        
        // Add error details
        span.SetAttributes(
            attribute.String("error.type", getErrorType(err)),
            attribute.Bool("error.retryable", isRetryable(err)),
        )
    } else {
        span.SetStatus(codes.Ok, "")
        
        // Add response attributes
        span.SetAttributes(
            attribute.Int("ai.response.tokens.input", result.Usage.InputTokens),
            attribute.Int("ai.response.tokens.output", result.Usage.OutputTokens),
            attribute.Int("ai.response.tokens.total", result.Usage.TotalTokens),
            attribute.Int("ai.response.steps", len(result.Steps)),
        )
        
        // Add tool execution details
        for i, step := range result.Steps {
            for j, call := range step.ToolCalls {
                span.SetAttributes(
                    attribute.String(fmt.Sprintf("ai.step.%d.tool.%d.name", i, j), call.Name),
                )
            }
        }
    }
    
    return result, err
}

// Middleware integration
func TracingMiddleware(tm *TracingManager) middleware.Middleware {
    return func(provider core.Provider) core.Provider {
        return &tracingProvider{
            inner:   provider,
            tracer:  tm,
        }
    }
}

type tracingProvider struct {
    inner  core.Provider
    tracer *TracingManager
}

func (tp *tracingProvider) GenerateText(ctx context.Context, req core.Request) (*core.TextResult, error) {
    return tp.tracer.TraceRequest(ctx, "ai.generate_text", &req, func(ctx context.Context) (*core.TextResult, error) {
        return tp.inner.GenerateText(ctx, req)
    })
}
```

This comprehensive guide provides the foundation for building production-ready AI applications with GAI. Each section includes practical implementation examples and real-world patterns that you can adapt to your specific requirements.

The key to success is implementing these patterns incrementally, monitoring their effectiveness, and iterating based on your application's specific needs and constraints.