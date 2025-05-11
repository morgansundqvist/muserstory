package application

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/morgansundqvist/muserstory/internal/domain"
	"github.com/morgansundqvist/muserstory/internal/ports"
)

// GetProjectRemote fetches a project by ID from the remote API and prints its user stories.
func (s *UserStoryService) GetProjectRemote(id string) error {
	if id == "" {
		return fmt.Errorf("project id must be provided with --id flag")
	}
	apiHost := os.Getenv("API_HOST")
	if apiHost == "" {
		apiHost = "http://localhost:3000"
	}
	url := strings.TrimRight(apiHost, "/") + "/api/projects/" + id

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to GET project: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("failed to fetch project, status: %s", resp.Status)
	}

	var project domain.Project
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&project); err != nil {
		return fmt.Errorf("failed to decode project response: %w", err)
	}

	fmt.Printf("Project: %s (UUID: %s)\n", project.Name, project.ID)
	if len(project.UserStories) == 0 {
		fmt.Println("No user stories found for this project.")
		return nil
	}
	fmt.Println("User Stories:")
	for _, story := range project.UserStories {
		fmt.Printf("- %s [Category: %s]\n", story.Description, story.Category)
	}
	return nil
}

type UserStoryService struct {
	llmService ports.LLMService
	filePath   string
	fileReader ports.FileReader
}

func NewUserStoryService(
	llmService ports.LLMService, filePath string, fileReader ports.FileReader) *UserStoryService {
	return &UserStoryService{
		llmService: llmService,
		filePath:   filePath,
		fileReader: fileReader,
	}
}

func generateID() string {
	uuidID := uuid.NewString()
	return uuidID
}

func (s *UserStoryService) ReadUserStoriesFromFile() (*domain.MarkdownFile, error) {
	content, err := s.fileReader.ReadFileContent(s.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}
	markdownFile, err := domain.ParseMarkdownFileContent(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse markdown file content: %w", err)
	}
	return markdownFile, nil
}

func (s *UserStoryService) AddUserStory(description string) error {
	markdownFile, err := s.ReadUserStoriesFromFile()
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

	markdownFile.Stories = append(markdownFile.Stories, newStory)

	err = markdownFile.WriteToFile(s.filePath)
	if err != nil {
		return fmt.Errorf("could not write new story to file: %w", err)
	}
	fmt.Printf("User story added: \"%s\" [Category: %s]\n", newStory.Description, newStory.Category)
	return nil
}

func (s *UserStoryService) CategorizeAllStories() error {
	markdownFile, err := s.ReadUserStoriesFromFile()
	if err != nil {
		return fmt.Errorf("could not read stories for categorization: %w", err)
	}

	if len(markdownFile.Stories) == 0 {
		return nil
	}

	possibleCategories := s.GeneratePossibleCategories(markdownFile.Stories)

	possibleCategoriesString := strings.Join(possibleCategories, ", ")

	categorizedStories := make([]domain.UserStory, len(markdownFile.Stories))
	for i, story := range markdownFile.Stories {
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

	markdownFile.Stories = categorizedStories

	err = markdownFile.WriteToFile(s.filePath)
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
	markdownFile, err := s.ReadUserStoriesFromFile()
	if err != nil {
		return fmt.Errorf("could not read stories for summarization: %w", err)
	}

	if len(markdownFile.Stories) == 0 {
		fmt.Println("No stories to summarize.")
		return nil
	}

	var storyDescriptions strings.Builder
	for i, story := range markdownFile.Stories {
		storyDescriptions.WriteString(story.Description)
		if i < len(markdownFile.Stories)-1 {
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

	markdownFile.Summary = generatedSummary
	err = markdownFile.WriteToFile(s.filePath)
	if err != nil {
		return fmt.Errorf("could not write new summary and stories to file: %w", err)
	}

	fmt.Println("File has been updated with the new summary and existing stories.")
	return nil
}

func (s *UserStoryService) ListUserStories() error {
	markdownFile, err := s.ReadUserStoriesFromFile()
	if err != nil {
		return fmt.Errorf("could not read stories for listing: %w", err)
	}

	if len(markdownFile.Stories) == 0 {
		if markdownFile.Summary == "" {
			fmt.Println("No user stories found in the file.")
		} else {
			fmt.Println("No user stories found in the file (summary is present).")
		}
		return nil
	}

	// Group stories by category
	categoryMap := make(map[string][]domain.UserStory)
	var categories []string
	for _, story := range markdownFile.Stories {
		category := story.Category
		if _, exists := categoryMap[category]; !exists {
			categories = append(categories, category)
		}
		categoryMap[category] = append(categoryMap[category], story)
	}

	// Sort categories alphabetically for consistent output
	sort.Strings(categories)

	fmt.Println("User Stories:")
	for i, category := range categories {
		fmt.Printf("Category: %s\n", category)
		for _, story := range categoryMap[category] {
			fmt.Printf("- %s [UUID: %s]\n", story.Description, story.ID)
		}
		if i < len(categories)-1 {
			fmt.Println()
		}
	}
	return nil
}

type GeneratedStoriesResponse struct {
	NewUserStories []string `json:"new_user_stories" jsonschema_description:"A list of new user story descriptions."`
}

func (s *UserStoryService) GenerateNewStories(numStoriesToGenerate int) error {
	markdownFile, err := s.ReadUserStoriesFromFile()
	if err != nil {
		return fmt.Errorf("could not read existing stories: %w", err)
	}

	var existingStoryDescriptions strings.Builder
	if len(markdownFile.Stories) > 0 {
		existingStoryDescriptions.WriteString("Existing user stories for context:\n")
		for _, story := range markdownFile.Stories {
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

	allStories := markdownFile.Stories
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

	markdownFile.Stories = allStories

	err = markdownFile.WriteToFile(s.filePath)
	if err != nil {
		return fmt.Errorf("could not write new stories to file: %w", err)
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

func (s *UserStoryService) PushProject() error {
	markdownFile, err := s.ReadUserStoriesFromFile()
	if err != nil {
		return fmt.Errorf("could not read markdown file: %w", err)
	}

	// Ensure metadata map exists
	if markdownFile.Metadata == nil {
		markdownFile.Metadata = make(map[string]interface{})
	}

	projectID, _ := markdownFile.Metadata["project_id"].(string)
	projectName, _ := markdownFile.Metadata["project_name"].(string)

	// Prompt for project name and generate ID if missing
	if projectID == "" {
		if projectName == "" {
			fmt.Print("Enter project name: ")
			reader := bufio.NewReader(os.Stdin)
			nameInput, _ := reader.ReadString('\n')
			projectName = strings.TrimSpace(nameInput)
			if projectName == "" {
				return fmt.Errorf("project name cannot be empty")
			}
		}
		projectID = generateID()
	}

	// Build Project entity
	project := domain.Project{
		ID:          projectID,
		Name:        projectName,
		Summary:     markdownFile.Summary,
		UserStories: markdownFile.Stories,
	}

	// Build API URL
	apiHost := os.Getenv("API_HOST")
	if apiHost == "" {
		apiHost = "http://localhost:3000"
	}
	apiPath := "/api/projects"
	url := strings.TrimRight(apiHost, "/") + apiPath

	// Marshal project to JSON
	body, err := json.Marshal(project)
	if err != nil {
		return fmt.Errorf("failed to marshal project: %w", err)
	}

	// HTTP POST
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to POST project: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("failed to push project to remote, status: %s", resp.Status)
	}

	// Write project_id and project_name to metadata and save
	markdownFile.Metadata["project_id"] = projectID
	markdownFile.Metadata["project_name"] = projectName
	if err := markdownFile.WriteToFile(s.filePath); err != nil {
		return fmt.Errorf("could not update markdown file with project metadata: %w", err)
	}

	fmt.Printf("Project pushed and metadata updated in %s\n", s.filePath)
	return nil
}

// ListProjectsRemote fetches all projects from the remote API and prints their name and UUID.
func (s *UserStoryService) ListProjectsRemote() error {
	apiHost := os.Getenv("API_HOST")
	if apiHost == "" {
		apiHost = "http://localhost:3000"
	}
	url := strings.TrimRight(apiHost, "/") + "/api/projects"

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to GET projects: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("failed to fetch projects, status: %s", resp.Status)
	}

	var projects []domain.Project
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&projects); err != nil {
		return fmt.Errorf("failed to decode projects response: %w", err)
	}

	if len(projects) == 0 {
		fmt.Println("No remote projects found.")
		return nil
	}

	fmt.Println("Remote Projects:")
	for _, project := range projects {
		fmt.Printf("- %s (UUID: %s)\n", project.Name, project.ID)
	}
	return nil
}
