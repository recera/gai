// Package main demonstrates structured output generation with the GAI framework.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/recera/gai/core"
	"github.com/recera/gai/middleware"
	"github.com/recera/gai/providers/openai"
)

// Define structured types for our examples

// Recipe represents a cooking recipe
type Recipe struct {
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	PrepTime     int          `json:"prep_time_minutes"`
	CookTime     int          `json:"cook_time_minutes"`
	Servings     int          `json:"servings"`
	Ingredients  []Ingredient `json:"ingredients"`
	Instructions []string     `json:"instructions"`
	Tags         []string     `json:"tags"`
}

// Ingredient represents a recipe ingredient
type Ingredient struct {
	Item     string  `json:"item"`
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`
}

// TodoList represents a structured todo list
type TodoList struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Priority    string     `json:"priority"` // high, medium, low
	Tasks       []TodoTask `json:"tasks"`
	DueDate     string     `json:"due_date,omitempty"`
}

// TodoTask represents a single task
type TodoTask struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"` // pending, in_progress, completed
	Priority    int    `json:"priority"` // 1-5
}

// BusinessAnalysis represents a structured business analysis
type BusinessAnalysis struct {
	CompanyName     string     `json:"company_name"`
	Industry        string     `json:"industry"`
	MarketPosition  string     `json:"market_position"`
	Strengths       []string   `json:"strengths"`
	Weaknesses      []string   `json:"weaknesses"`
	Opportunities   []string   `json:"opportunities"`
	Threats         []string   `json:"threats"`
	Competitors     []string   `json:"competitors"`
	Recommendations []Recommendation `json:"recommendations"`
	Score           float64    `json:"overall_score"` // 0-100
}

// Recommendation represents a business recommendation
type Recommendation struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Impact      string `json:"impact"` // high, medium, low
	Timeframe   string `json:"timeframe"` // immediate, short-term, long-term
}

// CodeReview represents a structured code review
type CodeReview struct {
	Summary       string         `json:"summary"`
	OverallRating int            `json:"overall_rating"` // 1-10
	Issues        []Issue        `json:"issues"`
	Improvements  []Improvement  `json:"improvements"`
	BestPractices []string       `json:"best_practices_followed"`
}

// Issue represents a code issue
type Issue struct {
	Severity    string `json:"severity"` // critical, major, minor
	Category    string `json:"category"` // bug, security, performance, style
	Description string `json:"description"`
	Line        int    `json:"line,omitempty"`
	Suggestion  string `json:"suggestion"`
}

// Improvement represents a suggested improvement
type Improvement struct {
	Category    string `json:"category"`
	Description string `json:"description"`
	Example     string `json:"example,omitempty"`
}

func main() {
	// Check for API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set the OPENAI_API_KEY environment variable")
	}

	// Create the OpenAI provider
	var provider core.Provider = openai.New(
		openai.WithAPIKey(apiKey),
		openai.WithModel("gpt-4o-mini"),
	)

	// Apply middleware for production readiness
	provider = middleware.Chain(
		middleware.WithRetry(middleware.RetryOpts{
			MaxAttempts: 3,
			BaseDelay:   time.Second,
			MaxDelay:    10 * time.Second,
			Jitter:      true,
		}),
		middleware.WithRateLimit(middleware.RateLimitOpts{
			RPS:   5,
			Burst: 10,
		}),
	)(provider)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Example 1: Generate a recipe
	fmt.Println("=== Example 1: Generate a Recipe ===\n")
	recipeExample(ctx, provider)

	// Example 2: Generate a todo list
	fmt.Println("\n=== Example 2: Generate a Todo List ===\n")
	todoExample(ctx, provider)

	// Example 3: Generate a business analysis
	fmt.Println("\n=== Example 3: Generate Business Analysis ===\n")
	businessExample(ctx, provider)

	// Example 4: Generate a code review
	fmt.Println("\n=== Example 4: Generate Code Review ===\n")
	codeReviewExample(ctx, provider)
}

func recipeExample(ctx context.Context, provider core.Provider) {
	request := core.Request{
		Messages: []core.Message{
			{
				Role: core.System,
				Parts: []core.Part{
					core.Text{Text: "You are a professional chef. Generate detailed, authentic recipes in JSON format."},
				},
			},
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Create a recipe for vegetarian pad thai that serves 4 people."},
				},
			},
		},
	}

	result, err := provider.GenerateObject(ctx, request, Recipe{})
	if err != nil {
		log.Printf("Error generating recipe: %v", err)
		return
	}

	recipe, ok := result.Value.(*Recipe)
	if !ok {
		log.Printf("Unexpected type: %T", result.Value)
		return
	}

	fmt.Printf("Recipe: %s\n", recipe.Name)
	fmt.Printf("Description: %s\n", recipe.Description)
	fmt.Printf("Prep Time: %d minutes, Cook Time: %d minutes\n", recipe.PrepTime, recipe.CookTime)
	fmt.Printf("Servings: %d\n\n", recipe.Servings)

	fmt.Println("Ingredients:")
	for _, ing := range recipe.Ingredients {
		fmt.Printf("  - %.1f %s %s\n", ing.Quantity, ing.Unit, ing.Item)
	}

	fmt.Println("\nInstructions:")
	for i, instruction := range recipe.Instructions {
		fmt.Printf("  %d. %s\n", i+1, instruction)
	}

	fmt.Printf("\nTags: %v\n", recipe.Tags)
	
	// Also show the raw JSON
	jsonData, _ := json.MarshalIndent(recipe, "", "  ")
	fmt.Printf("\nRaw JSON:\n%s\n", string(jsonData))
}

func todoExample(ctx context.Context, provider core.Provider) {
	request := core.Request{
		Messages: []core.Message{
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Create a structured todo list for launching a new mobile app. Include development, marketing, and launch tasks."},
				},
			},
		},
	}

	result, err := provider.GenerateObject(ctx, request, TodoList{})
	if err != nil {
		log.Printf("Error generating todo list: %v", err)
		return
	}

	todoList, ok := result.Value.(*TodoList)
	if !ok {
		log.Printf("Unexpected type: %T", result.Value)
		return
	}

	fmt.Printf("📋 %s\n", todoList.Title)
	fmt.Printf("Description: %s\n", todoList.Description)
	fmt.Printf("Priority: %s\n", todoList.Priority)
	if todoList.DueDate != "" {
		fmt.Printf("Due Date: %s\n", todoList.DueDate)
	}

	fmt.Println("\nTasks:")
	for _, task := range todoList.Tasks {
		status := "⬜"
		if task.Status == "completed" {
			status = "✅"
		} else if task.Status == "in_progress" {
			status = "🔄"
		}
		
		priority := ""
		for i := 0; i < task.Priority; i++ {
			priority += "⭐"
		}
		
		fmt.Printf("%s [%d] %s %s\n", status, task.ID, task.Title, priority)
		fmt.Printf("     %s\n", task.Description)
	}

	fmt.Printf("\nUsage: %d tokens\n", result.Usage.TotalTokens)
}

func businessExample(ctx context.Context, provider core.Provider) {
	request := core.Request{
		Messages: []core.Message{
			{
				Role: core.System,
				Parts: []core.Part{
					core.Text{Text: "You are a business analyst. Provide thorough, data-driven analyses."},
				},
			},
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: "Analyze a fictional sustainable energy startup called 'SolarFlow' that provides residential solar panel installation and maintenance services."},
				},
			},
		},
		Temperature: 0.3, // Lower temperature for more consistent analysis
	}

	result, err := provider.GenerateObject(ctx, request, BusinessAnalysis{})
	if err != nil {
		log.Printf("Error generating business analysis: %v", err)
		return
	}

	analysis, ok := result.Value.(*BusinessAnalysis)
	if !ok {
		log.Printf("Unexpected type: %T", result.Value)
		return
	}

	fmt.Printf("🏢 Company: %s\n", analysis.CompanyName)
	fmt.Printf("Industry: %s\n", analysis.Industry)
	fmt.Printf("Market Position: %s\n", analysis.MarketPosition)
	fmt.Printf("Overall Score: %.1f/100\n\n", analysis.Score)

	fmt.Println("SWOT Analysis:")
	fmt.Println("Strengths:")
	for _, s := range analysis.Strengths {
		fmt.Printf("  ✓ %s\n", s)
	}

	fmt.Println("\nWeaknesses:")
	for _, w := range analysis.Weaknesses {
		fmt.Printf("  ✗ %s\n", w)
	}

	fmt.Println("\nOpportunities:")
	for _, o := range analysis.Opportunities {
		fmt.Printf("  ↗ %s\n", o)
	}

	fmt.Println("\nThreats:")
	for _, t := range analysis.Threats {
		fmt.Printf("  ⚠ %s\n", t)
	}

	fmt.Printf("\nKey Competitors: %v\n", analysis.Competitors)

	fmt.Println("\nRecommendations:")
	for i, rec := range analysis.Recommendations {
		impact := "●"
		if rec.Impact == "high" {
			impact = "●●●"
		} else if rec.Impact == "medium" {
			impact = "●●"
		}
		
		fmt.Printf("%d. %s [%s] %s\n", i+1, rec.Title, rec.Timeframe, impact)
		fmt.Printf("   %s\n", rec.Description)
	}
}

func codeReviewExample(ctx context.Context, provider core.Provider) {
	// Sample code to review
	sampleCode := `
func calculateTotal(items []Item) float64 {
    var total float64
    for i := 0; i < len(items); i++ {
        total = total + items[i].Price * float64(items[i].Quantity)
    }
    return total
}

func getUserById(id string) *User {
    db := database.Connect()
    query := "SELECT * FROM users WHERE id = '" + id + "'"
    result := db.Query(query)
    if result == nil {
        return nil
    }
    return result[0]
}
`

	request := core.Request{
		Messages: []core.Message{
			{
				Role: core.System,
				Parts: []core.Part{
					core.Text{Text: "You are a senior software engineer conducting a code review. Be thorough but constructive."},
				},
			},
			{
				Role: core.User,
				Parts: []core.Part{
					core.Text{Text: fmt.Sprintf("Review this Go code and provide structured feedback:\n```go%s```", sampleCode)},
				},
			},
		},
	}

	result, err := provider.GenerateObject(ctx, request, CodeReview{})
	if err != nil {
		log.Printf("Error generating code review: %v", err)
		return
	}

	review, ok := result.Value.(*CodeReview)
	if !ok {
		log.Printf("Unexpected type: %T", result.Value)
		return
	}

	fmt.Printf("📝 Code Review Summary\n")
	fmt.Printf("Overall Rating: %d/10\n", review.OverallRating)
	fmt.Printf("Summary: %s\n\n", review.Summary)

	if len(review.Issues) > 0 {
		fmt.Println("Issues Found:")
		for _, issue := range review.Issues {
			severity := "ℹ"
			if issue.Severity == "critical" {
				severity = "🔴"
			} else if issue.Severity == "major" {
				severity = "🟡"
			} else if issue.Severity == "minor" {
				severity = "🔵"
			}
			
			fmt.Printf("%s [%s/%s] %s\n", severity, issue.Severity, issue.Category, issue.Description)
			if issue.Line > 0 {
				fmt.Printf("   Line %d\n", issue.Line)
			}
			fmt.Printf("   Suggestion: %s\n", issue.Suggestion)
		}
	}

	if len(review.Improvements) > 0 {
		fmt.Println("\nSuggested Improvements:")
		for _, imp := range review.Improvements {
			fmt.Printf("  • [%s] %s\n", imp.Category, imp.Description)
			if imp.Example != "" {
				fmt.Printf("    Example: %s\n", imp.Example)
			}
		}
	}

	if len(review.BestPractices) > 0 {
		fmt.Println("\nBest Practices Followed:")
		for _, bp := range review.BestPractices {
			fmt.Printf("  ✓ %s\n", bp)
		}
	}
}