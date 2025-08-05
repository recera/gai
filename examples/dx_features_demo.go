package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/collinshill/gai"
)

// CodeReview represents a structured code review response
type CodeReview struct {
	Issues      []Issue  `json:"issues"`
	Suggestions []string `json:"suggestions"`
	Score       int      `json:"score"`
	Summary     string   `json:"summary"`
}

type Issue struct {
	Line        int    `json:"line"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

func main() {
	// Example 1: Simple typed action with fluent builder
	example1SimpleAction()
	
	// Example 2: Template-based prompts
	example2Templates()
	
	// Example 3: Conversation management
	example3ConversationManagement()
	
	// Example 4: Agent workflow with tools
	example4AgentWorkflow()
	
	// Example 5: Token management
	example5TokenManagement()
}

func example1SimpleAction() {
	fmt.Println("=== Example 1: Simple Typed Action ===")
	
	// Create client with options
	client, err := gai.NewClient(
		gai.WithHTTPTimeout(60*time.Second),
		gai.WithDefaultProvider("openai"),
		gai.WithDefaultModel("gpt-4o-mini"),
	)
	if err != nil {
		log.Fatal(err)
	}
	
	// Create a typed action for code review
	codeReviewAction := gai.NewAction[CodeReview]().
		WithSystem("You are an expert code reviewer. Analyze the given code and provide structured feedback.").
		WithUserMessage(`Please review this Go function:
func processData(data []string) {
    for i := 0; i < len(data); i++ {
        fmt.Println(data[i])
    }
}`)
	
	// Execute the action
	ctx := context.Background()
	review, err := codeReviewAction.Run(ctx, client)
	if err != nil {
		fmt.Printf("Code review failed: %v\n", err)
		return
	}
	
	fmt.Printf("Code Review Score: %d/100\n", review.Score)
	fmt.Printf("Summary: %s\n", review.Summary)
	fmt.Printf("Found %d issues\n", len(review.Issues))
}

func example2Templates() {
	fmt.Println("\n=== Example 2: Template-Based Prompts ===")
	
	// Create a reusable template for code analysis
	analysisTemplate, err := gai.NewPromptTemplate(`
You are analyzing {{.Language}} code.
Project: {{.ProjectName}}
Focus Areas:
{{range .FocusAreas}}- {{.}}
{{end}}
Please provide detailed analysis focusing on the specified areas.
`)
	if err != nil {
		log.Fatal(err)
	}
	
	// Use the template
	data := map[string]interface{}{
		"Language":    "Go",
		"ProjectName": "E-commerce API",
		"FocusAreas":  []string{"Security", "Performance", "Error Handling"},
	}
	
	parts := gai.NewLLMCallParts()
	parts.WithProvider("anthropic").
		WithModel("claude-3-haiku")
	
	// Use the template to render the system message
	if err := gai.RenderSystemTemplate(&parts, analysisTemplate, data); err != nil {
		log.Fatal(err)
	}
	parts.WithUserMessage("Review the authentication middleware")
	
	fmt.Printf("System prompt from template:\n%s\n", parts.System.GetTextContent())
}

func example3ConversationManagement() {
	fmt.Println("\n=== Example 3: Conversation Management ===")
	
	// Build a conversation
	conv := gai.NewLLMCallParts()
	conv.WithSystem("You are a helpful coding tutor.").
		WithUserMessage("What is a goroutine?").
		WithAssistantMessage("A goroutine is a lightweight thread managed by the Go runtime...").
		WithUserMessage("How do I create one?").
		WithAssistantMessage("You create a goroutine using the 'go' keyword...").
		WithUserMessage("Can you show an example?")
	
	// Demonstrate conversation utilities
	fmt.Printf("Total messages: %d\n", conv.CountMessages())
	lastUserMsg, _ := conv.GetLastUserMessage()
	fmt.Printf("Last user message: %s\n", lastUserMsg)
	
	// Filter to only user messages
	userOnly := conv.FilterMessages(func(m gai.Message) bool {
		return m.Role == "user"
	})
	fmt.Printf("Filtered to %d user messages\n", len(userOnly))
	
	// Generate transcript
	transcript := conv.Transcript()
	fmt.Printf("\nTranscript preview (first 200 chars):\n%s...\n", transcript[:min(200, len(transcript))])
}

func example4AgentWorkflow() {
	fmt.Println("\n=== Example 4: Agent Workflow ===")
	
	// Simulate an agent that can use tools
	agent := gai.NewLLMCallParts()
	agent.WithProvider("openai").
		WithModel("gpt-4o").
		WithSystem("You are an AI assistant that can search documentation and write code.").
		WithUserMessage("Create a function to validate email addresses in Go")
	
	// Simulate tool calls
	agent.WithMessage(gai.NewToolRequestMessage("search_docs", `{"query": "Go email validation regex"}`))
	agent.WithMessage(gai.NewToolResponseMessage("search_docs", `{"results": ["Use regexp package", "Email regex pattern: ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"]}`))
	
	// Continue conversation after tool use
	agent.WithAssistantMessage("Based on the search results, here's a function to validate email addresses:\n\n" +
		"```go\n" +
		"func isValidEmail(email string) bool {\n" +
		"    pattern := \"^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\\\.[a-zA-Z]{2,}$\"\n" +
		"    matched, _ := regexp.MatchString(pattern, email)\n" +
		"    return matched\n" +
		"}\n" +
		"```")
	
	fmt.Printf("Agent conversation has %d messages including %d tool interactions\n", 
		len(agent.Messages), 2)
}

func example5TokenManagement() {
	fmt.Println("\n=== Example 5: Token Management ===")
	
	// Create a long conversation
	conv := gai.NewLLMCallParts()
	conv.WithSystem("You are a technical documentation assistant.")
	
	// Add many messages to simulate a long conversation
	for i := 0; i < 50; i++ {
		conv.WithUserMessage(fmt.Sprintf("Explain concept %d in detail", i))
		conv.WithAssistantMessage(fmt.Sprintf("Concept %d is a complex topic that involves many aspects...", i))
	}
	
	fmt.Printf("Original conversation: %d messages\n", len(conv.Messages))
	
	// Estimate tokens
	tokenizer := gai.NewSimpleTokenizer()
	tokens := conv.EstimateTokens(tokenizer)
	fmt.Printf("Estimated tokens: %d\n", tokens)
	
	// Prune to fit in context window
	contextLimit := 4000
	removed, err := conv.PruneToTokens(contextLimit, tokenizer)
	if err != nil {
		fmt.Printf("Pruning error: %v\n", err)
	} else {
		fmt.Printf("Removed %d messages to fit in %d tokens\n", removed, contextLimit)
		fmt.Printf("Remaining messages: %d\n", len(conv.Messages))
		fmt.Printf("New token count: %d\n", conv.EstimateTokens(tokenizer))
	}
	
	// Keep only recent messages
	conv2 := conv.Clone()
	conv2.KeepLastMessages(10)
	fmt.Printf("\nAfter keeping last 10 messages: %d messages, %d tokens\n", 
		len(conv2.Messages), conv2.EstimateTokens(tokenizer))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}