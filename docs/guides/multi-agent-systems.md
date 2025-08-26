# Building Multi-Agent Systems with GAI

This comprehensive guide covers designing, implementing, and managing sophisticated multi-agent AI systems using GAI's advanced workflow capabilities.

## Table of Contents
- [Multi-Agent Architecture](#multi-agent-architecture)
- [Agent Design Patterns](#agent-design-patterns)
- [Communication and Coordination](#communication-and-coordination)
- [Workflow Orchestration](#workflow-orchestration)
- [State Management](#state-management)
- [Real-World Examples](#real-world-examples)
- [Monitoring and Debugging](#monitoring-and-debugging)
- [Performance Optimization](#performance-optimization)
- [Best Practices](#best-practices)

## Multi-Agent Architecture

### Core Agent Framework

Build a flexible foundation for multi-agent systems:

```go
// Base agent interface
type Agent interface {
    ID() string
    Name() string
    Role() string
    Capabilities() []string
    Execute(ctx context.Context, task *Task) (*TaskResult, error)
    CanHandle(task *Task) bool
    GetStatus() AgentStatus
}

// Agent implementation
type BaseAgent struct {
    id           string
    name         string
    role         string
    provider     core.Provider
    tools        []tools.Handle
    capabilities []string
    memory       AgentMemory
    config       AgentConfig
    metrics      AgentMetrics
    logger       Logger
}

type AgentConfig struct {
    MaxSteps         int           `json:"max_steps"`
    Temperature      float32       `json:"temperature"`
    MaxTokens        int           `json:"max_tokens"`
    ToolTimeout      time.Duration `json:"tool_timeout"`
    EnableMemory     bool          `json:"enable_memory"`
    MemoryLimit      int           `json:"memory_limit"`
    PreferredModel   string        `json:"preferred_model"`
    SystemPrompt     string        `json:"system_prompt"`
}

func NewAgent(id, name, role string, provider core.Provider, config AgentConfig) *BaseAgent {
    return &BaseAgent{
        id:           id,
        name:         name,
        role:         role,
        provider:     provider,
        config:       config,
        memory:       NewAgentMemory(config.MemoryLimit),
        metrics:      NewAgentMetrics(id),
        logger:       NewLogger(fmt.Sprintf("[Agent:%s]", name)),
    }
}

func (a *BaseAgent) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
    a.logger.Info("Executing task", "task_id", task.ID, "task_type", task.Type)
    
    startTime := time.Now()
    defer func() {
        a.metrics.RecordTaskExecution(task.Type, time.Since(startTime))
    }()
    
    // Check if agent can handle this task
    if !a.CanHandle(task) {
        return nil, &AgentError{
            Code:    "TASK_NOT_SUPPORTED",
            Message: fmt.Sprintf("Agent %s cannot handle task type %s", a.name, task.Type),
        }
    }
    
    // Prepare messages with context
    messages := a.buildTaskMessages(task)
    
    // Execute with stop conditions
    request := core.Request{
        Messages:    messages,
        Tools:       a.tools,
        Temperature: a.config.Temperature,
        MaxTokens:   a.config.MaxTokens,
        StopWhen: core.CombineConditions(
            core.MaxSteps(a.config.MaxSteps),
            core.NoMoreTools(),
            a.createTaskSpecificStopCondition(task),
        ),
    }
    
    result, err := a.provider.GenerateText(ctx, request)
    if err != nil {
        a.metrics.RecordTaskError(task.Type, err)
        return nil, fmt.Errorf("task execution failed: %w", err)
    }
    
    // Process result and update memory
    taskResult := a.processTaskResult(task, result)
    
    if a.config.EnableMemory {
        a.memory.Store(task.ID, taskResult)
    }
    
    a.metrics.RecordTaskSuccess(task.Type, result.Usage.TotalTokens)
    return taskResult, nil
}

func (a *BaseAgent) buildTaskMessages(task *Task) []core.Message {
    messages := []core.Message{
        {
            Role: core.System,
            Parts: []core.Part{
                core.Text{Text: a.buildSystemPrompt(task)},
            },
        },
    }
    
    // Add relevant memory context
    if a.config.EnableMemory {
        if context := a.memory.GetRelevantContext(task); len(context) > 0 {
            messages = append(messages, core.Message{
                Role: core.System,
                Parts: []core.Part{
                    core.Text{Text: "Relevant context from previous tasks:\n" + context},
                },
            })
        }
    }
    
    // Add task-specific messages
    for _, part := range task.Input {
        messages = append(messages, core.Message{
            Role:  core.User,
            Parts: []core.Part{part},
        })
    }
    
    return messages
}

func (a *BaseAgent) buildSystemPrompt(task *Task) string {
    prompt := fmt.Sprintf(`You are %s, a %s agent.

Your role: %s

Available capabilities: %v

Current task: %s

Instructions:
- Follow your role and use your capabilities effectively
- Use available tools when needed
- Be thorough and accurate in your work
- Provide clear reasoning for your decisions
`,
        a.name,
        a.role,
        a.config.SystemPrompt,
        a.capabilities,
        task.Description,
    )
    
    return prompt
}
```

### Specialized Agent Types

Create specialized agents for different domains:

```go
// Research Agent - specializes in information gathering
type ResearchAgent struct {
    *BaseAgent
    searchEngine   tools.Handle
    database       tools.Handle
    documentReader tools.Handle
}

func NewResearchAgent(provider core.Provider) *ResearchAgent {
    base := NewAgent(
        "research-001",
        "ResearchBot",
        "researcher",
        provider,
        AgentConfig{
            MaxSteps:     15,
            Temperature:  0.3, // More factual responses
            MaxTokens:    2000,
            SystemPrompt: "You are a thorough researcher. Gather comprehensive, accurate information from multiple sources. Cite your sources and provide detailed analysis.",
        },
    )
    
    ra := &ResearchAgent{
        BaseAgent:      base,
        searchEngine:   createWebSearchTool(),
        database:       createDatabaseQueryTool(),
        documentReader: createDocumentAnalysisTool(),
    }
    
    ra.tools = []tools.Handle{ra.searchEngine, ra.database, ra.documentReader}
    ra.capabilities = []string{"web_search", "database_query", "document_analysis", "fact_checking"}
    
    return ra
}

func (ra *ResearchAgent) CanHandle(task *Task) bool {
    supportedTypes := []string{"research", "fact_check", "information_gathering", "analysis"}
    for _, supported := range supportedTypes {
        if task.Type == supported {
            return true
        }
    }
    return false
}

// Analysis Agent - specializes in data analysis and insights
type AnalysisAgent struct {
    *BaseAgent
    statisticalTool tools.Handle
    visualization   tools.Handle
    mlTool         tools.Handle
}

func NewAnalysisAgent(provider core.Provider) *AnalysisAgent {
    base := NewAgent(
        "analysis-001",
        "AnalyticsBot",
        "analyst",
        provider,
        AgentConfig{
            MaxSteps:     20,
            Temperature:  0.4,
            MaxTokens:    3000,
            SystemPrompt: "You are a data analyst expert. Perform thorough statistical analysis, identify patterns, and generate actionable insights. Always support conclusions with data.",
        },
    )
    
    aa := &AnalysisAgent{
        BaseAgent:       base,
        statisticalTool: createStatisticalAnalysisTool(),
        visualization:   createVisualizationTool(),
        mlTool:         createMLAnalysisTool(),
    }
    
    aa.tools = []tools.Handle{aa.statisticalTool, aa.visualization, aa.mlTool}
    aa.capabilities = []string{"statistical_analysis", "data_visualization", "pattern_recognition", "predictive_modeling"}
    
    return aa
}

// Coordination Agent - manages and orchestrates other agents
type CoordinationAgent struct {
    *BaseAgent
    agentManager   *AgentManager
    taskScheduler  *TaskScheduler
    workflowEngine *WorkflowEngine
}

func NewCoordinationAgent(provider core.Provider, manager *AgentManager) *CoordinationAgent {
    base := NewAgent(
        "coordinator-001",
        "CoordinatorBot",
        "coordinator",
        provider,
        AgentConfig{
            MaxSteps:     30,
            Temperature:  0.6,
            MaxTokens:    2000,
            SystemPrompt: "You are a project coordinator. Break down complex tasks, assign work to appropriate agents, monitor progress, and ensure quality outcomes.",
        },
    )
    
    ca := &CoordinationAgent{
        BaseAgent:      base,
        agentManager:   manager,
        taskScheduler:  NewTaskScheduler(),
        workflowEngine: NewWorkflowEngine(),
    }
    
    ca.tools = []tools.Handle{
        createTaskManagementTool(ca.taskScheduler),
        createAgentCommunicationTool(ca.agentManager),
        createWorkflowControlTool(ca.workflowEngine),
    }
    ca.capabilities = []string{"task_decomposition", "agent_coordination", "workflow_management", "quality_assurance"}
    
    return ca
}
```

## Agent Design Patterns

### Hierarchical Agent Pattern

Implement hierarchical agent structures for complex organizations:

```go
type AgentHierarchy struct {
    root     Agent
    children map[string][]*AgentNode
    parent   map[string]Agent
    levels   map[Agent]int
}

type AgentNode struct {
    agent    Agent
    children []*AgentNode
    parent   *AgentNode
    level    int
}

func (ah *AgentHierarchy) AddAgent(agent Agent, parentAgent Agent) error {
    if parentAgent == nil {
        // Root agent
        ah.root = agent
        ah.levels[agent] = 0
    } else {
        // Add as child
        ah.children[parentAgent.ID()] = append(ah.children[parentAgent.ID()], &AgentNode{
            agent: agent,
            level: ah.levels[parentAgent] + 1,
        })
        ah.parent[agent.ID()] = parentAgent
        ah.levels[agent] = ah.levels[parentAgent] + 1
    }
    
    return nil
}

func (ah *AgentHierarchy) ExecuteTask(ctx context.Context, task *Task) (*TaskResult, error) {
    // Start with root agent for task decomposition
    return ah.executeWithHierarchy(ctx, ah.root, task, 0)
}

func (ah *AgentHierarchy) executeWithHierarchy(ctx context.Context, agent Agent, task *Task, depth int) (*TaskResult, error) {
    if depth > 10 { // Prevent infinite recursion
        return nil, fmt.Errorf("maximum hierarchy depth reached")
    }
    
    // Try to handle task directly
    if agent.CanHandle(task) {
        result, err := agent.Execute(ctx, task)
        if err == nil {
            return result, nil
        }
    }
    
    // If agent can't handle or failed, delegate to children
    children := ah.children[agent.ID()]
    if len(children) == 0 {
        return nil, fmt.Errorf("no capable agent found for task %s", task.Type)
    }
    
    // Try each child in order of capability match
    for _, child := range children {
        if child.agent.CanHandle(task) {
            result, err := ah.executeWithHierarchy(ctx, child.agent, task, depth+1)
            if err == nil {
                return result, nil
            }
        }
    }
    
    return nil, fmt.Errorf("task %s could not be completed by hierarchy", task.Type)
}
```

### Swarm Agent Pattern

Implement swarm intelligence for parallel problem-solving:

```go
type AgentSwarm struct {
    agents      []Agent
    coordinator Agent
    consensus   ConsensusEngine
    aggregator  ResultAggregator
    config      SwarmConfig
}

type SwarmConfig struct {
    MinConsensus    float64       `json:"min_consensus"`
    MaxAgents       int           `json:"max_agents"`
    Timeout         time.Duration `json:"timeout"`
    VotingStrategy  string        `json:"voting_strategy"` // "majority", "weighted", "unanimous"
    QualityThreshold float64      `json:"quality_threshold"`
}

func (s *AgentSwarm) ExecuteWithConsensus(ctx context.Context, task *Task) (*TaskResult, error) {
    // Filter capable agents
    capableAgents := s.filterCapableAgents(task)
    if len(capableAgents) == 0 {
        return nil, fmt.Errorf("no capable agents for task %s", task.Type)
    }
    
    // Limit number of agents for efficiency
    if len(capableAgents) > s.config.MaxAgents {
        capableAgents = s.selectBestAgents(capableAgents, task, s.config.MaxAgents)
    }
    
    // Execute task in parallel across agents
    results := make(chan *AgentResult, len(capableAgents))
    
    for _, agent := range capableAgents {
        go func(a Agent) {
            result, err := a.Execute(ctx, task)
            results <- &AgentResult{
                AgentID: a.ID(),
                Result:  result,
                Error:   err,
                Agent:   a,
            }
        }(agent)
    }
    
    // Collect results with timeout
    var agentResults []*AgentResult
    timeout := time.After(s.config.Timeout)
    
    for i := 0; i < len(capableAgents); i++ {
        select {
        case result := <-results:
            agentResults = append(agentResults, result)
        case <-timeout:
            break
        case <-ctx.Done():
            return nil, ctx.Err()
        }
    }
    
    // Check for consensus
    consensus, quality := s.consensus.EvaluateConsensus(agentResults)
    if consensus < s.config.MinConsensus {
        return nil, fmt.Errorf("insufficient consensus: %.2f < %.2f", consensus, s.config.MinConsensus)
    }
    
    if quality < s.config.QualityThreshold {
        return nil, fmt.Errorf("quality threshold not met: %.2f < %.2f", quality, s.config.QualityThreshold)
    }
    
    // Aggregate results
    finalResult := s.aggregator.Aggregate(agentResults, s.config.VotingStrategy)
    return finalResult, nil
}

type ConsensusEngine struct {
    similarityThreshold float64
    qualityWeights      map[string]float64
}

func (ce *ConsensusEngine) EvaluateConsensus(results []*AgentResult) (consensus, quality float64) {
    if len(results) < 2 {
        return 1.0, 0.0 // Single result - consensus but unknown quality
    }
    
    // Calculate pairwise similarities
    similarities := make([][]float64, len(results))
    for i := range similarities {
        similarities[i] = make([]float64, len(results))
    }
    
    for i := 0; i < len(results); i++ {
        for j := i + 1; j < len(results); j++ {
            sim := ce.calculateSimilarity(results[i], results[j])
            similarities[i][j] = sim
            similarities[j][i] = sim
        }
    }
    
    // Calculate consensus as average similarity
    var totalSimilarity float64
    var pairCount int
    
    for i := 0; i < len(results); i++ {
        for j := i + 1; j < len(results); j++ {
            totalSimilarity += similarities[i][j]
            pairCount++
        }
    }
    
    if pairCount == 0 {
        return 1.0, 0.0
    }
    
    consensus = totalSimilarity / float64(pairCount)
    
    // Calculate quality based on multiple factors
    quality = ce.calculateQuality(results, consensus)
    
    return consensus, quality
}

func (ce *ConsensusEngine) calculateSimilarity(r1, r2 *AgentResult) float64 {
    if r1.Error != nil || r2.Error != nil {
        if r1.Error != nil && r2.Error != nil {
            return 1.0 // Both failed - similar outcome
        }
        return 0.0 // One succeeded, one failed - dissimilar
    }
    
    // Compare result content using various metrics
    textSimilarity := calculateTextSimilarity(r1.Result.Content, r2.Result.Content)
    confidenceSimilarity := 1.0 - math.Abs(r1.Result.Confidence-r2.Result.Confidence)
    
    // Weighted average
    return 0.7*textSimilarity + 0.3*confidenceSimilarity
}
```

## Communication and Coordination

### Inter-Agent Messaging

Implement robust communication between agents:

```go
type MessageBus struct {
    subscribers map[string][]chan *AgentMessage
    mutex       sync.RWMutex
    logger      Logger
    metrics     MessageMetrics
}

type AgentMessage struct {
    ID          string                 `json:"id"`
    FromAgent   string                 `json:"from_agent"`
    ToAgent     string                 `json:"to_agent"`
    Type        MessageType            `json:"type"`
    Content     string                 `json:"content"`
    Data        map[string]interface{} `json:"data,omitempty"`
    Timestamp   time.Time              `json:"timestamp"`
    ReplyTo     string                 `json:"reply_to,omitempty"`
    Priority    Priority               `json:"priority"`
    TTL         time.Duration          `json:"ttl,omitempty"`
}

type MessageType int
const (
    MessageTypeRequest MessageType = iota
    MessageTypeResponse
    MessageTypeNotification
    MessageTypeCommand
    MessageTypeBroadcast
)

type Priority int
const (
    PriorityLow Priority = iota
    PriorityNormal
    PriorityHigh
    PriorityCritical
)

func (mb *MessageBus) Subscribe(agentID string) <-chan *AgentMessage {
    mb.mutex.Lock()
    defer mb.mutex.Unlock()
    
    ch := make(chan *AgentMessage, 100)
    mb.subscribers[agentID] = append(mb.subscribers[agentID], ch)
    
    return ch
}

func (mb *MessageBus) SendMessage(message *AgentMessage) error {
    mb.mutex.RLock()
    defer mb.mutex.RUnlock()
    
    // Set timestamp
    message.Timestamp = time.Now()
    message.ID = generateMessageID()
    
    // Log the message
    mb.logger.Info("Sending message",
        "from", message.FromAgent,
        "to", message.ToAgent,
        "type", message.Type,
        "priority", message.Priority)
    
    // Record metrics
    mb.metrics.RecordMessage(message)
    
    // Send to specific agent
    if message.ToAgent != "" {
        return mb.sendToAgent(message.ToAgent, message)
    }
    
    // Broadcast to all agents
    if message.Type == MessageTypeBroadcast {
        return mb.broadcast(message)
    }
    
    return fmt.Errorf("invalid message routing")
}

func (mb *MessageBus) sendToAgent(agentID string, message *AgentMessage) error {
    channels := mb.subscribers[agentID]
    if len(channels) == 0 {
        return fmt.Errorf("agent %s not subscribed", agentID)
    }
    
    // Send to all channels for this agent (for redundancy)
    delivered := false
    for _, ch := range channels {
        select {
        case ch <- message:
            delivered = true
        default:
            mb.logger.Warn("Message channel full", "agent", agentID)
        }
    }
    
    if !delivered {
        return fmt.Errorf("failed to deliver message to agent %s", agentID)
    }
    
    return nil
}

// Request-Response Pattern
func (mb *MessageBus) SendRequest(fromAgent, toAgent string, content string, data map[string]interface{}, timeout time.Duration) (*AgentMessage, error) {
    requestID := generateMessageID()
    
    request := &AgentMessage{
        FromAgent: fromAgent,
        ToAgent:   toAgent,
        Type:      MessageTypeRequest,
        Content:   content,
        Data:      data,
        Priority:  PriorityNormal,
        TTL:       timeout,
    }
    
    // Create response channel
    responseChannel := make(chan *AgentMessage, 1)
    
    // Subscribe to response
    go func() {
        subscription := mb.Subscribe(fromAgent)
        defer close(responseChannel)
        
        timeoutTimer := time.After(timeout)
        
        for {
            select {
            case msg := <-subscription:
                if msg.Type == MessageTypeResponse && msg.ReplyTo == requestID {
                    responseChannel <- msg
                    return
                }
            case <-timeoutTimer:
                return
            }
        }
    }()
    
    // Send request
    request.ID = requestID
    if err := mb.SendMessage(request); err != nil {
        return nil, fmt.Errorf("failed to send request: %w", err)
    }
    
    // Wait for response
    select {
    case response := <-responseChannel:
        return response, nil
    case <-time.After(timeout):
        return nil, fmt.Errorf("request timeout")
    }
}
```

### Task Coordination

Implement sophisticated task coordination mechanisms:

```go
type TaskCoordinator struct {
    agents         map[string]Agent
    taskQueue      *PriorityQueue
    dependencies   *DependencyGraph
    scheduler      *TaskScheduler
    monitor        *TaskMonitor
    messageBus     *MessageBus
    workflowEngine *WorkflowEngine
}

type Task struct {
    ID           string                 `json:"id"`
    Type         string                 `json:"type"`
    Description  string                 `json:"description"`
    Input        []core.Part           `json:"input"`
    Priority     Priority              `json:"priority"`
    Dependencies []string              `json:"dependencies"`
    Deadline     *time.Time            `json:"deadline,omitempty"`
    RequiredCapabilities []string      `json:"required_capabilities"`
    Metadata     map[string]interface{} `json:"metadata"`
    Status       TaskStatus            `json:"status"`
    AssignedAgent string               `json:"assigned_agent,omitempty"`
    CreatedAt    time.Time             `json:"created_at"`
    UpdatedAt    time.Time             `json:"updated_at"`
}

type TaskStatus int
const (
    TaskStatusPending TaskStatus = iota
    TaskStatusAssigned
    TaskStatusInProgress
    TaskStatusCompleted
    TaskStatusFailed
    TaskStatusCancelled
)

func (tc *TaskCoordinator) SubmitTask(task *Task) error {
    // Validate task
    if err := tc.validateTask(task); err != nil {
        return fmt.Errorf("task validation failed: %w", err)
    }
    
    // Set initial status
    task.Status = TaskStatusPending
    task.CreatedAt = time.Now()
    task.UpdatedAt = time.Now()
    task.ID = generateTaskID()
    
    // Add to queue
    tc.taskQueue.Push(task)
    
    // Start coordination process
    go tc.coordinateTask(task)
    
    return nil
}

func (tc *TaskCoordinator) coordinateTask(task *Task) {
    // Check dependencies
    if !tc.dependencies.AreResolved(task.Dependencies) {
        tc.waitForDependencies(task)
    }
    
    // Find suitable agent
    agent := tc.findBestAgent(task)
    if agent == nil {
        tc.handleUnassignableTask(task)
        return
    }
    
    // Assign task
    task.AssignedAgent = agent.ID()
    task.Status = TaskStatusAssigned
    task.UpdatedAt = time.Now()
    
    // Notify agent
    message := &AgentMessage{
        FromAgent: "coordinator",
        ToAgent:   agent.ID(),
        Type:      MessageTypeCommand,
        Content:   "execute_task",
        Data: map[string]interface{}{
            "task": task,
        },
        Priority: convertTaskPriority(task.Priority),
    }
    
    if err := tc.messageBus.SendMessage(message); err != nil {
        tc.handleTaskError(task, fmt.Errorf("failed to notify agent: %w", err))
        return
    }
    
    // Monitor execution
    tc.monitor.StartMonitoring(task)
}

func (tc *TaskCoordinator) findBestAgent(task *Task) Agent {
    var bestAgent Agent
    var bestScore float64
    
    for _, agent := range tc.agents {
        if !agent.CanHandle(task) {
            continue
        }
        
        score := tc.calculateAgentScore(agent, task)
        if score > bestScore {
            bestScore = score
            bestAgent = agent
        }
    }
    
    return bestAgent
}

func (tc *TaskCoordinator) calculateAgentScore(agent Agent, task *Task) float64 {
    score := 0.0
    
    // Capability match score
    capabilities := agent.Capabilities()
    matchCount := 0
    for _, required := range task.RequiredCapabilities {
        for _, available := range capabilities {
            if required == available {
                matchCount++
                break
            }
        }
    }
    
    if len(task.RequiredCapabilities) > 0 {
        score += float64(matchCount) / float64(len(task.RequiredCapabilities)) * 0.4
    }
    
    // Agent load score (prefer less busy agents)
    status := agent.GetStatus()
    loadScore := 1.0 - status.CurrentLoad
    score += loadScore * 0.3
    
    // Agent performance score
    performanceScore := tc.getAgentPerformanceScore(agent, task.Type)
    score += performanceScore * 0.3
    
    return score
}

// Complex Workflow Coordination
func (tc *TaskCoordinator) ExecuteWorkflow(ctx context.Context, workflow *Workflow) (*WorkflowResult, error) {
    // Parse workflow into task graph
    taskGraph := tc.parseWorkflowToTasks(workflow)
    
    // Execute tasks according to dependencies
    results := make(map[string]*TaskResult)
    errors := make(map[string]error)
    
    // Track completion
    completed := make(chan TaskCompletion, len(taskGraph.Tasks))
    
    // Execute ready tasks
    for {
        readyTasks := taskGraph.GetReadyTasks(results)
        if len(readyTasks) == 0 {
            break // All done or blocked
        }
        
        // Execute ready tasks in parallel
        for _, task := range readyTasks {
            go func(t *Task) {
                result, err := tc.executeTaskWithAgent(ctx, t)
                completed <- TaskCompletion{
                    TaskID: t.ID,
                    Result: result,
                    Error:  err,
                }
            }(task)
        }
        
        // Wait for completions
        for i := 0; i < len(readyTasks); i++ {
            select {
            case completion := <-completed:
                if completion.Error != nil {
                    errors[completion.TaskID] = completion.Error
                    // Handle workflow failure policy
                    if workflow.FailurePolicy == FailOnAnyError {
                        return nil, fmt.Errorf("workflow failed on task %s: %w", completion.TaskID, completion.Error)
                    }
                } else {
                    results[completion.TaskID] = completion.Result
                }
            case <-ctx.Done():
                return nil, ctx.Err()
            }
        }
    }
    
    // Compile workflow result
    workflowResult := &WorkflowResult{
        WorkflowID:    workflow.ID,
        Status:        WorkflowStatusCompleted,
        TaskResults:   results,
        TaskErrors:    errors,
        StartTime:     workflow.StartTime,
        EndTime:       time.Now(),
        TotalTasks:    len(taskGraph.Tasks),
        SuccessfulTasks: len(results),
        FailedTasks:   len(errors),
    }
    
    return workflowResult, nil
}
```

## Real-World Examples

### Customer Service Multi-Agent System

Build a comprehensive customer service system:

```go
type CustomerServiceSystem struct {
    triageAgent      Agent
    supportAgent     Agent
    techSupportAgent Agent
    escalationAgent  Agent
    coordinator      *TaskCoordinator
    knowledgeBase    *KnowledgeBase
    ticketManager    *TicketManager
}

func NewCustomerServiceSystem() *CustomerServiceSystem {
    // Create specialized agents
    triage := NewTriageAgent()
    support := NewSupportAgent()
    techSupport := NewTechSupportAgent()
    escalation := NewEscalationAgent()
    
    // Create coordination infrastructure
    coordinator := NewTaskCoordinator()
    coordinator.RegisterAgent(triage)
    coordinator.RegisterAgent(support)
    coordinator.RegisterAgent(techSupport)
    coordinator.RegisterAgent(escalation)
    
    return &CustomerServiceSystem{
        triageAgent:      triage,
        supportAgent:     support,
        techSupportAgent: techSupport,
        escalationAgent:  escalation,
        coordinator:      coordinator,
        knowledgeBase:    NewKnowledgeBase(),
        ticketManager:    NewTicketManager(),
    }
}

func (css *CustomerServiceSystem) HandleCustomerInquiry(ctx context.Context, inquiry *CustomerInquiry) (*ServiceResponse, error) {
    // Create initial ticket
    ticket := css.ticketManager.CreateTicket(inquiry)
    
    // Define workflow
    workflow := &Workflow{
        ID:          generateWorkflowID(),
        Name:        "customer_service_workflow",
        Description: "Handle customer inquiry end-to-end",
        Tasks: []*WorkflowTask{
            {
                ID:          "triage",
                Type:        "triage_inquiry",
                Agent:       "triage_agent",
                Description: "Classify and prioritize customer inquiry",
                Input:       inquiry.Content,
            },
            {
                ID:          "primary_support",
                Type:        "provide_support",
                Agent:       "support_agent",
                Description: "Provide initial customer support",
                Dependencies: []string{"triage"},
                Conditions: []Condition{
                    {
                        Field:    "triage.priority",
                        Operator: "in",
                        Value:    []string{"low", "medium"},
                    },
                },
            },
            {
                ID:          "tech_support",
                Type:        "technical_support",
                Agent:       "tech_support_agent", 
                Description: "Provide technical support for complex issues",
                Dependencies: []string{"triage"},
                Conditions: []Condition{
                    {
                        Field:    "triage.category",
                        Operator: "equals",
                        Value:    "technical",
                    },
                },
            },
            {
                ID:          "escalation",
                Type:        "escalate_issue",
                Agent:       "escalation_agent",
                Description: "Escalate high-priority or unresolved issues",
                Dependencies: []string{"triage", "primary_support", "tech_support"},
                Conditions: []Condition{
                    {
                        Field:    "triage.priority",
                        Operator: "equals",
                        Value:    "high",
                    },
                    {
                        Field:    "primary_support.resolution_status",
                        Operator: "equals",
                        Value:    "unresolved",
                    },
                },
            },
        },
        FailurePolicy: ContinueOnError,
    }
    
    // Execute workflow
    result, err := css.coordinator.ExecuteWorkflow(ctx, workflow)
    if err != nil {
        return nil, fmt.Errorf("customer service workflow failed: %w", err)
    }
    
    // Compile final response
    response := css.compileServiceResponse(ticket, result)
    
    // Update ticket status
    css.ticketManager.UpdateTicket(ticket.ID, response)
    
    return response, nil
}

// Triage Agent Implementation
type TriageAgent struct {
    *BaseAgent
    classifier tools.Handle
    prioritizer tools.Handle
}

func NewTriageAgent() *TriageAgent {
    base := NewAgent(
        "triage-001",
        "TriageBot",
        "triage_specialist",
        anthropic.New(anthropic.WithModel(anthropic.ClaudeSonnet4)),
        AgentConfig{
            MaxSteps:    8,
            Temperature: 0.2, // More consistent classifications
            MaxTokens:   1000,
            SystemPrompt: `You are a customer service triage specialist. Your job is to:
1. Analyze customer inquiries and classify them by category and priority
2. Extract key information and context
3. Route inquiries to the appropriate support team
4. Ensure critical issues are flagged appropriately

Categories: technical, billing, general, product_info, complaint, feature_request
Priorities: low, medium, high, critical
`,
        },
    )
    
    ta := &TriageAgent{
        BaseAgent:   base,
        classifier:  createInquiryClassifierTool(),
        prioritizer: createPriorityAssessmentTool(),
    }
    
    ta.tools = []tools.Handle{ta.classifier, ta.prioritizer}
    ta.capabilities = []string{"inquiry_classification", "priority_assessment", "routing"}
    
    return ta
}

// Support Agent Implementation  
type SupportAgent struct {
    *BaseAgent
    knowledgeSearch tools.Handle
    solutionFinder  tools.Handle
    ticketUpdater   tools.Handle
}

func NewSupportAgent() *SupportAgent {
    base := NewAgent(
        "support-001", 
        "SupportBot",
        "support_specialist",
        openai.New(openai.WithModel(openai.GPT4oMini)),
        AgentConfig{
            MaxSteps:    15,
            Temperature: 0.4,
            MaxTokens:   2000,
            SystemPrompt: `You are a customer support specialist. Your job is to:
1. Provide helpful, accurate solutions to customer problems
2. Search knowledge base for relevant information
3. Escalate issues when necessary
4. Maintain a helpful and professional tone
5. Follow up to ensure customer satisfaction
`,
        },
    )
    
    sa := &SupportAgent{
        BaseAgent:       base,
        knowledgeSearch: createKnowledgeSearchTool(),
        solutionFinder:  createSolutionFinderTool(), 
        ticketUpdater:   createTicketUpdateTool(),
    }
    
    sa.tools = []tools.Handle{sa.knowledgeSearch, sa.solutionFinder, sa.ticketUpdater}
    sa.capabilities = []string{"problem_solving", "knowledge_search", "customer_communication", "solution_provision"}
    
    return sa
}
```

### Research and Analysis Pipeline

Create a sophisticated research and analysis system:

```go
type ResearchPipeline struct {
    researchAgent   Agent
    analysisAgent   Agent
    synthesisAgent  Agent
    reviewAgent     Agent
    coordinator     *PipelineCoordinator
    dataStore      *DataStore
    reportGenerator *ReportGenerator
}

func (rp *ResearchPipeline) ExecuteResearchProject(ctx context.Context, project *ResearchProject) (*ResearchReport, error) {
    // Define research pipeline workflow
    pipeline := &Pipeline{
        ID:   project.ID,
        Name: "research_pipeline",
        Stages: []*PipelineStage{
            {
                Name:        "information_gathering",
                Agent:       rp.researchAgent,
                Description: "Gather comprehensive information on research topics",
                Inputs:      project.Topics,
                ParallelTasks: len(project.Topics), // Research topics in parallel
            },
            {
                Name:        "data_analysis", 
                Agent:       rp.analysisAgent,
                Description: "Analyze gathered data and identify patterns",
                Dependencies: []string{"information_gathering"},
            },
            {
                Name:        "synthesis",
                Agent:       rp.synthesisAgent,
                Description: "Synthesize findings into coherent insights",
                Dependencies: []string{"data_analysis"},
            },
            {
                Name:        "peer_review",
                Agent:       rp.reviewAgent,
                Description: "Review findings for accuracy and completeness",
                Dependencies: []string{"synthesis"},
                ValidationCriteria: []ValidationCriterion{
                    {
                        Type:      "factual_accuracy",
                        Threshold: 0.95,
                    },
                    {
                        Type:      "completeness",
                        Threshold: 0.85,
                    },
                },
            },
        },
        QualityGate: QualityGate{
            RequiredAccuracy: 0.9,
            RequiredCompleteness: 0.8,
            MaxIterations: 3,
        },
    }
    
    // Execute pipeline
    result, err := rp.coordinator.ExecutePipeline(ctx, pipeline)
    if err != nil {
        return nil, fmt.Errorf("research pipeline execution failed: %w", err)
    }
    
    // Generate final report
    report, err := rp.reportGenerator.GenerateReport(project, result)
    if err != nil {
        return nil, fmt.Errorf("report generation failed: %w", err)
    }
    
    return report, nil
}
```

This comprehensive guide demonstrates how to build sophisticated multi-agent systems using GAI's advanced capabilities. The patterns and examples provided can be adapted and combined to create powerful AI systems that can handle complex, real-world scenarios requiring coordination between multiple specialized agents.