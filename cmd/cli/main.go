package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/morgansundqvist/muserstory/internal/adapters"
	"github.com/morgansundqvist/muserstory/internal/application"
	"github.com/spf13/cobra"
)

type ctxKey string

const svcKey ctxKey = "userStoryService"

func main() {
	var filePath string

	rootCmd := &cobra.Command{
		Use:   "muserstory",
		Short: "Manage user stories with LLM support",
		Long:  "A CLI tool to categorize, add, list, summarize, and generate user stories using an LLM service.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if filePath == "" {
				cmd.Println("Error: markdown file path must be provided with --file flag")
				return fmt.Errorf("missing required flag: --file")
			}
			llmAPI := adapters.NewOpenAILLMService()
			svc := application.NewUserStoryService(llmAPI, filePath)
			existingCtx := cmd.Context()
			ctx := context.WithValue(existingCtx, svcKey, svc)
			cmd.SetContext(ctx)
			return nil
		},
	}

	rootCmd.PersistentFlags().StringVarP(&filePath, "file", "f", "userstories.md", "Path to the markdown file containing user stories.")

	rootCmd.AddCommand(categorizeCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(summarizeCmd)
	rootCmd.AddCommand(generateCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var categorizeCmd = &cobra.Command{
	Use:   "categorize",
	Short: "Categorize all user stories in the file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("'categorize' takes no arguments")
		}
		svc := cmd.Context().Value(svcKey).(*application.UserStoryService)
		file := cmd.Flag("file").Value.String()
		fmt.Printf("Starting categorization for stories in %s...\n", file)
		if err := svc.CategorizeAllStories(); err != nil {
			return err
		}
		fmt.Println("Categorization process complete.")
		return nil
	},
}

var addCmd = &cobra.Command{
	Use:   "add [story]",
	Short: "Add a new user story to the file",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		story := strings.Join(args, " ")
		svc := cmd.Context().Value(svcKey).(*application.UserStoryService)
		file := cmd.Flag("file").Value.String()
		fmt.Printf("Adding story to %s: \"%s\"\n", file, story)
		return svc.AddUserStory(story)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all user stories from the file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("'list' takes no arguments")
		}
		svc := cmd.Context().Value(svcKey).(*application.UserStoryService)
		file := cmd.Flag("file").Value.String()
		fmt.Printf("Listing stories from %s...\n", file)
		return svc.ListUserStories()
	},
}

var summarizeCmd = &cobra.Command{
	Use:   "summarize",
	Short: "Generate and save a summary of all user stories",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("'summarize' takes no arguments")
		}
		svc := cmd.Context().Value(svcKey).(*application.UserStoryService)
		file := cmd.Flag("file").Value.String()
		fmt.Printf("Starting summarization for stories in %s...\n", file)
		return svc.SummarizeStories()
	},
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate new user stories based on existing ones",
	RunE: func(cmd *cobra.Command, args []string) error {
		n, err := cmd.Flags().GetInt("num")
		if err != nil {
			return err
		}
		if n <= 0 {
			return fmt.Errorf("number of stories must be positive")
		}
		svc := cmd.Context().Value(svcKey).(*application.UserStoryService)
		file := cmd.Flag("file").Value.String()
		fmt.Printf("Starting generation of %d new stories for %s...\n", n, file)
		return svc.GenerateNewStories(n)
	},
}

func init() {
	generateCmd.Flags().IntP("num", "n", 1, "Number of user stories to generate")
}
