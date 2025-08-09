package main

import (
	"context"
	"fmt"
	"log"

	"github.com/recera/gai"
)

// Typed struct we want back via tool-call arguments
type Book struct {
	Title  string `json:"title" desc:"Book title"`
	Author string `json:"author" desc:"Book author"`
	Year   int    `json:"year" desc:"Publication year"`
}

func main() {
	gai.FindAndLoadEnv()
	client, err := gai.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	parts := gai.NewLLMCallParts().
		WithProvider("openai").
		WithModel("gpt-4o-mini").
		WithSystem("Call the tool to provide structured book details.").
		WithUserMessage("Recommend one sci-fi book.")

	var out Book
	if err := gai.GetResponseObjectViaTools(context.Background(), client, parts.Value(), "recommend_book", &out, gai.ToolGenOptions{
		Description: "Provide a single book recommendation",
		Doc:         "Fill title, author, and year.",
	}); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Book: %s by %s (%d)\n", out.Title, out.Author, out.Year)
}
