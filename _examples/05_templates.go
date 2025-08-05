package main

import (
	"context"
	"fmt"
	"log"

	"github.com/collinshill/gai"
)

// Code review structure
type CodeReview struct {
	Issues []Issue `json:"issues" desc:"List of issues found"`
	Score  int     `json:"score" desc:"Code quality score 1-10"`
	Summary string `json:"summary" desc:"Overall assessment"`
}

type Issue struct {
	Type        string `json:"type" desc:"bug, style, performance, security"`
	Line        int    `json:"line" desc:"Line number (0 if general)"`
	Severity    string `json:"severity" desc:"low, medium, high"`
	Description string `json:"description" desc:"What's wrong"`
	Suggestion  string `json:"suggestion" desc:"How to fix it"`
}

func main() {
	// Load environment
	gai.FindAndLoadEnv()

	// Create client
	client, err := gai.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	// Create a code review template
	reviewTemplate := `You are an expert {{.Language}} code reviewer.
Please review the following {{.Language}} code for:
- Bugs and potential issues
- Code style and best practices
- Performance concerns
- Security vulnerabilities

Focus areas: {{range .FocusAreas}}{{.}}, {{end}}

Code to review:
{{.Code}}`

	// Parse template
	tmpl, err := gai.NewPromptTemplate(reviewTemplate)
	if err != nil {
		log.Fatal(err)
	}

	// Sample code to review
	sampleCode := `func authenticateUser(username, password string) bool {
    query := "SELECT * FROM users WHERE username='" + username + 
             "' AND password='" + password + "'"
    
    rows, err := db.Query(query)
    if err != nil {
        fmt.Println(err)
        return false
    }
    
    return rows.Next()
}`

	// Create action with template
	action := gai.NewAction[CodeReview]().
		WithProvider("openai").
		WithModel("gpt-4o")

	// Render template into system message
	templateData := map[string]interface{}{
		"Language": "Go",
		"FocusAreas": []string{
			"SQL injection vulnerabilities",
			"Error handling",
			"Security best practices",
		},
		"Code": sampleCode,
	}

	if err := gai.RenderSystemTemplate(action.GetParts(), tmpl, templateData); err != nil {
		log.Fatal(err)
	}

	// Add user message
	action.WithUserMessage("Please review this code and provide detailed feedback.")

	// Get structured review
	ctx := context.Background()
	review, err := action.Run(ctx, client)
	if err != nil {
		log.Fatal(err)
	}

	// Display results
	fmt.Printf("Code Review Results\n")
	fmt.Printf("==================\n")
	fmt.Printf("Score: %d/10\n", review.Score)
	fmt.Printf("Summary: %s\n\n", review.Summary)

	if len(review.Issues) > 0 {
		fmt.Printf("Issues Found:\n")
		for i, issue := range review.Issues {
			fmt.Printf("\n%d. [%s] %s (Severity: %s)\n", 
				i+1, issue.Type, issue.Description, issue.Severity)
			if issue.Line > 0 {
				fmt.Printf("   Line: %d\n", issue.Line)
			}
			fmt.Printf("   Suggestion: %s\n", issue.Suggestion)
		}
	}
}