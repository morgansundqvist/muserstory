package application

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort" 
	"strings"

	"github.com/morgansundqvist/muserstory/internal/domain"
	"github.com/morgansundqvist/muserstory/internal/ports"
)

type UserStoryService struct {
	llmService ports.LLMService
	filePath   string
}

func NewUserStoryService(llmService ports.LLMService, filePath string) *UserStoryService {
	return &UserStoryService{
		llmService: llmService,
		filePath:   filePath,
	}
}

func generateID() string {
	bytes := make([]byte, 8) 
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("errID-%d", os.Getpid())
	}
	return hex.EncodeToString(bytes)
}

func (s *UserStoryService) ReadUserStoriesFromFile() (string, []domain.UserStory, error) {
	file, err := os.Open(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", []domain.UserStory{}, nil 
		}
		return "", nil, fmt.Errorf("error opening file %s: %w", s.filePath, err)
	}
	defer file.Close()

	var stories []domain.UserStory
	var summaryBuilder strings.Builder
	var storyContentLines []string 

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
		currentLineIndex = 1 
	}

	for ; currentLineIndex < len(allFileLines); currentLineIndex++ {
		line := allFileLines[currentLineIndex]
		trimmedLine := strings.TrimSpace(line)

		if isReadingSummary {
			if strings.HasPrefix(trimmedLine, "- ") || strings.HasPrefix(trimmedLine, "**") {
				isReadingSummary = false                            
				storyContentLines = append(storyContentLines, line) 
			} else {
				if summaryBuilder.Len() > 0 {
					summaryBuilder.WriteString("\n") 
				}
				summaryBuilder.WriteString(line)
			}
		} else {
			storyContentLines = append(storyContentLines, line)
		}
	}

	lineNumber := 0 
	for _, line := range storyContentLines {
		lineNumber++
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "- ") {
			content := strings.TrimPrefix(trimmedLine, "- ")
			description := content
			category := "Uncategorized" 

			catStartIndex := strings.LastIndex(content, "[Category: ")
			catEndIndex := strings.LastIndex(content, "]")

			if catStartIndex != -1 && catEndIndex != -1 && catEndIndex > catStartIndex && catEndIndex == len(content)-1 {
				description = strings.TrimSpace(content[:catStartIndex])
				category = content[catStartIndex+len("[Category: ") : catEndIndex]
			}

			stories = append(stories, domain.UserStory{
				ID:          fmt.Sprintf("%s-%d", generateID(), lineNumber), 
				Description: description,
				Category:    category,
			})
		}
	}

	finalSummary := strings.TrimSpace(summaryBuilder.String())
	return finalSummary, stories, nil
}

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
		if _, err := writer.WriteString("\n"); err != nil { 
			return fmt.Errorf("error writing newline after summary section: %w", err)
		}
	}

	if len(stories) == 0 {
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
			categoryOrder = append(categoryOrder, catToUse) 
		}
		storiesByCategory[catToUse] = append(storiesByCategory[catToUse], story)
	}

	sort.Strings(categoryOrder)

	for _, categoryName := range categoryOrder {
		if _, err := writer.WriteString(fmt.Sprintf("**%s**\n", categoryName)); err != nil {
			return fmt.Errorf("error writing category header %s: %w", categoryName, err)
		}

		categoryStories := storiesByCategory[categoryName]
		for _, story := range categoryStories {
			catToPrintInTag := story.Category
			if catToPrintInTag == "" {
				catToPrintInTag = "Uncategorized"
			}
			line := fmt.Sprintf("- %s [Category: %s]\n", story.Description, catToPrintInTag)
			if _, err := writer.WriteString(line); err != nil {
				return fmt.Errorf("error writing story (ID: %s) to file: %w", story.ID, err)
			}
		}
		if _, err := writer.WriteString("\n"); err != nil { 
			return fmt.Errorf("error writing newline separator: %w", err)
		}
	}

	return writer.Flush()
}

func (s *UserStoryService) AddUserStory(description string) error {
	currentSummary, stories, err := s.ReadUserStoriesFromFile()
	if err != nil {
		return fmt.Errorf("could not read existing stories: %w", err)
	}

	newStory := domain.UserStory{
		ID:          generateID(),
		Description: description,
		Category:    "Uncategorized", 
	}

	llmInput := domain.LLMSimpleInput{
		SystemMessage: "Categorize the following user story. Only return the category name.",
		UserMessage:   newStory.Description,
		ModelType:     domain.ModelTypeSimple,
	}

	category, err := s.llmService.AskSimple(llmInput)

	if err != nil {
		return fmt.Errorf("could not categorize new story: %w", err)
	}
	category = strings.TrimSpace(category)
	if category == "" {
		category = "Uncategorized" 
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

func (s *UserStoryService) CategorizeAllStories() error {
	currentSummary, stories, err := s.ReadUserStoriesFromFile()
	if err != nil {
		return fmt.Errorf("could not read stories for categorization: %w", err)
	}

	if len(stories) == 0 {
		fmt.Println("No stories found in the file to categorize.")
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
			category = "Uncategorized" 
		}
		categorizedStories[i] = story
		categorizedStories[i].Category = category
	}

	sort.Slice(categorizedStories, func(i, j int) bool {
		return categorizedStories[i].Category < categorizedStories[j].Category
	})


	err = s.WriteUserStoriesToFile(currentSummary, categorizedStories) 
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

func (s *UserStoryService) SummarizeStories() error {
	_, stories, err := s.ReadUserStoriesFromFile()
	if err != nil {
		return fmt.Errorf("could not read stories for summarization: %w", err)
	}

	if len(stories) == 0 {
		fmt.Println("No stories to summarize.")
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

func (s *UserStoryService) ListUserStories() error {
	summary, stories, err := s.ReadUserStoriesFromFile()
	if err != nil {
		return fmt.Errorf("could not read stories for listing: %w", err)
	}

	if summary != "" {
		fmt.Println("# Summary")
		fmt.Println(summary)
		fmt.Println() 
	}

	if len(stories) == 0 {
		if summary == "" { 
			fmt.Println("No user stories found in the file.")
		} else {
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

type GeneratedStoriesResponse struct {
	NewUserStories []string `json:"new_user_stories" jsonschema_description:"A list of new user story descriptions."`
}

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
		ModelType:         domain.ModelTypeReasoningSimple, 
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
			Category:    "Uncategorized", 
		}

		categorizationInput := domain.LLMSimpleInput{
			SystemMessage: "Categorize the following user story. Only return the category name.",
			UserMessage:   newStory.Description,
			ModelType:     domain.ModelTypeSimple,
		}
		category, catErr := s.llmService.AskSimple(categorizationInput)
		if catErr != nil {
			fmt.Printf("Could not categorize new story \"%s\": %v. Assigning 'Uncategorized'.\n", newStory.Description, catErr)
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

	var storyDescriptions strings.Builder
	for _, story := range stories {
		storyDescriptions.WriteString(story.Description)
		storyDescriptions.WriteString("\n")
	}

	responseInterface := domain.GenerateSchema[CategoryResponse]()

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

	err = json.Unmarshal([]byte(categoriesResponse), &categoriesResponseStruct)
	if err != nil {
		fmt.Printf("Error unmarshalling categories response: %v\n", err)
		return nil
	}

	categories = categoriesResponseStruct.Categories

	return categories
}
