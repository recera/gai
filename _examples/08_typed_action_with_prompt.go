package main

import (
	"context"
	"fmt"
	"log"

	"github.com/recera/gai"
)

// A typed response struct with desc tags used for instructions and tool schemas
type CodeReview struct {
	Summary string   `json:"summary" desc:"Short summary of issues and strengths"`
	Issues  []string `json:"issues" desc:"List of issues found"`
	Score   int      `json:"score" desc:"Overall score 1-10"`
}

func main() {
	// Load .env from repo root if present
	gai.FindAndLoadEnv()

	client, err := gai.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	// Build a prompt from a file template and append JSON formatting instructions
	prompt, err := gai.BuildActionPrompt("_examples/prompts/code_review.txt", CodeReview{})
	if err != nil {
		log.Fatal(err)
	}

	action := gai.NewAction[CodeReview]().
		WithProvider("openai").
		WithModel("gpt-4o-mini").
		WithSystem(prompt).
		WithUserMessage(`package main\nfunc sum(a, b int) int { return a + b }`)

	ctx := context.Background()
	review, err := action.Run(ctx, client)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Score: %d\nSummary: %s\nIssues: %v\n", review.Score, review.Summary, review.Issues)
}
