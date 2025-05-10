package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/morgansundqvist/muserstory/internal/adapters"
	"github.com/morgansundqvist/muserstory/internal/application"
)

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s -f <markdown_file_path> <command> [arguments]\n", os.Args[0])
	fmt.Fprintln(os.Stderr, "\nFlags:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "\nCommands:")
	fmt.Fprintln(os.Stderr, "  categorize          Categorize all user stories in the file using an LLM service.")
	fmt.Fprintln(os.Stderr, "  add \"<story>\"     Add a new user story to the file. It will be automatically categorized.")
	fmt.Fprintln(os.Stderr, "  list                List all user stories from the file.")
	fmt.Fprintln(os.Stderr, "  summarize           Generate and save a summary of all user stories.")
	fmt.Fprintln(os.Stderr, "  generate -n <num>   Generate <num> new user stories based on existing ones.")
	fmt.Fprintln(os.Stderr, "\nExamples:")
	fmt.Fprintf(os.Stderr, "  %s -f stories.md categorize\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -f stories.md add \"As a user, I want to log in so that I can access my account.\"\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -f stories.md list\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -f stories.md summarize\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -f stories.md generate -n 5\n", os.Args[0])
}

func main() {
	filePath := flag.String("f", "userstories.md", "Path to the markdown file containing user stories.")
	flag.Usage = printUsage 
	flag.Parse()            

	if *filePath == "" {
		fmt.Fprintln(os.Stderr, "Error: Markdown file path (-f) must be provided.")
		printUsage()
		os.Exit(1)
	}

	llmAPIService := adapters.NewOpenAILLMService()
	storyService := application.NewUserStoryService(llmAPIService, *filePath)

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: No command provided.")
		printUsage()
		os.Exit(1)
	}

	command := args[0]
	switch command {
	case "categorize":
		if len(args) != 1 {
			fmt.Fprintln(os.Stderr, "Error: 'categorize' command takes no arguments.")
			printUsage()
			os.Exit(1)
		}
		fmt.Printf("Starting categorization for stories in %s...\n", *filePath)
		err := storyService.CategorizeAllStories()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error during categorization: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Categorization process complete.")

	case "add":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Error: 'add' command requires a user story description.")
			fmt.Fprintln(os.Stderr, "Usage: ... add \"Your new user story\"")
			printUsage()
			os.Exit(1)
		}
		storyDescription := strings.Join(args[1:], " ")
		fmt.Printf("Adding story to %s: \"%s\"\n", *filePath, storyDescription)
		err := storyService.AddUserStory(storyDescription)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error adding user story: %v\n", err)
			os.Exit(1)
		}

	case "list":
		if len(args) != 1 {
			fmt.Fprintln(os.Stderr, "Error: 'list' command takes no arguments.")
			printUsage()
			os.Exit(1)
		}
		fmt.Printf("Listing stories from %s...\n", *filePath)
		err := storyService.ListUserStories()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing user stories: %v\n", err)
			os.Exit(1)
		}

	case "summarize":
		if len(args) != 1 {
			fmt.Fprintln(os.Stderr, "Error: 'summarize' command takes no arguments.")
			printUsage()
			os.Exit(1)
		}
		fmt.Printf("Starting summarization for stories in %s...\n", *filePath)
		err := storyService.SummarizeStories()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error during summarization: %v\n", err)
			os.Exit(1)
		}

	case "generate":
		generateCmd := flag.NewFlagSet("generate", flag.ExitOnError)
		numStories := generateCmd.Int("n", 1, "Number of user stories to generate")

		generateCmd.Usage = func() {
			fmt.Fprintf(os.Stderr, "Usage: %s -f <file> generate -n <number_of_stories>\n", os.Args[0])
			fmt.Fprintln(os.Stderr, "\nFlags for generate:")
			generateCmd.PrintDefaults()
		}

		if err := generateCmd.Parse(args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing flags for 'generate' command: %v\n", err)
			os.Exit(1)
		}

		if *numStories <= 0 {
			fmt.Fprintln(os.Stderr, "Error: Number of stories to generate (-n) must be positive.")
			generateCmd.Usage()
			os.Exit(1)
		}

		if len(generateCmd.Args()) > 0 {
			fmt.Fprintf(os.Stderr, "Error: 'generate' command received unexpected arguments: %v\n", generateCmd.Args())
			generateCmd.Usage()
			os.Exit(1)
		}

		fmt.Printf("Starting generation of %d new stories for %s...\n", *numStories, *filePath)
		err := storyService.GenerateNewStories(*numStories)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error during story generation: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown command '%s'\n", command)
		printUsage()
		os.Exit(1)
	}
}
