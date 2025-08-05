package main

import (
	"context"
	"fmt"
	"log"

	"github.com/recera/gai"
)

// Define your response structure
type BookRecommendation struct {
	Title       string   `json:"title" desc:"Book title"`
	Author      string   `json:"author" desc:"Book author"`
	Year        int      `json:"year" desc:"Publication year"`
	Genre       string   `json:"genre" desc:"Primary genre"`
	Summary     string   `json:"summary" desc:"Brief plot summary"`
	WhyRecommend string  `json:"why_recommend" desc:"Why this book matches the request"`
	Rating      float64  `json:"rating" desc:"Rating out of 5"`
}

func main() {
	// Load environment
	gai.FindAndLoadEnv()

	// Create client
	client, err := gai.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	// Create a type-safe action
	action := gai.NewAction[BookRecommendation]().
		WithProvider("openai").
		WithModel("gpt-4o-mini").
		WithSystem("You are a knowledgeable book recommendation assistant.").
		WithUserMessage("Recommend a science fiction book about AI")

	// Execute and get typed response
	ctx := context.Background()
	book, err := action.Run(ctx, client)
	if err != nil {
		log.Fatal(err)
	}

	// Use the structured response
	fmt.Printf("📚 Book Recommendation:\n")
	fmt.Printf("Title: %s\n", book.Title)
	fmt.Printf("Author: %s\n", book.Author)
	fmt.Printf("Year: %d\n", book.Year)
	fmt.Printf("Genre: %s\n", book.Genre)
	fmt.Printf("Rating: %.1f/5\n", book.Rating)
	fmt.Printf("\nSummary: %s\n", book.Summary)
	fmt.Printf("\nWhy this book: %s\n", book.WhyRecommend)
}