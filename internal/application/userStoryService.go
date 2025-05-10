package application

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort" // Added import for sorting category names
	"strings"

	"github.com/morgansundqvist/muserstory/internal/domain"
	"github.com/morgansundqvist/muserstory/internal/ports"
)

// UserStoryService provides methods to manage user stories.
type UserStoryService struct {
	llmService ports.LLMService
	filePath   string
}

// NewUserStoryService creates a new UserStoryService.
func NewUserStoryService(llmService ports.LLMService, filePath string) *UserStoryService {
	return &UserStoryService{
		llmService: llmService,
		filePath:   filePath,
	}
}

// generateID creates a simple unique ID for new stories.
func generateID() string {
	bytes := make([]byte, 8) // Creates a 16-character hex string
	if _, err := rand.Read(bytes); err != nil {
		// Fallback for extremely rare error case
		return fmt.Sprintf("errID-%d", os.Getpid())
	}
	return hex.EncodeToString(bytes)
}

// ReadUserStoriesFromFile reads a potential summary and user stories from the file.
// The summary is expected at the top, starting with "# Summary" on its own line,
// followed by the summary content. The summary ends before the first story item ("- ")
// or category header ("**").
func (s *UserStoryService) ReadUserStoriesFromFile() (string, []domain.UserStory, error) {
	file, err := os.Open(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", []domain.UserStory{}, nil // No file, so no summary and no stories
		}
		return "", nil, fmt.Errorf("error opening file %s: %w", s.filePath, err)
	}
	defer file.Close()

	var stories []domain.UserStory
	var summaryBuilder strings.Builder
	var storyContentLines []string // Lines that are not part of the summary

	scanner := bufio.NewScanner(file)

	var allFileLines []string
	for scanner.Scan() {
		allFileLines = append(allFileLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return "", nil, fmt.Errorf("error reading all lines from file %s: %w", s.filePath, err)
	}

	isReadingSummary := false
	currentLineIndex := 0

	if len(allFileLines) > 0 && strings.TrimSpace(allFileLines[0]) == "# Summary" {
		isReadingSummary = true
		currentLineIndex = 1 // Start processing from the line after "# Summary"
	}

	for ; currentLineIndex < len(allFileLines); currentLineIndex++ {
		line := allFileLines[currentLineIndex]
		trimmedLine := strings.TrimSpace(line)

		if isReadingSummary {
			// Check if the current line marks the end of the summary
			if strings.HasPrefix(trimmedLine, "- ") || strings.HasPrefix(trimmedLine, "**") {
				isReadingSummary = false                            // Summary section ended
				storyContentLines = append(storyContentLines, line) // This line is the start of stories
			} else {
				// This line is part of the summary content
				if summaryBuilder.Len() > 0 {
					summaryBuilder.WriteString("\n") // Add newline separator for multi-line summaries
				}
				summaryBuilder.WriteString(line)
			}
		} else {
			// Not in summary mode (either never started or already ended)
			storyContentLines = append(storyContentLines, line)
		}
	}

	// Now parse the storyContentLines
	lineNumber := 0 // Line number relative to the start of story content
	for _, line := range storyContentLines {
		lineNumber++
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "- ") {
			content := strings.TrimPrefix(trimmedLine, "- ")
			description := content
			category := "Uncategorized" // Default

			catStartIndex := strings.LastIndex(content, "[Category: ")
			catEndIndex := strings.LastIndex(content, "]")

			if catStartIndex != -1 && catEndIndex != -1 && catEndIndex > catStartIndex && catEndIndex == len(content)-1 {
				description = strings.TrimSpace(content[:catStartIndex])
				category = content[catStartIndex+len("[Category: ") : catEndIndex]
			}

			stories = append(stories, domain.UserStory{
				ID:          fmt.Sprintf("%s-%d", generateID(), lineNumber), // ID generation remains non-persistent
				Description: description,
				Category:    category,
			})
		}
	}

	finalSummary := strings.TrimSpace(summaryBuilder.String())
	return finalSummary, stories, nil
}

// WriteUserStoriesToFile writes the given summary and user stories to the markdown file,
// overwriting its current content. Summary is written first, then stories grouped by category.
func (s *UserStoryService) WriteUserStoriesToFile(summary string, stories []domain.UserStory) error {
	file, err := os.OpenFile(s.filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("error opening/creating file %s for writing: %w", s.filePath, err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	if summary != "" {
		if _, err := writer.WriteString("# Summary\n"); err != nil {
			return fmt.Errorf("error writing summary header: %w", err)
		}
		summaryToWrite := strings.TrimSpace(summary)
		if summaryToWrite != "" {
			if _, err := writer.WriteString(summaryToWrite + "\n"); err != nil {
				return fmt.Errorf("error writing summary content: %w", err)
			}
		}
		if _, err := writer.WriteString("\n"); err != nil { // Extra blank line after summary section
			return fmt.Errorf("error writing newline after summary section: %w", err)
		}
	}

	if len(stories) == 0 {
		// If there are no stories, ensure the file is empty and return.
		return writer.Flush()
	}

	storiesByCategory := make(map[string][]domain.UserStory)
	var categoryOrder []string

	for _, story := range stories {
		catToUse := story.Category
		if catToUse == "" {
			catToUse = "Uncategorized"
		}
		if _, exists := storiesByCategory[catToUse]; !exists {
			categoryOrder = append(categoryOrder, catToUse) // Keep track of categories in order of appearance or sort later
		}
		storiesByCategory[catToUse] = append(storiesByCategory[catToUse], story)
	}

	// Sort categories alphabetically for consistent output
	sort.Strings(categoryOrder)

	for _, categoryName := range categoryOrder {
		if _, err := writer.WriteString(fmt.Sprintf("**%s**\n", categoryName)); err != nil {
			return fmt.Errorf("error writing category header %s: %w", categoryName, err)
		}

		categoryStories := storiesByCategory[categoryName]
		for _, story := range categoryStories {
			// Ensure category in the tag matches the group, or use story.Category if it's more specific
			// For simplicity, we'll use story.Category which should be consistent with categoryName here.
			catToPrintInTag := story.Category
			if catToPrintInTag == "" {
				catToPrintInTag = "Uncategorized"
			}
			line := fmt.Sprintf("- %s [Category: %s]\n", story.Description, catToPrintInTag)
			if _, err := writer.WriteString(line); err != nil {
				return fmt.Errorf("error writing story (ID: %s) to file: %w", story.ID, err)
			}
		}
		if _, err := writer.WriteString("\n"); err != nil { // Add a blank line between categories
			return fmt.Errorf("error writing newline separator: %w", err)
		}
	}

	return writer.Flush()
}

// AddUserStory adds a new user story, auto-categorizes it, and saves it to the file.
func (s *UserStoryService) AddUserStory(description string) error {
	currentSummary, stories, err := s.ReadUserStoriesFromFile()
	if err != nil {
		return fmt.Errorf("could not read existing stories: %w", err)
	}

	newStory := domain.UserStory{
		ID:          generateID(),
		Description: description,
		Category:    "Uncategorized", // Initial category before LLM call
	}

	llmInput := domain.LLMSimpleInput{
		SystemMessage: "Categorize the following user story. Only return the category name.",
		UserMessage:   newStory.Description,
		ModelType:     domain.ModelTypeSimple,
	}

	category, err := s.llmService.AskSimple(llmInput)

	// Attempt to categorize the new story using the LLM service
	if err != nil {
		return fmt.Errorf("could not categorize new story: %w", err)
	}
	category = strings.TrimSpace(category)
	if category == "" {
		category = "Uncategorized" // Fallback if LLM returns an empty category
	}

	newStory.Category = category

	stories = append(stories, newStory)

	err = s.WriteUserStoriesToFile(currentSummary, stories)
	if err != nil {
		return fmt.Errorf("could not write updated stories to file: %w", err)
	}
	fmt.Printf("User story added: \"%s\" [Category: %s]\n", newStory.Description, newStory.Category)
	return nil
}

// CategorizeAllStories reads all stories, re-categorizes them using the LLM service,
// and writes them back to the file.
func (s *UserStoryService) CategorizeAllStories() error {
	currentSummary, stories, err := s.ReadUserStoriesFromFile()
	if err != nil {
		return fmt.Errorf("could not read stories for categorization: %w", err)
	}

	if len(stories) == 0 {
		fmt.Println("No stories found in the file to categorize.")
		// Preserve existing summary even if there are no stories
		err = s.WriteUserStoriesToFile(currentSummary, []domain.UserStory{})
		if err != nil {
			return fmt.Errorf("could not write file with current summary and no stories: %w", err)
		}
		return nil
	}

	possibleCategories := s.GeneratePossibleCategories(stories)

	possibleCategoriesString := strings.Join(possibleCategories, ", ")

	categorizedStories := make([]domain.UserStory, len(stories))
	for i, story := range stories {
		llmInput := domain.LLMSimpleInput{
			SystemMessage: "Categorize the following user story. Only return the category name. Possible categories are: " + possibleCategoriesString,
			UserMessage:   story.Description,
			ModelType:     domain.ModelTypeSimple,
		}
		category, err := s.llmService.AskSimple(llmInput)
		if err != nil {
			fmt.Printf("Error categorizing story ID %s ('%s'): %v. Assigning 'Uncategorized'.\n", story.ID, story.Description, err)
			categorizedStories[i] = story
			categorizedStories[i].Category = "Uncategorized"
			continue
		}
		category = strings.TrimSpace(category)
		if category == "" {
			category = "Uncategorized" // Fallback if LLM returns an empty category
		}
		categorizedStories[i] = story
		categorizedStories[i].Category = category
	}

	// Sort categorized stories by their category for better organization
	sort.Slice(categorizedStories, func(i, j int) bool {
		return categorizedStories[i].Category < categorizedStories[j].Category
	})

	// Write the categorized stories back to the file
	// If the categorization fails for some stories, we still want to write the ones that were successfully categorized.
	// This is a change from the previous version where we would only write if all stories were categorized.
	// This allows for partial success in categorization.
	// If no stories were categorized, we still want to write an empty file.
	// This is a change from the previous version where we would only write if all stories were categorized.

	err = s.WriteUserStoriesToFile(currentSummary, categorizedStories) // Write the (potentially partially) categorized stories along with the original summary
	if err != nil {
		return fmt.Errorf("could not write categorized stories to file: %w", err)
	}

	fmt.Println("User stories have been processed for categorization.")
	if len(categorizedStories) > 0 {
		fmt.Println("Current stories and their categories:")
		for _, story := range categorizedStories {
			fmt.Printf("  - \"%s\" [Category: %s]\n", story.Description, story.Category)
		}
	} else {
		fmt.Println("No stories were written back to the file after categorization attempt.")
	}
	return nil
}

// SummarizeStories generates a summary of all user stories and writes it to the file and console.
func (s *UserStoryService) SummarizeStories() error {
	// Read existing stories. The current summary is ignored as we're generating a new one.
	_, stories, err := s.ReadUserStoriesFromFile()
	if err != nil {
		return fmt.Errorf("could not read stories for summarization: %w", err)
	}

	if len(stories) == 0 {
		fmt.Println("No stories to summarize.")
		// If there are no stories, write an empty summary and empty stories list
		// to ensure the file reflects the "no stories" state, clearing any old summary.
		err = s.WriteUserStoriesToFile("", []domain.UserStory{})
		if err != nil {
			return fmt.Errorf("could not clear file when no stories to summarize: %w", err)
		}
		return nil
	}

	var storyDescriptions strings.Builder
	for i, story := range stories {
		storyDescriptions.WriteString(story.Description)
		if i < len(stories)-1 {
			// Separate descriptions for the LLM; double newline might help context separation.
			storyDescriptions.WriteString("\n\n")
		}
	}

	llmInput := domain.LLMSimpleInput{
		SystemMessage: "Please create a summary of what the project is based on the user stories which are input. Write about what is is based on the user stories but also what it could become. Do not include any preamble like 'Here is the summary:'.",
		UserMessage:   storyDescriptions.String(),
		ModelType:     domain.ModelTypeSimple,
	}

	generatedSummary, err := s.llmService.AskSimple(llmInput)
	if err != nil {
		return fmt.Errorf("could not generate summary from LLM: %w", err)
	}

	generatedSummary = strings.TrimSpace(generatedSummary)

	if generatedSummary == "" {
		fmt.Println("LLM generated an empty summary. The file will be updated with no summary or an empty summary section.")
	} else {
		fmt.Println("# Summary")
		fmt.Println(generatedSummary)
		fmt.Println("\nSummary has been generated.")
	}

	err = s.WriteUserStoriesToFile(generatedSummary, stories)
	if err != nil {
		return fmt.Errorf("could not write new summary and stories to file: %w", err)
	}

	fmt.Println("File has been updated with the new summary and existing stories.")
	return nil
}

// ListUserStories reads all user stories from the file and prints them to the console.
func (s *UserStoryService) ListUserStories() error {
	summary, stories, err := s.ReadUserStoriesFromFile()
	if err != nil {
		return fmt.Errorf("could not read stories for listing: %w", err)
	}

	if summary != "" {
		fmt.Println("# Summary")
		fmt.Println(summary)
		fmt.Println() // Add a blank line for separation before stories
	}

	if len(stories) == 0 {
		if summary == "" { // Only print "No user stories" if there was no summary either
			fmt.Println("No user stories found in the file.")
		} else {
			// If there's a summary, it's already printed.
			// We can indicate that no stories follow.
			fmt.Println("No user stories found in the file (summary is present).")
		}
		return nil
	}

	fmt.Println("User Stories:")
	for _, story := range stories {
		fmt.Printf("- %s [Category: %s]\n", story.Description, story.Category)
	}
	return nil
}

// GeneratedStoriesResponse defines the structure for LLM response when generating new stories.
type GeneratedStoriesResponse struct {
	NewUserStories []string `json:"new_user_stories" jsonschema_description:"A list of new user story descriptions."`
}

// GenerateNewStories generates a specified number of new user stories based on existing ones,
// categorizes them, and saves them to the file.
func (s *UserStoryService) GenerateNewStories(numStoriesToGenerate int) error {
	currentSummary, existingStories, err := s.ReadUserStoriesFromFile()
	if err != nil {
		return fmt.Errorf("could not read existing stories: %w", err)
	}

	var existingStoryDescriptions strings.Builder
	if len(existingStories) > 0 {
		existingStoryDescriptions.WriteString("Existing user stories for context:\n")
		for _, story := range existingStories {
			existingStoryDescriptions.WriteString(fmt.Sprintf("- %s\n", story.Description))
		}
	} else {
		existingStoryDescriptions.WriteString("There are no existing user stories. Please generate initial stories for a new project.")
	}

	schemaDef := domain.GenerateSchema[GeneratedStoriesResponse]()

	llmInput := domain.LLMAdvancedInput{
		SystemMessage:     fmt.Sprintf("Based on the provided context of existing user stories (if any), generate exactly %d new, distinct, and relevant user stories. Each story should be a single descriptive sentence, typically following a format like 'As a [user type], I want [action] so that [benefit]'.", numStoriesToGenerate),
		UserMessage:       existingStoryDescriptions.String(),
		ModelType:         domain.ModelTypeSimple, // Or consider a more advanced model if available/needed
		SchemaName:        "GenerateNewUserStories",
		Schema:            schemaDef,
		SchemaDescription: "A list of newly generated user story descriptions.",
	}

	rawResponse, err := s.llmService.AskAdvanced(llmInput)
	if err != nil {
		return fmt.Errorf("llm service failed to generate stories: %w", err)
	}

	var generatedStoriesResponse GeneratedStoriesResponse
	if err := json.Unmarshal([]byte(rawResponse), &generatedStoriesResponse); err != nil {
		return fmt.Errorf("failed to unmarshal llm response for generated stories: %w. Response was: %s", err, rawResponse)
	}

	if len(generatedStoriesResponse.NewUserStories) == 0 {
		fmt.Println("LLM did not generate any new stories.")
		return nil
	}

	fmt.Printf("LLM generated %d potential new story descriptions. Reviewing each one...\n", len(generatedStoriesResponse.NewUserStories))

	allStories := existingStories
	newlyAddedStoriesCount := 0
	reader := bufio.NewReader(os.Stdin)

	for i, storyDesc := range generatedStoriesResponse.NewUserStories {
		trimmedStoryDesc := strings.TrimSpace(storyDesc)
		if trimmedStoryDesc == "" {
			fmt.Println("Skipping empty story description generated by LLM.")
			continue
		}

		fmt.Printf("\nGenerated story %d/%d: \"%s\"\n", i+1, len(generatedStoriesResponse.NewUserStories), trimmedStoryDesc)
		fmt.Print("Keep this story? (y/n): ")
		userInput, _ := reader.ReadString('\n')
		userInput = strings.ToLower(strings.TrimSpace(userInput))

		if userInput != "y" {
			fmt.Println("Story discarded.")
			continue
		}

		fmt.Println("Story accepted. Categorizing...")
		newStory := domain.UserStory{
			ID:          generateID(),
			Description: trimmedStoryDesc,
			Category:    "Uncategorized", // Default before categorization
		}

		// Categorize the new story
		categorizationInput := domain.LLMSimpleInput{
			SystemMessage: "Categorize the following user story. Only return the category name.",
			UserMessage:   newStory.Description,
			ModelType:     domain.ModelTypeSimple,
		}
		category, catErr := s.llmService.AskSimple(categorizationInput)
		if catErr != nil {
			fmt.Printf("Could not categorize new story \"%s\": %v. Assigning 'Uncategorized'.\n", newStory.Description, catErr)
			// Keep "Uncategorized"
		} else {
			trimmedCategory := strings.TrimSpace(category)
			if trimmedCategory == "" {
				newStory.Category = "Uncategorized"
			} else {
				newStory.Category = trimmedCategory
			}
		}

		allStories = append(allStories, newStory)
		fmt.Printf("Kept and categorized: \"%s\" [Category: %s]\n", newStory.Description, newStory.Category)
		newlyAddedStoriesCount++
	}

	if newlyAddedStoriesCount == 0 {
		fmt.Println("No valid new stories were generated or processed.")
		return nil
	}

	err = s.WriteUserStoriesToFile(currentSummary, allStories)
	if err != nil {
		return fmt.Errorf("could not write updated stories to file: %w", err)
	}

	fmt.Printf("%d new user stories have been generated, categorized, and added to %s.\n", newlyAddedStoriesCount, s.filePath)
	return nil
}

type CategoryResponse struct {
	Categories []string `json:"categories" jsonschema_description:"List of possible categories for the user stories"`
}

func (s *UserStoryService) GeneratePossibleCategories(stories []domain.UserStory) []string {
	var categories []string

	//Concatenate all story descriptions into a single string
	var storyDescriptions strings.Builder
	for _, story := range stories {
		storyDescriptions.WriteString(story.Description)
		storyDescriptions.WriteString("\n")
	}

	responseInterface := domain.GenerateSchema[CategoryResponse]()

	// Ask the LLM for possible categories based on the concatenated descriptions
	llmInput := domain.LLMAdvancedInput{
		SystemMessage:     "Generate a list of possible categories based on the following user stories. Only return the category names.",
		UserMessage:       storyDescriptions.String(),
		ModelType:         domain.ModelTypeSimple,
		SchemaName:        "GeneratePossibleCategories",
		Schema:            responseInterface,
		SchemaDescription: "List of possible categories for the user stories",
	}

	categoriesResponse, err := s.llmService.AskAdvanced(llmInput)
	if err != nil {
		fmt.Printf("Error generating categories: %v\n", err)
		return nil
	}

	var categoriesResponseStruct CategoryResponse

	// Unmarshal the response into the CategoryResponse struct
	err = json.Unmarshal([]byte(categoriesResponse), &categoriesResponseStruct)
	if err != nil {
		fmt.Printf("Error unmarshalling categories response: %v\n", err)
		return nil
	}

	categories = categoriesResponseStruct.Categories

	return categories
}
